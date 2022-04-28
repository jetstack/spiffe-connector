package main

import (
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Usage:     "SVID to external credential helper",
		ArgsUsage: "",
		Commands:  nil,
		Flags: []cli.Flag{
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
		},
		Action:                 Run,
		UseShortOptionHandling: false,
	}
	app.Run(os.Args)
}
