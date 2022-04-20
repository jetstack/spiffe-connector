package cmd

import (
	"crypto/tls"
	"fmt"
	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/jetstack/spiffe-connector/internal/pkg/cryptoutil"
	"github.com/urfave/cli/v2"
	"net"
	"net/http"
)

func Run(ctx *cli.Context) error {
	// TODO: Use config
	_, err := config.LoadConfigFromFs(realFS{}, ctx.String("config-file"))
	if err != nil {
		return cli.Exit(fmt.Sprintf("Couldn't load config file %s (%s)", ctx.String("config-file"), err.Error()), 1)
	}
	var listenerConfig *tls.Config
	if ctx.Bool("use-self-signed-certs") {
		cert, err := cryptoutil.SelfSignedServingCert()
		if err != nil {
			return cli.Exit(fmt.Sprintf("Couldn't generate self-signed cert (%s)", err.Error()), 1)
		}
		listenerConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAnyClientCert,
		}
	}

	l, err := tls.Listen("tcp", "[::]:4040", listenerConfig)

	// TODO: Add GRPC API
	httpErrors := make(chan error)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Couldn't create listener (%s)", err.Error()), 1)
	}
	go func(l net.Listener, httpError chan<- error) {
		fmt.Fprintf(ctx.App.Writer, "serving on %s", l.Addr().String())
		err := http.Serve(l, http.NotFoundHandler())
		httpError <- err
	}(l, httpErrors)

	select {
	case e := <-httpErrors:
		return cli.Exit(fmt.Sprintf("HTTP Error (%s)", e.Error()), 1)
	}
}
