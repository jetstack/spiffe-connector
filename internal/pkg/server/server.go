package server

import (
	"context"
	"google.golang.org/grpc"
	"net"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"

	"github.com/jetstack/spiffe-connector/internal/pkg/config"
)

type Server struct {
}

func (s *Server) Start(ctx context.Context) {

	server := grpc.NewServer(grpc.Creds(grpccredentials.MTLSServerCredentials(config.CurrentSource, config.CurrentSource, tlsconfig.AuthorizeAny())))

	listener, err := net.Listen("tcp", "[::]:9090")
	if err != nil {
		panic("fail")
	}

	if err := server.Serve(listener); err != nil {
		panic(err)
	}
}
