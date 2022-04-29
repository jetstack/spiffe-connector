package server

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maxatome/go-testdeep/td"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/jetstack/spiffe-connector/internal/pkg/cryptoutil"
	"github.com/jetstack/spiffe-connector/internal/pkg/provider"
	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
	"github.com/jetstack/spiffe-connector/types"
)

func TestServer_GetCredentials(t *testing.T) {
	testCtx, testCtxCancel := context.WithCancel(context.Background())

	certs, err := cryptoutil.GenerateTestCerts("spiffe://example.com/server", "spiffe://example.com/client")
	require.NoError(t, err)

	caCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certs[0].Certificate[1],
	})
	serverLeafCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certs[0].Certificate[0],
	})
	serverx509Key, _ := x509.MarshalPKCS8PrivateKey(certs[0].PrivateKey)
	serverKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: serverx509Key,
	})
	clientLeafCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certs[1].Certificate[0],
	})
	clientx509Key, _ := x509.MarshalPKCS8PrivateKey(certs[1].PrivateKey)
	clientKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: clientx509Key,
	})

	serverConfigSource, err := config.ConstructSpiffeConnectorSource(testCtx, testCtxCancel, &types.SpiffeConfig{
		SVIDSources: types.SVIDSources{
			InMemory: &types.InMemory{TrustDomainCA: caCert, SVIDCert: serverLeafCert, SVIDKey: serverKey},
		},
	})
	require.NoError(t, err)

	clientConfigSource, err := config.ConstructSpiffeConnectorSource(testCtx, testCtxCancel, &types.SpiffeConfig{
		SVIDSources: types.SVIDSources{
			InMemory: &types.InMemory{TrustDomainCA: caCert, SVIDCert: clientLeafCert, SVIDKey: clientKey},
		},
	})
	require.NoError(t, err)

	googleSAKeyFileData := "ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsCiAgInByb2plY3RfaWQiOiAiMTIzNCIsCiAgInByaXZhdGVfa2V5X2lkIjogInh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHgiLAogICJwcml2YXRlX2tleSI6ICJ4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHgiLAogICJjbGllbnRfZW1haWwiOiAib2stc2FAMTIzNC5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbSIsCiAgImNsaWVudF9pZCI6ICJ4eHh4eHh4eHh4eHh4eHh4eHh4eHgiLAogICJhdXRoX3VyaSI6ICJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20vby9vYXV0aDIvYXV0aCIsCiAgInRva2VuX3VyaSI6ICJodHRwczovL29hdXRoMi5nb29nbGVhcGlzLmNvbS90b2tlbiIsCiAgImF1dGhfcHJvdmlkZXJfeDUwOV9jZXJ0X3VybCI6ICJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9vYXV0aDIvdjEvY2VydHMiLAogICJjbGllbnRfeDUwOV9jZXJ0X3VybCI6ICJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9yb2JvdC92MS9tZXRhZGF0YS94NTA5L29rLXNhJTQwMTIzNC5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbSIKfQo="
	googleJSONKeyFileData, _ := base64.StdEncoding.DecodeString(googleSAKeyFileData)
	googleValidResponse := fmt.Sprintf(`{
		"name": "projects/1234/serviceAccounts/ok-sa@1234.iam.gserviceaccount.com/keys/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			"privateKeyType": "TYPE_GOOGLE_CREDENTIALS_FILE",
			"privateKeyData": "%s",
			"validAfterTime": "2022-04-20T10:39:55Z",
			"validBeforeTime": "9999-12-31T23:59:59Z",
			"keyAlgorithm": "KEY_ALG_RSA_2048",
			"keyOrigin": "GOOGLE_PROVIDED",
			"keyType": "USER_MANAGED"
	}`, googleSAKeyFileData)
	testServers := []*httptest.Server{
		httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(googleValidResponse))
		})),
	}

	require.NoError(t, err)
	testCases := map[string]struct {
		ACLs                []types.ACL
		ExpectedCredentials []*proto.Credential
	}{
		"when there are no matching credentials for the client": {
			ACLs: []types.ACL{
				{
					MatchPrincipal: "spiffe://bar/foo",
					Credentials: []types.Credential{
						{
							Provider:        "google",
							ObjectReference: "sa@example.com",
						},
					},
				},
			},
			ExpectedCredentials: nil,
		},
		"when there is a matching google credential": {
			ACLs: []types.ACL{
				{
					MatchPrincipal: "spiffe://example.com/client",
					Credentials: []types.Credential{
						{
							Provider:        "GoogleIAMServiceAccountKeyProvider",
							ObjectReference: "sa@example.com",
						},
					},
				},
			},
			ExpectedCredentials: []*proto.Credential{
				{
					Files: []*proto.File{
						{
							Path:     "key.json",
							Mode:     0644,
							Contents: []byte(googleJSONKeyFileData),
						},
					},
				},
			},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			googleProvider, err := provider.NewGoogleIAMServiceAccountKeyProvider(context.Background(), provider.GoogleIAMServiceAccountKeyProviderOptions{
				Endpoint: testServers[0].URL,
			})
			require.NoError(t, err)

			// create the server
			lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 3000))
			require.NoError(t, err)
			s := grpc.NewServer(grpc.Creds(grpccredentials.MTLSServerCredentials(serverConfigSource, serverConfigSource, tlsconfig.AuthorizeAny())))
			ss := Server{
				ACLs: testCase.ACLs,
				Providers: map[string]provider.Provider{
					"GoogleIAMServiceAccountKeyProvider": &googleProvider,
				},
			}
			proto.RegisterSpiffeConnectorServer(s, &ss)
			go func() {
				s.Serve(lis)
			}()
			defer s.Stop()

			// create the connection and client
			var opts []grpc.DialOption
			opts = append(opts, grpc.WithTransportCredentials(grpccredentials.MTLSClientCredentials(clientConfigSource, clientConfigSource, tlsconfig.AuthorizeAny())))
			conn, err := grpc.Dial("localhost:3000", opts...)
			require.NoError(t, err)
			defer conn.Close()
			client := proto.NewSpiffeConnectorClient(conn)

			// make the request to get the matching credentials
			resp, err := client.GetCredentials(context.Background(), &emptypb.Empty{})
			require.NoError(t, err)

			td.Cmp(t, resp.Credentials, testCase.ExpectedCredentials)
		})
	}
}
