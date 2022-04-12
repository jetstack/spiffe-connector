package principal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jetstack/spiffe-connector/types"
)

func TestMatchingACLs(t *testing.T) {
	testCases := map[string]struct {
		Principal   string
		ACLs        []types.ACL
		MatchingACL *types.ACL
	}{
		"principal matches exactly": {
			Principal: "spiffe://foo/bar",
			ACLs: []types.ACL{
				{
					MatchPrincipal: "spiffe://bar/foo",
				},
				{
					MatchPrincipal: "spiffe://foo/*",
				},
				{
					MatchPrincipal: "spiffe://foo/bar",
				},
			},
			MatchingACL: &types.ACL{
				MatchPrincipal: "spiffe://foo/bar",
			},
		},
		"principal matches glob, still returns exact match": {
			Principal: "spiffe://foo/bar",
			ACLs: []types.ACL{
				{
					MatchPrincipal: "spiffe://bar/foo",
				},
				{
					MatchPrincipal: "spiffe://foo/*",
				},
				{
					MatchPrincipal: "spiffe://foo/bar",
				},
			},
			MatchingACL: &types.ACL{
				MatchPrincipal: "spiffe://foo/bar",
			},
		},
		"principal only matches glob": {
			Principal: "spiffe://foo/bar",
			ACLs: []types.ACL{
				{
					MatchPrincipal: "spiffe://bar/foo",
				},
				{
					MatchPrincipal: "spiffe://foo/*",
				},
			},
			MatchingACL: &types.ACL{
				MatchPrincipal: "spiffe://foo/*",
			},
		},
		"result is most specific glob": {
			Principal: "spiffe://foo/bar",
			ACLs: []types.ACL{
				{
					MatchPrincipal: "spiffe://*/*",
				},
				{
					MatchPrincipal: "spiffe://foo/*",
				},
				{
					MatchPrincipal: "spiffe://foo/*/*",
				},
			},
			MatchingACL: &types.ACL{
				MatchPrincipal: "spiffe://foo/*",
			},
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			fmt.Println(testName)
			found, result, err := MatchingACL(tc.ACLs, tc.Principal)
			require.NoError(t, err)
			if !found {
				t.Fatal("expected to find matching ACL, did not")
			}
			assert.Equal(t, tc.MatchingACL, result, "unexpected result")
		})
	}
}
