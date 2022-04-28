package main

import (
	"context"
	"fmt"
	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"os"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
)

func main() {
	svid, err := x509svid.Load("./svid_cert.pem", "./svid_key.pem")
	bundle, err := x509bundle.Load(spiffeid.RequireTrustDomainFromString("test.domain"), "./ca.pem")
	exitOnErr(err)

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"localhost:9090",
		grpc.WithTransportCredentials(
			grpccredentials.MTLSClientCredentials(svid, bundle, tlsconfig.AuthorizeAny()),
		),
	)
	exitOnErr(err)

	client := proto.NewSpiffeConnectorClient(conn)
	resp, err := client.GetCredentials(ctx, &emptypb.Empty{})
	exitOnErr(err)
	println(resp.Credentials)
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
