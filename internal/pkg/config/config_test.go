package config

import (
	"errors"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jetstack/spiffe-connector/types"
)

func TestLoadConfigFromFs(t *testing.T) {
	testCases := map[string]struct {
		InputFile      string
		ExpectedConfig *types.Config
		ExpectedError  error
	}{
		"valid config with single principal": {
			InputFile: `---
acls:
- match_principal: "spiffe://foo/bar/baz"
  credentials:
  - provider: "google"
    object_reference: "service-account@example.com"
  - provider: "aws"
    object_reference: "aws::arn:foo"
`,
			ExpectedConfig: &types.Config{
				ACLs: []types.ACL{
					{
						MatchPrincipal: "spiffe://foo/bar/baz",
						Credentials: []types.Credential{
							{
								Provider:        "google",
								ObjectReference: "service-account@example.com",
							},
							{
								Provider:        "aws",
								ObjectReference: "aws::arn:foo",
							},
						},
					},
				},
			},
		},
		"valid config with multiple principals": {
			InputFile: `---
acls:
- match_principal: "spiffe://foo/bar/baz"
  credentials:
  - provider: "google"
    object_reference: "service-account@example.com"
  - provider: "aws"
    object_reference: "aws::arn:foo"
- match_principal: "spiffe://foo/bar/baz/foo/bar"
  credentials:
  - provider: "aws"
    object_reference: "XXX"
`,
			ExpectedConfig: &types.Config{
				ACLs: []types.ACL{
					{
						MatchPrincipal: "spiffe://foo/bar/baz",
						Credentials: []types.Credential{
							{
								Provider:        "google",
								ObjectReference: "service-account@example.com",
							},
							{
								Provider:        "aws",
								ObjectReference: "aws::arn:foo",
							},
						},
					},
					{
						MatchPrincipal: "spiffe://foo/bar/baz/foo/bar",
						Credentials: []types.Credential{
							{
								Provider:        "aws",
								ObjectReference: "XXX",
							},
						},
					},
				},
			},
		},
		"invalid config with duplicated principals": {
			InputFile: `---
acls:
- match_principal: "spiffe://foo/bar/baz"
  credentials:
  - provider: "google"
    object_reference: "service-account@example.com"
- match_principal: "spiffe://foo/bar/baz"
  credentials:
  - provider: "aws"
    object_reference: "XXX"
`,
			ExpectedError: errors.New("config validation failed: duplicate principal matching rule spiffe://foo/bar/baz (seen 2 times)"),
		},
		"invalid config with duplicated providers": {
			InputFile: `---
acls:
- match_principal: "spiffe://foo/bar/baz"
  credentials:
  - provider: "google"
    object_reference: "service-account@example.com"
  - provider: "google"
    object_reference: "service-account@example.com"
`,
			ExpectedError: errors.New("config validation failed: principal \"spiffe://foo/bar/baz\" is invalid: duplicate provider \"google\" (seen 2 times)"),
		},
		"invalid config with bad ACL match_principals": {
			InputFile: `---
acls:
- match_principal: "missing/spiffe/prefix"
`,
			ExpectedError: errors.New("config validation failed: principal \"missing/spiffe/prefix\" is invalid: scheme is missing or invalid"),
		},
		"invalid config with non-SPIFFE principal ID": {
			InputFile: `---
acls:
- match_principal: "https://foo/bar/baz"
`,
			ExpectedError: errors.New("config validation failed: principal \"https://foo/bar/baz\" is invalid: scheme is missing or invalid"),
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			testFs := make(fstest.MapFS)
			testFs["example.yaml"] = &fstest.MapFile{
				Data:    []byte(tc.InputFile),
				Mode:    0600,
				ModTime: time.Now(),
			}

			config, err := LoadConfigFromFs(testFs, "example.yaml")

			if tc.ExpectedError != nil {
				assert.EqualError(t, err, tc.ExpectedError.Error(), "unexpected error message")
			}
			if tc.ExpectedConfig != nil {
				require.NoError(t, err)
				assert.Equal(t, tc.ExpectedConfig, config, "unexpected config data")
			}
		})
	}
}
