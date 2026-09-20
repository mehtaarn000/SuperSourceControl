package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"ssc/core"
	"strings"
)

func validateBranches(refs map[string]string) error {
	names := map[string]string{}
	for name := range refs {
		if !core.ValidBranchName(name) || !core.ValidBranchName(strings.ToLower(name)) {
			return fmt.Errorf("invalid branch name")
		}
		lower := strings.ToLower(name)
		if prior, ok := names[lower]; ok && prior != name {
			return fmt.Errorf("case-conflicting branch names")
		}
		names[lower] = name
	}
	for name := range names {
		for parent := strings.LastIndexByte(name, '/'); parent >= 0; parent = strings.LastIndexByte(name[:parent], '/') {
			if _, ok := names[name[:parent]]; ok {
				return fmt.Errorf("overlapping branch namespaces")
			}
		}
	}
	return nil
}

// Clone stages an entire repository before publishing a new destination directory.
func Clone(ctx context.Context, c *Client, destination, branch string) error {
	destination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("clone destination already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	refs, err := c.Refs(ctx)
	if err != nil {
		return err
	}
	if err := validateBranches(refs); err != nil {
		return err
	}
	names := make([]string, 0, len(refs))
	roots := make([]string, 0, len(refs))
	for name, hash := range refs {
		names = append(names, name)
		roots = append(roots, hash)
	}
	sort.Strings(names)
	if branch == "" {
		branch = "main"
		if len(names) > 0 {
			if _, ok := refs["main"]; !ok {
				if _, ok := refs["master"]; ok {
					branch = "master"
				} else {
					branch = names[0]
				}
			}
		}
	}
	if !core.ValidBranchName(branch) || !core.ValidBranchName(strings.ToLower(branch)) {
		return fmt.Errorf("invalid selected branch")
	}
	if len(refs) > 0 && refs[branch] == "" {
		return fmt.Errorf("selected branch does not exist on remote")
	}
	g, err := loadGraph(ctx, roots, func(hash, kind string) ([]byte, error) { return c.Get(ctx, hash, kind) })
	if err != nil {
		return err
	}
	files, err := g.files(refs[branch])
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(destination), ".ssc-clone-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for _, dir := range []string{".ssc/objects", ".ssc/branches", ".ssc/tmp"} {
		if err := os.MkdirAll(filepath.Join(stage, dir), 0755); err != nil {
			return err
		}
	}
	repo := &Repository{Root: stage}
	for hash, obj := range g {
		if err := repo.saveObject(hash, obj.Kind, obj.Data); err != nil {
			return err
		}
	}
	if len(names) == 0 {
		names = []string{branch}
	}
	for _, name := range names {
		history, err := g.history(refs[name])
		if err != nil {
			return err
		}
		dir := repo.metadata("branches", name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		f, err := os.OpenFile(filepath.Join(dir, "commitlog"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := f.Write(historyBytes(history))
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if err := atomicWrite(repo.metadata("branch"), []byte(branch)); err != nil {
		return err
	}
	var trees []string
	for hash, obj := range g {
		if obj.Kind == "tree" {
			trees = append(trees, hash)
		}
	}
	sort.Strings(trees)
	if err := atomicWrite(repo.metadata("trees"), historyBytes(trees)); err != nil {
		return err
	}
	config, _ := json.Marshal(map[string]string{"url": c.base})
	if err := atomicWrite(repo.metadata("remote.json"), config); err != nil {
		return err
	}
	if err := materialize(stage, files); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Recheck after download; publish the completed directory in a single rename.
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("clone destination appeared during transfer")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stage, destination); err != nil {
		return err
	}

	return nil
}
func historyBytes(history []string) []byte {
	if len(history) == 0 {
		return nil
	}
	return []byte(strings.Join(history, "\n") + "\n")
}
func materialize(root string, files map[string][]byte) error {
	for name, data := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := f.Write(data)
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
