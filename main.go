package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/jetstack/spiffe-connector/internal/cmd"
)

func main() {
	app := &cli.App{
		Usage:     "SVID to external credential helper",
		ArgsUsage: "",
		Commands:  nil,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "tls-cert-file",
				Aliases:   []string{"cert"},
				Usage:     "Path to TLS serving cert file (PEM encoded)",
				EnvVars:   []string{"SPIFFE_CONNECTOR_TLS_CERT_FILE"},
				FilePath:  "",
				Required:  false,
				Hidden:    false,
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:      "tls-key-file",
				Aliases:   []string{"key"},
				Usage:     "Path to TLS serving key file (PEM encoded)",
				EnvVars:   []string{"SPIFFE_CONNECTOR_TLS_KEY_FILE"},
				FilePath:  "",
				Required:  false,
				Hidden:    false,
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:      "config-file",
				Aliases:   []string{"config"},
				Usage:     "Path to config file",
				EnvVars:   []string{"SPIFFE_CONNECTOR_CONFIG_FILE"},
				FilePath:  "",
				Required:  true,
				Hidden:    false,
				TakesFile: true,
			},
			&cli.BoolFlag{
				Name:        "use-self-signed-certs",
				Aliases:     nil,
				Usage:       "",
				EnvVars:     []string{"SPIFFE_CONNECTOR_USE_SELF_SIGNED_CERTS"},
				Required:    false,
				Value:       true,
				DefaultText: "",
				Destination: nil,
				HasBeenSet:  false,
			},
		},
		Action:                 cmd.Run,
		UseShortOptionHandling: false,
	}
	app.Run(os.Args)
}
