package sidecar

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sync/atomic"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
)

type CredentialManager struct {
	ServerSPIFFEID string
	ServerAddress  string

	client             proto.SpiffeConnectorClient
	currentCredentials atomic.Value // []*proto.Credential
	refresh            chan struct{}
}

func (c *CredentialManager) Run(ctx context.Context) error {
	var authorizer tlsconfig.Authorizer
	if len(c.ServerSPIFFEID) > 0 {
		id, err := spiffeid.FromString(c.ServerSPIFFEID)
		if err != nil {
			return fmt.Errorf("provided SPIFFE ID is invalid: %w", err)
		}
		authorizer = tlsconfig.AuthorizeID(id)
	} else {
		authorizer = tlsconfig.AuthorizeAny()
	}
	conn, err := grpc.DialContext(
		ctx,
		c.ServerAddress,
		grpc.WithTransportCredentials(
			grpccredentials.MTLSClientCredentials(config.CurrentSource, config.CurrentSource, authorizer),
		),
	)
	if err != nil {
		return fmt.Errorf("credentialmanager: while attempting to connect to server: %w", err)
	}
	c.client = proto.NewSpiffeConnectorClient(conn)
	err = c.refreshCredentials(ctx)
	if err != nil {
		return fmt.Errorf("couldn't retrieve credentials from %s: %w", c.ServerAddress, err)
	}
	c.scheduleNext()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.refresh:
			err := c.refreshCredentials(ctx)
			if err != nil {
				log.Printf("error retrieving credentials: %s", err.Error())
				go func() {
					time.Sleep(1 * time.Minute)
					c.refresh <- struct{}{}
				}()
			} else {
				c.scheduleNext()
			}
		}
	}
}

func (c *CredentialManager) refreshCredentials(ctx context.Context) error {
	connCtx, cancel := context.WithTimeout(ctx, time.Minute)
	creds, err := c.client.GetCredentials(connCtx, &emptypb.Empty{})
	cancel()
	if err != nil {
		return err
	}
	c.currentCredentials.Store(creds.GetCredentials())
	if err := c.applyCredentials(); err != nil {
		return err
	}
	return nil
}

func (c *CredentialManager) applyCredentials() error {
	creds := c.currentCredentials.Load().([]*proto.Credential)
	for _, cred := range creds {
		if cred == nil { // should never happen
			continue
		}
		for _, f := range cred.Files {
			if f == nil {
				continue
			}
			if err := os.WriteFile(f.Path, f.Contents, os.FileMode(f.Mode)); err != nil {
				return err
			}
		}
		for k, v := range cred.EnvVars {
			// TODO: This won't actually be useful as a sidecar. TODO: implement a container init / wrapper mode
			if err := os.Setenv(k, v); err != nil {
				return err
			}
		}
		// TODO: Username/Password/Token would only be useful speaking to the connector directly as an app. Implement as a library.
	}
	return nil
}

func (c *CredentialManager) scheduleNext() {
	creds := c.currentCredentials.Load().([]*proto.Credential)
	next := time.Now().Add(math.MaxInt)
	for _, cred := range creds {
		if cred == nil {
			continue
		}
		if cred.NotAfter.AsTime().Before(next) {
			next = cred.NotAfter.AsTime()
		}
	}
	go func(c *CredentialManager, at time.Time) {
		time.Sleep(time.Until(at) / 3 * 2)
		c.refresh <- struct{}{}
	}(c, next)
}
