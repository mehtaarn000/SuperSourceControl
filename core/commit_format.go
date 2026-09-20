package core

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ripemd160"
)

func validObjectID(id string) bool {
	decoded, err := hex.DecodeString(id)
	return err == nil && len(decoded) == 20 && id == strings.ToLower(id)
}

func validateIdentity(name, email string) error {
	for _, value := range []string{name, email} {
		if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf("configure a nonempty, single-line authorName and authorEmail with ssc config -c")
		}
	}
	return nil
}

// encodeCommit writes the version 2 format; parent order is significant.
func encodeCommit(c Commit) ([]byte, error) {
	if !validObjectID(c.Tree) {
		return nil, fmt.Errorf("invalid tree hash: %q", c.Tree)
	}
	if !validateBranchName(c.Branch) {
		return nil, fmt.Errorf("invalid branch: %q", c.Branch)
	}
	if _, err := time.Parse(time.RFC3339, c.Date); err != nil {
		return nil, err
	}
	if err := validateIdentity(c.AuthorName, c.AuthorEmail); err != nil {
		return nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "tree %s\ndate %s\nbranch %s\nformat 2\n", c.Tree, c.Date, c.Branch)
	seen := map[string]bool{}
	for _, parent := range c.Parents {
		if !validObjectID(parent) || seen[parent] {
			return nil, fmt.Errorf("invalid or duplicate parent: %q", parent)
		}
		seen[parent] = true
		fmt.Fprintf(&b, "parent %s\n", parent)
	}
	fmt.Fprintf(&b, "author-name %s\nauthor-email %s\n\n%s", c.AuthorName, c.AuthorEmail, c.Message)
	return []byte(b.String()), nil
}

func decodeCommit(data []byte) (Commit, error) {
	c := Commit{Format: 1}
	parts := strings.SplitN(string(data), "\n\n", 2)
	if len(parts) != 2 {
		return c, fmt.Errorf("invalid commit: missing message separator")
	}
	c.Message = parts[1]
	seen := map[string]bool{}
	for _, line := range strings.Split(parts[0], "\n") {
		entry := strings.SplitN(line, " ", 2)
		if len(entry) != 2 {
			return c, fmt.Errorf("invalid commit header: %q", line)
		}
		key, value := entry[0], entry[1]
		if seen[key] && key != "parent" {
			return c, fmt.Errorf("duplicate commit header: %s", key)
		}
		seen[key] = true
		switch key {
		case "tree":
			c.Tree = value
		case "date":
			c.Date = value
		case "branch":
			c.Branch = value
		case "format":
			if value != "2" {
				return c, fmt.Errorf("unsupported commit format: %q", value)
			}
			c.Format = 2
		case "parent":
			c.Parents = append(c.Parents, value)
		case "author-name":
			c.AuthorName = value
		case "author-email":
			c.AuthorEmail = value
		default:
			return c, fmt.Errorf("unknown commit header: %s", key)
		}
	}
	if !validObjectID(c.Tree) || !validateBranchName(c.Branch) {
		return c, fmt.Errorf("invalid commit tree or branch")
	}
	if _, err := time.Parse(time.RFC3339, c.Date); err != nil {
		return c, err
	}
	if c.Format == 2 {
		if _, err := encodeCommit(c); err != nil {
			return c, err
		}
	} else if seen["parent"] || seen["author-name"] || seen["author-email"] {
		return c, fmt.Errorf("metadata requires commit format 2")
	}
	return c, nil
}

func commitObjectID(data []byte) string {
	h := ripemd160.New()
	h.Write([]byte("commit " + strconv.Itoa(len(data)) + "\x00"))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// ReadCommit supports legacy snapshots and verifies version 2 object identities.
func ReadCommit(hash string) (Commit, error) {
	data, err := readObject(hash)
	if err != nil {
		return Commit{}, err
	}
	c, err := decodeCommit(data)
	if err != nil {
		return c, err
	}
	if c.Format == 2 && commitObjectID(data) != hash {
		return c, fmt.Errorf("commit hash mismatch: %s", hash)
	}
	return c, nil
}
