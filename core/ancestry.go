package core

import "errors"

// ErrLegacyHistory means an old commit lacks the links required to answer safely.
var ErrLegacyHistory = errors.New("ancestry is incomplete: legacy commits have no parent links")

// IsAncestor reports whether ancestor is reachable through descendant's parents.
// A commit is its own ancestor. False with nil error means a complete search;
// legacy boundaries return ErrLegacyHistory instead of a misleading false result.
func IsAncestor(ancestor, descendant string) (bool, error) {
	if _, err := ReadCommit(ancestor); err != nil {
		return false, err
	}
	pending := []string{descendant}
	seen := map[string]bool{}
	legacy := false
	for len(pending) > 0 {
		hash := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[hash] {
			continue
		}
		seen[hash] = true
		c, err := ReadCommit(hash)
		if err != nil {
			return false, err
		}
		if hash == ancestor {
			return true, nil
		}
		if c.Format == 1 {
			legacy = true
		}
		pending = append(pending, c.Parents...)
	}
	if legacy {
		return false, ErrLegacyHistory
	}
	return false, nil
}
