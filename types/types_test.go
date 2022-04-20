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
				errors.New("path segment characters are limited to letters, numbers, dots, dashes, and underscores"),
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
				errors.New("path cannot have a trailing slash"),
			},
		},
		"with match principal with no SPIFFE scheme": {
			ACL: ACL{
				MatchPrincipal: "bar/foo",
			},
			ExpectedErrors: []error{
				errors.New("scheme is missing or invalid"),
			},
		},
		"with match principal invalid path chars": {
			ACL: ACL{
				MatchPrincipal: "spiffe://td/🌐/foo",
			},
			ExpectedErrors: []error{
				errors.New("path segment characters are limited to letters, numbers, dots, dashes, and underscores"),
			},
		},
		"with match principal invalid trust domain chars": {
			ACL: ACL{
				MatchPrincipal: "spiffe://🌐/foo",
			},
			ExpectedErrors: []error{
				errors.New("trust domain characters are limited to lowercase letters, numbers, dots, dashes, and underscores"),
			},
		},
		"with match principal with query": {
			ACL: ACL{
				MatchPrincipal: "spiffe://bar/foo?=baz",
			},
			ExpectedErrors: []error{
				errors.New("path segment characters are limited to letters, numbers, dots, dashes, and underscores"),
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
		`malformed spiffe ID with "//"`: {
			ACL: ACL{
				MatchPrincipal: "spiffe://bar/things//baz",
				Credentials: []Credential{
					{Provider: "google", ObjectReference: "foo/bar"},
				},
			},
			ExpectedErrors: []error{
				errors.New("path cannot contain empty segments"),
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
