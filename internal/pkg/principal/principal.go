package principal

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jetstack/spiffe-connector/types"
)

func MatchingACL(acls []types.ACL, principal string) (bool, *types.ACL, error) {
	// attempt to find an exact match
	for _, acl := range acls {
		if acl.MatchPrincipal == principal {
			return true, &acl, nil
		}
	}

	// attempt to find match from glob
	globCount := func(acl types.ACL) int {
		return strings.Count(acl.MatchPrincipal, "*")
	}
	globMatches := make(map[int]int)
	// defaulted to index from any match so only glob ACLs are compared.
	var mostSpecificACL int
	for i, acl := range acls {
		if !strings.Contains(acl.MatchPrincipal, "*") {
			continue
		}
		fmt.Println(acl.MatchPrincipal)

		generatedPattern := strings.ReplaceAll(acl.MatchPrincipal, "*", `[A-Za-z0-9-]+`)

		re, err := regexp.Compile(generatedPattern)
		if err != nil {
			return false, &types.ACL{}, fmt.Errorf("failed to generate pattern to match %q ACL: %w", acl.MatchPrincipal, err)
		}

		if re.Match([]byte(principal)) {
			mostSpecificACL = i
			globMatches[i] = globCount(acl)
		}
	}
	if len(globMatches) > 0 {
		for index, score := range globMatches {
			if score < globMatches[mostSpecificACL] {
				mostSpecificACL = index
			}
		}
		return true, &acls[mostSpecificACL], nil
	}

	return false, &types.ACL{}, nil
}
