// Package types contains the config file structs
package types

import (
	"fmt"
	"net/url"
	"strings"
)

// ACL is a mapping between a given principal and the credentials for services it will gain access to.
type ACL struct {
	MatchPrincipal string       `yaml:"match_principal"`
	Credentials    []Credential `yaml:"credentials"`
}

func (a *ACL) Validate() []error {
	var errors []error

	nonWildCardMatchPrincipal := strings.TrimSuffix(a.MatchPrincipal, "/*")

	if strings.Contains(nonWildCardMatchPrincipal, "*") {
		errors = append(errors, fmt.Errorf(`cannot non-suffix wildcard character "*"`))
	}

	if !strings.HasPrefix(a.MatchPrincipal, "spiffe://") {
		errors = append(errors, fmt.Errorf(`must start with "spiffe://"`))
	} else {
		if strings.Contains(strings.TrimPrefix(a.MatchPrincipal, "spiffe://"), "//") {
			errors = append(errors, fmt.Errorf(`cannot have double "//"`))
		}
	}

	if strings.HasSuffix(a.MatchPrincipal, "/") {
		errors = append(errors, fmt.Errorf(`cannot have trailing "/"`))
	}

	parsedURL, err := url.Parse(nonWildCardMatchPrincipal)
	if err != nil {
		errors = append(errors, err)
	}
	if parsedURL.String() != nonWildCardMatchPrincipal {
		errors = append(errors, fmt.Errorf("parsed URI did not match: %q", parsedURL.String()))
	}
	if parsedURL.RawQuery != "" {
		errors = append(errors, fmt.Errorf("URI must have blank query, has: %q", parsedURL.RawQuery))
	}

	seenProviders := make(map[string]int)
	for _, provider := range a.Credentials {
		if _, found := seenProviders[provider.Provider]; !found {
			seenProviders[provider.Provider] = 1
		} else {
			seenProviders[provider.Provider]++
		}
	}
	for provider, count := range seenProviders {
		if count > 1 {
			errors = append(errors, fmt.Errorf("duplicate provider %q (seen %d times)", provider, count))
		}
	}

	return errors
}

// Credential represents any remote credential that the connector can give out.
type Credential struct {
	Provider        string `yaml:"provider"`
	ObjectReference string `yaml:"object_reference"`
}

// Config represents the config file that will be loaded from disk, or some other mechanism.
type Config struct {
	ACLs []ACL `yaml:"acls"`
}

func (c *Config) Validate() []error {
	var errors []error

	// Validate principals are not duplicated
	seenPrincipals := make(map[string]int)
	for _, acl := range c.ACLs {
		if _, found := seenPrincipals[acl.MatchPrincipal]; !found {
			seenPrincipals[acl.MatchPrincipal] = 1
		} else {
			seenPrincipals[acl.MatchPrincipal]++
		}

		// Validate all ACLs
		if errs := acl.Validate(); len(errs) > 0 {
			for _, e := range errs {
				errors = append(errors, fmt.Errorf("principal %q is invalid: %w", acl.MatchPrincipal, e))
			}
		}
	}
	for principal, count := range seenPrincipals {
		if count > 1 {
			errors = append(errors, fmt.Errorf("duplicate principal matching rule %s (seen %d times)", principal, count))
		}
	}

	return errors
}
