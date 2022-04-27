package server

import (
	"context"
	"errors"
	"log"
	"net"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
)

type Server struct {
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
