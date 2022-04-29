package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/jetstack/spiffe-connector/internal/pkg/sidecar"
	"github.com/jetstack/spiffe-connector/types"
)

func Run(ctx *cli.Context) error {
	cfg := &types.SpiffeConfig{}
	if len(ctx.String("workload-api-socket")) > 0 {
		cfg.SVIDSources.WorkloadAPI = &types.WorkloadAPI{
			SocketPath: ctx.String("workload-api-socket"),
		}
	} else {
		cert, key := ctx.String("tls-cert-file"), ctx.String("tls-key-file")
		if len(cert) == 0 || len(key) == 0 {
			return cli.Exit(
				fmt.Sprintf("Either --workload-api-socket or both --tls-cert-file and --tls-key-file must be set"),
				1,
			)
		}
		ca := ctx.String("trusted-ca-file")
		if len(ca) == 0 {
			return cli.Exit(
				fmt.Sprintf("--trusted-ca-file is required"), 1,
			)
		}
		cfg.SVIDSources.Files = &types.Files{
			TrustDomainCA: ca,
			SVIDCert:      cert,
			SVIDKey:       key,
		}
	}

	// Set up X509 SVID Source
	x509SourceCtx, x509SourceCancel := context.WithCancel(ctx.Context)
	source, err := config.ConstructSpiffeConnectorSource(x509SourceCtx, x509SourceCancel, cfg)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Couldn't get SPIFFE ID from workload API or files (%s)", err.Error()), 1)
	}
	config.StoreCurrentSource(source)

	mgr := sidecar.CredentialManager{
		ServerSPIFFEID: ctx.String("server-spiffe-id"),
		ServerAddress:  ctx.String("server-address"),
	}
	if err := mgr.Run(ctx.Context); err != nil {
		return cli.Exit(fmt.Sprintf("%s", err.Error()), 1)
	}
	return nil
}
