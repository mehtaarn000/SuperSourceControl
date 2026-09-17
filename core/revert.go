package core

import (
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"ssc/utils"
	"strings"
)

// RevertTo restores a snapshot, replacing uncommitted working files.
func RevertTo(hash string) {
	if err := restoreSnapshot(hash); err != nil { utils.Exit(err) }
}

func readObject(hash string) ([]byte, error) {
	decoded, err := hex.DecodeString(hash)
	if err != nil || len(decoded) != 20 { return nil, fmt.Errorf("invalid object hash: %q", hash) }
	f, err := os.Open(filepath.Join(".ssc", "objects", hash))
	if err != nil { return nil, err }
	defer f.Close()
	r, err := zlib.NewReader(f)
	if err != nil { return nil, err }
	defer r.Close()
	return ioutil.ReadAll(r)
}

func restoreSnapshot(hash string) error {
	commit, err := readObject(hash)
	if err != nil { return err }
	first := strings.SplitN(string(commit), "\n", 2)[0]
	if !strings.HasPrefix(first, "tree ") { return fmt.Errorf("object is not a commit: %s", hash) }
	tree, err := readObject(strings.TrimPrefix(first, "tree "))
	if err != nil { return err }
	// Load and validate every object before changing working files.
	files := map[string][]byte{}
	for _, line := range strings.Split(string(tree), "\n") {
		if line == "" { continue }
		sep := strings.LastIndexByte(line, ' ')
		if sep < 1 { return fmt.Errorf("invalid tree entry: %q", line) }
		path := filepath.FromSlash(line[:sep])
		clean := filepath.Clean(path)
		if filepath.IsAbs(path) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".." + string(os.PathSeparator)) || strings.Contains(clean, ".ssc") {
			return fmt.Errorf("invalid snapshot path: %q", path)
		}
		data, err := readObject(line[sep+1:])
		if err != nil { return err }
		files[clean] = data
	}
	for _, path := range utils.GetFiles() {
		if _, keep := files[filepath.Clean(path)]; !keep {
			if err := os.Remove(path); err != nil { return err }
		}
	}
	for path, data := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { return err }
		if err := ioutil.WriteFile(path, data, 0644); err != nil { return err }
	}
	return nil
}
