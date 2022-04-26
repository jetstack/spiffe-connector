package server

import (
	"context"
	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
)

type Server struct {
}

func (s *Server) Listen(ctx context.Context) {
	_ = grpccredentials.MTLSServerCredentials(config.CurrentSource, config.CurrentSource, tlsconfig.AuthorizeAny())
}
