package core

import (
	"encoding/hex"
	"fmt"
	"path"
	"strings"

	"golang.org/x/crypto/ripemd160"
)

// ObjectReference describes a typed edge in SSC's stored object graph.
type ObjectReference struct {
	Hash string
	Type string
}

// ValidObjectID and ValidBranchName share the local repository's validation rules.
func ValidObjectID(id string) bool     { return validObjectID(id) }
func ValidBranchName(name string) bool { return validateBranchName(name) }

// InspectObject validates a wire object and returns its ID and direct references.
// Wire bodies are decompressed object bytes, not local zlib files.
func InspectObject(kind string, data []byte) (string, []ObjectReference, error) {
	var refs []ObjectReference
	switch kind {
	case "blob":
	case "tree":
		seen := map[string]bool{}
		for _, line := range strings.Split(string(data), "\n") {
			if line == "" {
				continue
			}
			split := strings.LastIndexByte(line, ' ')
			if split < 1 {
				return "", nil, fmt.Errorf("invalid tree entry")
			}
			name, hash := line[:split], line[split+1:]
			if path.IsAbs(name) || path.Clean(name) != name || name == "." || name == ".." || strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\\x00\r:") || strings.Contains(name, ".ssc") || strings.HasPrefix(name, ".git/") || name == ".git" || !validObjectID(hash) || seen[name] {
				return "", nil, fmt.Errorf("invalid or duplicate tree path/hash")
			}
			seen[name] = true
			refs = append(refs, ObjectReference{Hash: hash, Type: "blob"})
		}
		// A file cannot also be a directory containing another file.
		for name := range seen {
			for dir := path.Dir(name); dir != "."; dir = path.Dir(dir) {
				if seen[dir] {
					return "", nil, fmt.Errorf("conflicting tree paths")
				}
			}
		}
	case "commit":
		c, err := decodeCommit(data)
		if err != nil {
			return "", nil, err
		}
		refs = append(refs, ObjectReference{Hash: c.Tree, Type: "tree"})
		for _, parent := range c.Parents {
			refs = append(refs, ObjectReference{Hash: parent, Type: "commit"})
		}
		if c.Format == 2 {
			return commitObjectID(data), refs, nil
		}
	default:
		return "", nil, fmt.Errorf("unsupported object type")
	}
	// Preserve the historical hash convention for blobs, trees, and legacy commits.
	h := ripemd160.New()
	h.Write([]byte(kind + " " + string(rune(len(data)))))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), refs, nil
}
