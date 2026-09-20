package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"ssc/core"
	"strings"
)

// Pull advances only the active branch and never replaces uncommitted files.
func Pull(ctx context.Context, r *Repository, c *Client) error {
	unlock, err := core.LockRepository(r.Root)
	if err != nil {
		return err
	}
	defer unlock()
	branch, err := r.Branch()
	if err != nil {
		return err
	}
	history, err := r.History(branch)
	if err != nil {
		return err
	}
	old := tip(history)
	refs, err := c.Refs(ctx)
	if err != nil {
		return err
	}
	next := refs[branch]
	if next == "" {
		return fmt.Errorf("active branch does not exist on remote")
	}
	if next == old {
		return nil
	}
	var local graph = graph{}
	if old != "" {
		local, err = loadGraph(ctx, []string{old}, r.readObject)
		if err != nil {
			return err
		}
		if local.ancestor(next, old) {
			return nil
		}
	}
	oldFiles, err := local.files(old)
	if err != nil {
		return err
	}
	if err := r.checkClean(oldFiles); err != nil {
		return err
	}
	g, err := loadGraph(ctx, []string{next}, func(hash, kind string) ([]byte, error) {
		data, err := r.readObject(hash, kind)
		if err == nil {
			return data, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		data, err = c.Get(ctx, hash, kind)
		if err != nil {
			return nil, err
		}
		if err := r.saveObject(hash, kind, data); err != nil {
			return nil, err
		}
		return data, nil
	})
	if err != nil {
		return err
	}
	if old != "" && !g.ancestor(old, next) {
		return fmt.Errorf("pull would diverge or cross an unknown legacy history boundary; automatic merging is not supported")
	}
	files, err := g.files(next)
	if err != nil {
		return err
	}
	downloadedHistory, err := g.history(next)
	if err != nil {
		return err
	}
	// Keep older entries that exist only in a legacy branch log.
	known := map[string]bool{}
	for _, id := range history {
		known[id] = true
	}
	for _, id := range downloadedHistory {
		if !known[id] {
			history = append(history, id)
			known[id] = true
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.applyPull(branch, history, oldFiles, files, os.Rename)
}

func protected(name string) bool {
	return strings.EqualFold(name, ".ssc") || strings.EqualFold(name, ".git")
}
func (r *Repository) checkClean(expected map[string][]byte) error {
	seen := map[string]bool{}
	err := filepath.WalkDir(r.Root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == r.Root {
			return nil
		}
		rel, err := filepath.Rel(r.Root, path)
		if err != nil {
			return err
		}
		if !strings.ContainsRune(rel, os.PathSeparator) && protected(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("pull refused: symlink or special file %q", rel)
		}
		want, ok := expected[filepath.ToSlash(rel)]
		if !ok {
			return fmt.Errorf("pull refused: untracked file %q; move or commit it first", rel)
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(io.LimitReader(f, MaxObjectBytes+1))
		closeErr := f.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if !bytes.Equal(data, want) {
			return fmt.Errorf("pull refused: uncommitted changes in %q", rel)
		}
		seen[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("pull refused: tracked files are missing")
	}
	return nil
}
func entryNames(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !protected(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// applyPull stages files and retains the previous checkout until the ref is published.
// A sync-pending marker and backup survive interruption or failed rollback.
func (r *Repository) applyPull(branch string, history []string, oldFiles, files map[string][]byte, rename func(string, string) error) (result error) {
	logPath, err := r.historyPath(branch)
	if err != nil {
		return err
	}
	oldLog, err := os.ReadFile(logPath)
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp(r.metadata("tmp"), "pull-")
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			os.RemoveAll(stage)
		}
	}()
	newDir, backup := filepath.Join(stage, "new"), filepath.Join(stage, "backup")
	for _, dir := range []string{newDir, backup} {
		if err := os.Mkdir(dir, 0700); err != nil {
			return err
		}
	}
	if err := materialize(newDir, files); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stage, "old-commitlog"), oldLog, 0600); err != nil {
		return err
	}
	if err := r.checkClean(oldFiles); err != nil {
		return err
	}
	oldEntries, err := entryNames(r.Root)
	if err != nil {
		return err
	}
	newEntries, err := entryNames(newDir)
	if err != nil {
		return err
	}
	marker := r.metadata("sync-pending")
	journal, _ := json.MarshalIndent(map[string]interface{}{"backupDirectory": stage, "branch": branch, "oldEntries": oldEntries, "newEntries": newEntries}, "", "  ")
	if err := atomicWrite(marker, journal); err != nil {
		return err
	}
	var movedOld, movedNew []string
	rollback := func(cause error) error {
		var rollbackErr error
		for i := len(movedNew) - 1; i >= 0; i-- {
			name := movedNew[i]
			if err := os.Rename(filepath.Join(r.Root, name), filepath.Join(newDir, name)); err != nil {
				rollbackErr = err
			}
		}
		for i := len(movedOld) - 1; i >= 0; i-- {
			name := movedOld[i]
			if err := os.Rename(filepath.Join(backup, name), filepath.Join(r.Root, name)); err != nil {
				rollbackErr = err
			}
		}
		if rollbackErr != nil {
			keep = true
			return fmt.Errorf("pull failed (%v) and rollback needs recovery; backups retained at %s: %w", cause, stage, rollbackErr)
		}
		if err := os.Remove(marker); err != nil {
			keep = true
			return fmt.Errorf("checkout restored but recovery marker remains: %w", err)
		}
		return cause
	}
	for _, name := range oldEntries {
		if err := rename(filepath.Join(r.Root, name), filepath.Join(backup, name)); err != nil {
			return rollback(err)
		}
		movedOld = append(movedOld, name)
	}
	for _, name := range newEntries {
		if err := rename(filepath.Join(newDir, name), filepath.Join(r.Root, name)); err != nil {
			return rollback(err)
		}
		movedNew = append(movedNew, name)
	}
	if err := atomicWrite(logPath, historyBytes(history)); err != nil {
		return rollback(err)
	}
	if err := os.Remove(marker); err != nil {
		keep = true
		return fmt.Errorf("pull completed but could not remove recovery marker: %w", err)
	}
	return nil
}
