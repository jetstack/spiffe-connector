package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
)

func TestGoogleIAMServiceAccountKeyProvider_Name(t *testing.T) {
	// create a new test server to use in configuration, it will not be used in this test
	testIAMServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("test server should not have been called")
	}))

	// create a new provider backed by our test server
	p, err := NewGoogleIAMServiceAccountKeyProvider(context.Background(), GoogleIAMServiceAccountKeyProviderOptions{
		Endpoint: testIAMServer.URL,
	})
	require.NoError(t, err)

	assert.Equal(t, "GoogleIAMServiceAccountKeyProvider", p.Name())
}

func TestGoogleIAMServiceAccountKeyProvider_Ping(t *testing.T) {
	// create a new test server to ping
	testIAMServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not have been invoked during ping")
	}))

	// create a new provider backed by our test server
	p, err := NewGoogleIAMServiceAccountKeyProvider(context.Background(), GoogleIAMServiceAccountKeyProviderOptions{
		Endpoint: testIAMServer.URL,
	})
	require.NoError(t, err)

	// when the server is up the ping should complete ok
	err = p.Ping()
	require.NoError(t, err)

	// if the server is unavailable, then we expect an error
	testIAMServer.Close()
	err = p.Ping()

	assert.ErrorContains(t, err, "provider ping failed")
}

func TestGoogleIAMServiceAccountKeyProvider_GetCredential(t *testing.T) {
	validbase64KeyData := "ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsCiAgInByb2plY3RfaWQiOiAiMTIzNCIsCiAgInByaXZhdGVfa2V5X2lkIjogInh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHgiLAogICJwcml2YXRlX2tleSI6ICJ4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHgiLAogICJjbGllbnRfZW1haWwiOiAib2stc2FAMTIzNC5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbSIsCiAgImNsaWVudF9pZCI6ICJ4eHh4eHh4eHh4eHh4eHh4eHh4eHgiLAogICJhdXRoX3VyaSI6ICJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20vby9vYXV0aDIvYXV0aCIsCiAgInRva2VuX3VyaSI6ICJodHRwczovL29hdXRoMi5nb29nbGVhcGlzLmNvbS90b2tlbiIsCiAgImF1dGhfcHJvdmlkZXJfeDUwOV9jZXJ0X3VybCI6ICJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9vYXV0aDIvdjEvY2VydHMiLAogICJjbGllbnRfeDUwOV9jZXJ0X3VybCI6ICJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9yb2JvdC92MS9tZXRhZGF0YS94NTA5L29rLXNhJTQwMTIzNC5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbSIKfQo="
	validJSONKeyFileData, _ := base64.StdEncoding.DecodeString(validbase64KeyData)
	validResponse := fmt.Sprintf(`{
  "name": "projects/1234/serviceAccounts/ok-sa@1234.iam.gserviceaccount.com/keys/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "privateKeyType": "TYPE_GOOGLE_CREDENTIALS_FILE",
  "privateKeyData": "%s",
  "validAfterTime": "2022-04-20T10:39:55Z",
  "validBeforeTime": "9999-12-31T23:59:59Z",
  "keyAlgorithm": "KEY_ALG_RSA_2048",
  "keyOrigin": "GOOGLE_PROVIDED",
  "keyType": "USER_MANAGED"
}`, validbase64KeyData)
	testCases := map[string]struct {
		objectReference      string
		expectedError        error
		expectedCredential   *proto.Credential
		testServer           func(*int) *httptest.Server
		expectedRequestCount int
	}{
		"when object does not exist": {
			objectReference: "missing-sa@1234.iam.gserviceaccount.com",
			expectedError:   errors.New("failed to create service account key: googleapi: Error 404: Unknown service account"),
			testServer: func(count *int) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					*count++
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`{
  "error": {
    "code": 404,
    "message": "Unknown service account",
    "status": "NOT_FOUND"
  }
}`))
				}))
			},
			expectedRequestCount: 1,
		},
		"when permission denied": {
			objectReference: "denied-sa@1234.iam.gserviceaccount.com",
			expectedError:   errors.New("failed to create service account key: googleapi: Error 403: Missing key or some authorization error"),
			testServer: func(count *int) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					*count++
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{
  "error": {
    "code": 403,
    "message": "Missing key or some authorization error",
    "status": "PERMISSION_DENIED"
  }
}`))
				}))
			},
			expectedRequestCount: 1,
		},
		"key created ok": {
			objectReference: "ok-sa@1234.iam.gserviceaccount.com",
			testServer: func(count *int) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					*count++
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(validResponse))
				}))
			},
			expectedRequestCount: 1,
			expectedCredential: &proto.Credential{
				Files: []*proto.File{
					{
						Path:     "key.json",
						Mode:     0644,
						Contents: []byte(validJSONKeyFileData),
					},
				},
			},
		},
	}

	for testName, testCase := range testCases {
		var count int
		testServer := testCase.testServer(&count)

		// create a new provider backed by our test server
		p, err := NewGoogleIAMServiceAccountKeyProvider(context.Background(), GoogleIAMServiceAccountKeyProviderOptions{
			Endpoint: testServer.URL,
		})
		require.NoError(t, err)

		t.Run(testName, func(t *testing.T) {
			cred, err := p.GetCredential(testCase.objectReference)
			if testCase.expectedError != nil {
				assert.EqualError(t, err, testCase.expectedError.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedCredential, cred)
				assert.Equal(t, testCase.expectedRequestCount, count, "unexpected number of requests made to test instance")
			}
		})
	}
}
