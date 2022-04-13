package principal

import (
	"fmt"
	"strings"

	"github.com/jetstack/spiffe-connector/types"
)

// MatchingACLs accepts a list of types.ACL and a principal string to match against.
// It returns false if there was no match found, and an ACL which matched if
// one was found.
func MatchingACL(acls []types.ACL, principal string) (bool, *types.ACL, error) {
	var globMatchingACLIndexes []int
	for i, acl := range acls {
		if acl.MatchPrincipal == principal {
			return true, &acl, nil
		}

		if strings.HasPrefix(principal, strings.TrimSuffix(acl.MatchPrincipal, "*")) {
			globMatchingACLIndexes = append(globMatchingACLIndexes, i)
		}
	}

	if len(globMatchingACLIndexes) == 1 {
		return true, &acls[globMatchingACLIndexes[0]], nil
	}

	if len(globMatchingACLIndexes) > 1 {
		return false, &types.ACL{}, fmt.Errorf("principal matched multple ACLs")
	}

	return false, &types.ACL{}, nil
}
