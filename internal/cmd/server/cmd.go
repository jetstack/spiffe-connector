package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/jetstack/spiffe-connector/internal/pkg/server"
)

func Run(ctx *cli.Context) error {
	// Load config
	cfg, err := config.ReadConfigFromFS(realFS{}, ctx.String("config-file"))
	if err != nil {
		return cli.Exit(fmt.Sprintf("Couldn't load config file %s (%s)", ctx.String("config-file"), err.Error()), 1)
	}
	config.StoreConfig(cfg)
	fmt.Printf("Loaded config %s\n", ctx.String("config-file"))

	// Set up X509 SVID Source
	x509SourceCtx, x509SourceCancel := context.WithCancel(ctx.Context)
	source, err := config.ConstructSpiffeConnectorSource(x509SourceCtx, x509SourceCancel, cfg.SPIFFE)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Couldn't get SPIFFE ID from workload API or files (%s)", err.Error()), 1)
	}
	config.StoreCurrentSource(source)

	// Start watching the config for reloads
	_, err = config.NewWatcher(ctx.Context, ctx.String("config-file"),
		func() error {
			if err := config.ReadAndStoreConfig(realFS{}, ctx.String("config-file")); err != nil {
				return err
			}
			cfg := config.GetCurrentConfig()
			oldSource := config.GetCurrentSource()
			newSourceCtx, newSourceCancel := context.WithCancel(ctx.Context)
			newSource, err := config.ConstructSpiffeConnectorSource(newSourceCtx, newSourceCancel, cfg.SPIFFE)
			if err != nil {
				newSourceCancel()
				return err
			}
			oldSource.Cancel()
			config.StoreCurrentSource(newSource)
			return nil
		},
	)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Couldn't set up config reloader (%s)", err.Error()), 1)
	}

	s := &server.Server{}
	s.Start(ctx.Context)
	return nil
}
