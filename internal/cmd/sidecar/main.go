package main

import (
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Usage:     "SVID to external credential sidecar",
		ArgsUsage: "",
		Commands:  nil,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "server-address",
				Aliases:  []string{"s"},
				Usage:    "address / port to connect to the SPIFFE connector server",
				EnvVars:  []string{"SPIFFE_CONNECTOR_SERVER_ADDRESS"},
				Required: true,
				Hidden:   false,
				Value:    "localhost:9090",
			},
			&cli.StringFlag{
				Name:     "server-spiffe-id",
				Aliases:  []string{"sid"},
				Usage:    "Expected SPIFFE ID of the SPIFFE connector server",
				EnvVars:  []string{"SPIFFE_CONNECTOR_SERVER_SPIFFE_ID"},
				Required: false,
				Hidden:   false,
			},
			&cli.StringFlag{
				Name:      "workload-api-socket",
				Aliases:   []string{"w"},
				Usage:     "Path to SPIFFE workload API socket",
				EnvVars:   []string{"SPIFFE_CONNECTOR_WORKLOAD_API_SOCKET"},
				Required:  false,
				Hidden:    false,
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:      "tls-cert-file",
				Aliases:   []string{"cert"},
				Usage:     "Path to X509 SVID cert file",
				EnvVars:   []string{"SPIFFE_CONNECTOR_TLS_CERT_FILE"},
				Required:  false,
				Hidden:    false,
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:      "tls-key-file",
				Aliases:   []string{"key"},
				Usage:     "Path to X509 SVID private key file",
				EnvVars:   []string{"SPIFFE_CONNECTOR_TLS_KEY_FILE"},
				Required:  false,
				Hidden:    false,
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:      "trusted-ca-file",
				Aliases:   []string{"key"},
				Usage:     "Path to CAs that are trusted to sign SVIDs",
				EnvVars:   []string{"SPIFFE_CONNECTOR_TRUSTED_CA_FILE"},
				Required:  false,
				Hidden:    false,
				TakesFile: true,
			},
		},
		Action:                 Run,
		UseShortOptionHandling: false,
	}
	app.Run(os.Args)
}
