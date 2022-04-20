package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestACLValidate(t *testing.T) {
	testCases := map[string]struct {
		ACL            ACL
		ExpectedErrors []error
	}{
		"with valid match principal": {
			ACL: ACL{
				MatchPrincipal: "spiffe://bar/foo",
			},
			ExpectedErrors: []error{},
		},
		"with match principal with non-trailing wildcard": {
			ACL: ACL{
				MatchPrincipal: "spiffe://bar/*/foo",
			},
			ExpectedErrors: []error{
				errors.New("cannot non-suffix wildcard character \"*\""),
			},
		},
		"with match principal with wildcard": {
			ACL: ACL{
				MatchPrincipal: "spiffe://bar/foo/*",
			},
			ExpectedErrors: []error{},
		},
		"with match principal with trailing /": {
			ACL: ACL{
				MatchPrincipal: "spiffe://bar/foo/",
			},
			ExpectedErrors: []error{
				errors.New("cannot have trailing \"/\""),
			},
		},
		"with match principal with no SPIFFE scheme": {
			ACL: ACL{
				MatchPrincipal: "bar/foo",
			},
			ExpectedErrors: []error{
				errors.New("must start with \"spiffe://\""),
			},
		},
		"with match principal invalid URL chars": {
			ACL: ACL{
				MatchPrincipal: "spiffe://🌐/foo",
			},
			ExpectedErrors: []error{
				errors.New("parsed URI did not match: \"spiffe://%F0%9F%8C%90/foo\""),
			},
		},
		"with match principal with query": {
			ACL: ACL{
				MatchPrincipal: "spiffe://bar/foo?=baz",
			},
			ExpectedErrors: []error{
				errors.New("URI must have blank query, has: \"=baz\""),
			},
		},
		"with duplicated providers": {
			ACL: ACL{
				MatchPrincipal: "spiffe://bar/things",
				Credentials: []Credential{
					{Provider: "google", ObjectReference: "foo/bar"},
					{Provider: "google", ObjectReference: "foo/baz"},
				},
			},
			ExpectedErrors: []error{
				errors.New("duplicate provider \"google\" (seen 2 times)"),
			},
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			errs := tc.ACL.Validate()
			assert.ElementsMatch(t, errs, tc.ExpectedErrors)
		})
	}
}
