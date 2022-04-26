// Package types contains the config file structs
package types

import (
	"fmt"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
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

	// After removing a possible trailing wildcard, defer validation of the SVID to SPIFFE
	_, err := spiffeid.FromString(nonWildCardMatchPrincipal)
	if err != nil {
		errors = append(errors, err)
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

// ConfigFile represents the config file that will be loaded from disk, or some other mechanism.
type ConfigFile struct {
	SPIFFE SpiffeConfig `yaml:"spiffe"`
	ACLs   []ACL        `yaml:"acls"`
}

func (c *ConfigFile) Validate() []error {
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

// SpiffeConfig represents the SPIFFE configuration section of spiffe-connector's config file
type SpiffeConfig struct {
	SVIDSources SVIDSources `yaml:"svid_sources"`
}

// SVIDSources determines where spiffe-connector will obtain its own SVID and trust domain information.
// The SPIFFE Workload API and Static files are supported.
type SVIDSources struct {
	WorkloadAPI *struct {
		SocketPath string `yaml:"socket_path"`
	} `yaml:"workload_api,omitempty"`
	Files *struct {
		TrustDomainCA string `yaml:"trust_domain_ca"`
		SVIDCert      string `yaml:"svid_cert"`
		SVIDKey       string `yaml:"svid_key"`
	} `yaml:"files,omitempty"`
}
