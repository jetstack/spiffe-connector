// Package types contains the config file structs
package types

import "fmt"

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
	seenPrincipals := make(map[string]int)
	for _, principal := range c.ACLs {
		if _, found := seenPrincipals[principal.MatchPrincipal]; !found {
			seenPrincipals[principal.MatchPrincipal] = 1
		} else {
			seenPrincipals[principal.MatchPrincipal]++
		}
		seenProviders := make(map[string]int)
		for _, provider := range principal.Credentials {
			if _, found := seenProviders[provider.Provider]; !found {
				seenProviders[provider.Provider] = 1
			} else {
				seenProviders[provider.Provider]++
			}
		}
		for provider, count := range seenProviders {
			if count > 1 {
				errors = append(errors, fmt.Errorf("duplicate provider %q for principal %q (seen %d times)", provider, principal.MatchPrincipal, count))
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
