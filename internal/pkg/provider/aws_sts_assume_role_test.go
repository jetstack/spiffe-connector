package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maxatome/go-testdeep/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSSTSAssumeRoleProvider_Name(t *testing.T) {
	// create a new test server to use in configuration, it will not be used in this test
	testIAMServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("test server should not have been called")
	}))

	// create a new provider backed by our test server
	p, err := NewAWSSTSAssumeRoleProvider(context.Background(), AWSSTSAssumeRoleProviderOptions{
		Endpoint: testIAMServer.URL,
	})
	require.NoError(t, err)

	assert.Equal(t, "AWSSTSAssumeRoleProvider", p.Name())
}

func TestAWSSTSAssumeRoleProvider_Ping(t *testing.T) {
	// create a new test server to ping
	testIAMServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not have been invoked during ping")
	}))

	// create a new provider backed by our test server
	p, err := NewAWSSTSAssumeRoleProvider(context.Background(), AWSSTSAssumeRoleProviderOptions{
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

func TestAWSSTSAssumeRoleProvider_GetCredential(t *testing.T) {
	testCases := map[string]struct {
		objectReference      string
		expectedError        error
		expectedCredential   td.TestDeep
		testServer           func(*int) *httptest.Server
		expectedRequestCount int
	}{
		"when no permission to assume role": {
			objectReference: "arn:aws:iam::xxxxxxxxxxxx:role/MissingRole",
			expectedError:   errors.New("failed to get temporary credentials from STS: AccessDenied: User: test is not authorized to perform: sts:AssumeRole on resource: arn:aws:iam::xxxxxxxxxxxx:role/MissingRole"),
			testServer: func(count *int) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					*count++
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`
<ErrorResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <Error>
    <Type>Sender</Type>
    <Code>AccessDenied</Code>
    <Message>User: test is not authorized to perform: sts:AssumeRole on resource: arn:aws:iam::xxxxxxxxxxxx:role/MissingRole</Message>
  </Error>
  <RequestId>9a5aaaed-abdc-4eaf-9e48-9ae4da8caba9</RequestId>
</ErrorResponse>
`))
				}))
			},
			expectedRequestCount: 1,
		},
		"successful example": {
			objectReference: "arn:aws:iam::xxxxxxxxxxxx:role/Role",
			testServer: func(count *int) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					*count++
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
      <SessionToken>sessiontoken</SessionToken>
      <Expiration>%s</Expiration>
    </Credentials>
  </AssumeRoleResult>
  <ResponseMetadata>
    <RequestId>9a5aaaed-abdc-4eaf-9e48-9ae4da8caba9</RequestId>
  </ResponseMetadata>
</AssumeRoleResponse>
`, time.Now().UTC().Add(time.Hour).Format("2006-01-02T15:04:05Z"))))
				}))
			},
			expectedRequestCount: 1,
			expectedCredential: td.Struct(
				Credential{
					Files: []CredentialFile{
						{
							Path: "~/.aws/credentials",
							Mode: 0644,
							Contents: []byte(`[default]
aws_access_key_id = keyid
aws_secret_access_key = key
aws_session_token = sessiontoken
`),
						},
					},
				},
				td.StructFields{
					"NotAfter": td.Between(time.Now().UTC().Add(time.Hour-5*time.Second), time.Now().UTC().Add(time.Hour+5*time.Second)),
				},
			),
		},
	}

	for testName, testCase := range testCases {
		var count int
		testServer := testCase.testServer(&count)

		// create a new provider backed by our test server
		p, err := NewAWSSTSAssumeRoleProvider(context.Background(), AWSSTSAssumeRoleProviderOptions{
			Endpoint: testServer.URL,
		})
		require.NoError(t, err)

		t.Run(testName, func(t *testing.T) {
			cred, err := p.GetCredential(testCase.objectReference)
			if testCase.expectedError != nil {
				assert.EqualError(t, err, testCase.expectedError.Error())
			} else {
				require.NoError(t, err)
				td.Cmp(t, cred, testCase.expectedCredential)
				assert.Equal(t, testCase.expectedRequestCount, count, "unexpected number of requests made to test instance")
			}
		})
	}
}
