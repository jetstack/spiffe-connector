package main

import (
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Usage:                  "Example spiffe-connector workload app",
		Action:                 Run,
		UseShortOptionHandling: false,
	}
	app.Run(os.Args)
}
