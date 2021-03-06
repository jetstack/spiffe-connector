package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/jetstack/spiffe-connector/internal/pkg/principal"
	"github.com/jetstack/spiffe-connector/internal/pkg/provider"
	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
	"github.com/jetstack/spiffe-connector/types"
)

type Server struct {
	// ACLs is a mapping for which spiffe IDs the server will match to credentials from providers
	ACLs []types.ACL

	// Providers is a list of the credential providers available to get credentials
	Providers map[string]provider.Provider

	credentialStore map[string]proto.Credential

	proto.UnimplementedSpiffeConnectorServer
}

func (s *Server) GetCredentials(ctx context.Context, empty *emptypb.Empty) (*proto.GetCredentialsResponse, error) {
	resp := &proto.GetCredentialsResponse{}

	// Get the connecting SPIFFE ID
	clientSVID, hasSVID := grpccredentials.PeerIDFromContext(ctx)
	if !hasSVID {
		return resp, errors.New("no SVID provided")
	}
	log.Printf("Obtaining credentials for %s\n", clientSVID.String())

	// find any ACL matches for the caller. If there are no matches, an empty list of credentials will be returned
	acl, err := principal.MatchingACL(s.ACLs, clientSVID.String())
	if err != nil {
		err := fmt.Errorf("failed to determine matching ACLs: %s", err)
		log.Println(err)
		return nil, err
	}
	if acl == nil {
		return resp, nil
	}

	for _, aclCred := range acl.Credentials {
		// if the config references a provider not initialized for the server, then we error out. This is most likely
		// invalid config
		p, ok := s.Providers[aclCred.Provider]
		if !ok {
			err := fmt.Errorf("server is not configured with %q provider", aclCred.Provider)
			log.Println(err)
			return nil, err
		}

		fmt.Println(aclCred.Key())
		existingCredential, ok := s.credentialStore[aclCred.Key()]
		if ok {
			// TODO make this expiry logic based on the lifetime of the credential?
			if existingCredential.NotAfter.AsTime().After(time.Now().UTC().Add(5 * time.Minute)) {
				resp.Credentials = append(resp.Credentials, &existingCredential)
				continue
			}
		}

		credential, err := p.GetCredential(aclCred.ObjectReference)
		if err != nil {
			err := fmt.Errorf("failed to get credential %q from %q provider: %w", aclCred.ObjectReference, aclCred.Provider, err)
			log.Println(err)
			return nil, err
		}

		if s.credentialStore == nil {
			s.credentialStore = make(map[string]proto.Credential)
		}
		s.credentialStore[aclCred.Key()] = *credential

		resp.Credentials = append(resp.Credentials, credential)
	}

	return resp, nil
}

func (s *Server) Start(ctx context.Context) {
	server := grpc.NewServer(grpc.Creds(grpccredentials.MTLSServerCredentials(config.CurrentSource, config.CurrentSource, tlsconfig.AuthorizeAny())))
	proto.RegisterSpiffeConnectorServer(server, s)
	listener, err := net.Listen("tcp", "[::]:9090")
	if err != nil {
		panic("fail")
	}

	if err := server.Serve(listener); err != nil {
		panic(err)
	}
}
