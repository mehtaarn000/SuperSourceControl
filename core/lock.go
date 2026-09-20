package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// LockRepository serializes commands that change working files or branch history.
func LockRepository(root string) (func(), error) {
	if _, err := os.Lstat(filepath.Join(root, ".ssc", "sync-pending")); err == nil {
		return nil, fmt.Errorf("an interrupted pull needs recovery; see .ssc/sync-pending")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	path := filepath.Join(root, ".ssc", "commit.lock")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("repository is locked; another command may be running: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return nil, err
	}
	return func() { os.Remove(path) }, nil
}

func withRepositoryLock(action func() error) error {
	unlock, err := LockRepository(".")
	if err != nil {
		return err
	}
	defer unlock()
	return action()
}
