// Package types contains the config file structs
package types

import (
	"fmt"
	"strings"
)

// ACL is a mapping between a given principal and the credentials for services it will gain access to.
type ACL struct {
	MatchPrincipal string       `yaml:"match_principal"`
	Credentials    []Credential `yaml:"credentials"`
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

	// Validate that MatchPrincipals are SPIFFE IDs or patterns
	for _, acl := range c.ACLs {
		if !strings.HasPrefix(acl.MatchPrincipal, "spiffe://") {
			errors = append(errors, fmt.Errorf("%q is not a valid principal matcher", acl.MatchPrincipal))
		}
	}

	// Validate principals are not duplicated
	seenPrincipals := make(map[string]int)
	for _, acl := range c.ACLs {
		if _, found := seenPrincipals[acl.MatchPrincipal]; !found {
			seenPrincipals[acl.MatchPrincipal] = 1
		} else {
			seenPrincipals[acl.MatchPrincipal]++
		}

		// Validate providers are not duplicated
		seenProviders := make(map[string]int)
		for _, provider := range acl.Credentials {
			if _, found := seenProviders[provider.Provider]; !found {
				seenProviders[provider.Provider] = 1
			} else {
				seenProviders[provider.Provider]++
			}
		}
		for provider, count := range seenProviders {
			if count > 1 {
				errors = append(errors, fmt.Errorf("duplicate provider %q for principal %q (seen %d times)", provider, acl.MatchPrincipal, count))
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
