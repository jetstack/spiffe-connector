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
	"time"

	"github.com/maxatome/go-testdeep/td"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jetstack/spiffe-connector/internal/pkg/config"
	"github.com/jetstack/spiffe-connector/internal/pkg/cryptoutil"
	"github.com/jetstack/spiffe-connector/internal/pkg/provider"
	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
	"github.com/jetstack/spiffe-connector/types"
)

func TestServer_GetCredentials(t *testing.T) {
	testCtx, testCtxCancel := context.WithCancel(context.Background())

	// create the client and server SVIDs for use in test cases, these enable identification of clients used to match
	// against credentials
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

	// create providers
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

	makeGoogleTestServer := func(t *testing.T, invocations *int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*invocations++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(googleValidResponse))
		}))
	}

	makeAWSTestServer := func(t *testing.T, invocations *int, lifetimes []time.Duration) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(lifetimes) < 1 {
				t.Fatal("lifetimes must be set for aws test server if called")
			}
			if *invocations > len(lifetimes) {
				t.Fatal("aws test server called more than expected")
			}

			*invocations++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`
<AssumeRoleResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <AssumeRoleResult>
    <AssumedRoleUser>
      <AssumedRoleId>XXXXXXXXXXXXXXXXXXXXX:spiffe-connector-...</AssumedRoleId>
      <Arn>arn:aws:sts::xxxxxxxxxxxx:assumed-role/XXXXXX/spiffe-connector-...</Arn>
    </AssumedRoleUser>
    <Credentials>
      <AccessKeyId>keyid</AccessKeyId>
      <SecretAccessKey>key</SecretAccessKey>
      <SessionToken>sessiontoken-%d</SessionToken>
      <Expiration>%s</Expiration>
    </Credentials>
  </AssumeRoleResult>
  <ResponseMetadata>
    <RequestId>9a5aaaed-abdc-4eaf-9e48-9ae4da8caba9</RequestId>
  </ResponseMetadata>
</AssumeRoleResponse>
`, *invocations, time.Now().UTC().Add(lifetimes[*invocations-1]).Format("2006-01-02T15:04:05Z"))))
		}))
	}

	require.NoError(t, err)
	testCases := map[string]struct {
		Invocations               int
		ACLs                      []types.ACL
		GoogleExpectedInvocations int
		AWSExpectedInvocations    int
		AWSCredentialLifetimes    []time.Duration
		ExpectedCredentials       []td.TestDeep
	}{
		"when there are no matching credentials, client calls once": {
			Invocations: 1,
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
			GoogleExpectedInvocations: 0,
			AWSExpectedInvocations:    0,
			ExpectedCredentials: []td.TestDeep{
				td.Slice([]*proto.Credential{}, td.ArrayEntries{}),
			},
		},
		"when there are matching credentials, client calls once": {
			Invocations: 1,
			ACLs: []types.ACL{
				{
					MatchPrincipal: "spiffe://example.com/client",
					Credentials: []types.Credential{
						{
							Provider:        "GoogleIAMServiceAccountKeyProvider",
							ObjectReference: "sa@example.com",
						},
						{
							Provider:        "AWSSTSAssumeRoleProvider",
							ObjectReference: "arn:aws:iam::xxxxxxxxxxxx:role/Role",
						},
					},
				},
			},
			GoogleExpectedInvocations: 1,
			AWSExpectedInvocations:    1,
			AWSCredentialLifetimes:    []time.Duration{time.Hour},
			ExpectedCredentials: []td.TestDeep{
				td.Slice([]*proto.Credential{}, td.ArrayEntries{
					0: &proto.Credential{
						Files: []*proto.File{
							{
								Path:     "key.json",
								Mode:     0644,
								Contents: []byte(googleJSONKeyFileData),
							},
						},
					},
					1: td.Struct(
						&proto.Credential{
							Files: []*proto.File{
								{
									Path: "~/.aws/credentials",
									Mode: 0644,
									Contents: []byte(`[default]
aws_access_key_id = keyid
aws_secret_access_key = key
aws_session_token = sessiontoken-1
`),
								},
							},
						},
						td.StructFields{
							"NotAfter": td.Code(func(tspb *timestamppb.Timestamp) bool {
								t := tspb.AsTime()
								return t.Before(time.Now().UTC().Add(time.Hour+5*time.Second)) &&
									t.After(time.Now().UTC().Add(time.Hour-5*time.Second))
							}),
						},
					),
				}),
			},
		},
		"when there are matching credentials, client calls twice": {
			Invocations: 2,
			ACLs: []types.ACL{
				{
					MatchPrincipal: "spiffe://example.com/client",
					Credentials: []types.Credential{
						{
							Provider:        "GoogleIAMServiceAccountKeyProvider",
							ObjectReference: "sa@example.com",
						},
						{
							Provider:        "AWSSTSAssumeRoleProvider",
							ObjectReference: "arn:aws:iam::xxxxxxxxxxxx:role/Role",
						},
					},
				},
			},
			GoogleExpectedInvocations: 1,
			AWSExpectedInvocations:    2,
			AWSCredentialLifetimes:    []time.Duration{time.Second, time.Hour},
			ExpectedCredentials: []td.TestDeep{
				td.Slice([]*proto.Credential{}, td.ArrayEntries{
					0: &proto.Credential{
						Files: []*proto.File{
							{
								Path:     "key.json",
								Mode:     0644,
								Contents: []byte(googleJSONKeyFileData),
							},
						},
					},
					1: td.Struct(
						&proto.Credential{
							Files: []*proto.File{
								{
									Path: "~/.aws/credentials",
									Mode: 0644,
									Contents: []byte(`[default]
aws_access_key_id = keyid
aws_secret_access_key = key
aws_session_token = sessiontoken-1
`),
								},
							},
						},
						td.StructFields{
							"NotAfter": td.Code(func(tspb *timestamppb.Timestamp) bool {
								t := tspb.AsTime()
								return t.Before(time.Now().UTC().Add(2*time.Second)) &&
									t.After(time.Now().UTC().Add(-2*time.Second))
							}),
						},
					),
				}),
				td.Slice([]*proto.Credential{}, td.ArrayEntries{
					0: &proto.Credential{
						Files: []*proto.File{
							{
								Path:     "key.json",
								Mode:     0644,
								Contents: []byte(googleJSONKeyFileData),
							},
						},
					},
					1: td.Struct(
						&proto.Credential{
							Files: []*proto.File{
								{
									Path: "~/.aws/credentials",
									Mode: 0644,
									Contents: []byte(`[default]
aws_access_key_id = keyid
aws_secret_access_key = key
aws_session_token = sessiontoken-2
`),
								},
							},
						},
						td.StructFields{
							"NotAfter": td.Code(func(tspb *timestamppb.Timestamp) bool {
								t := tspb.AsTime()
								return t.Before(time.Now().UTC().Add(time.Hour+2*time.Second)) &&
									t.After(time.Now().UTC().Add(time.Hour-2*time.Second))
							}),
						},
					),
				}),
			},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			var googleInvocations int
			googleTestServer := makeGoogleTestServer(t, &googleInvocations)
			googleProvider, err := provider.NewGoogleIAMServiceAccountKeyProvider(context.Background(), provider.GoogleIAMServiceAccountKeyProviderOptions{
				Endpoint: googleTestServer.URL,
			})
			require.NoError(t, err)

			var awsInvocations int
			awsTestServer := makeAWSTestServer(t, &awsInvocations, testCase.AWSCredentialLifetimes)
			awsProvider, err := provider.NewAWSSTSAssumeRoleProvider(context.Background(), provider.AWSSTSAssumeRoleProviderOptions{
				Endpoint: awsTestServer.URL,
			})
			require.NoError(t, err)

			// create the server
			lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 3000))
			require.NoError(t, err)
			s := grpc.NewServer(grpc.Creds(grpccredentials.MTLSServerCredentials(serverConfigSource, serverConfigSource, tlsconfig.AuthorizeAny())))
			ss := Server{
				ACLs: testCase.ACLs,
				Providers: map[string]provider.Provider{
					"AWSSTSAssumeRoleProvider":           &awsProvider,
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

			if testCase.Invocations != len(testCase.ExpectedCredentials) {
				t.Fatal("Invocations must match the number of expected credential sets")
			}

			for i := 0; i < testCase.Invocations; i++ {
				t.Run(fmt.Sprintf("client-call-%d", i+1), func(t *testing.T) {
					// make the request to get the matching credentials
					resp, err := client.GetCredentials(context.Background(), &emptypb.Empty{})
					require.NoError(t, err)

					td.Cmp(t, resp.Credentials, testCase.ExpectedCredentials[i])
				})
			}

			assert.Equal(t, testCase.AWSExpectedInvocations, awsInvocations, "unexpected number of AWS provider invocations")
			assert.Equal(t, testCase.GoogleExpectedInvocations, googleInvocations, "unexpected number of google provider invocations")
		})
	}
}
