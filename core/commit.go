package core

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"ssc/utils"
	"strings"
	"time"
)

// CreateCommit creates an ordinary commit, deriving its parent from the branch tip.
func CreateCommit(c Commit) {
	hash, err := createCommit(c)
	if err != nil {
		utils.Exit(err)
	}
	println(hash)
}

func createCommit(c Commit) (string, error) {
	if err := validateIdentity(c.AuthorName, c.AuthorEmail); err != nil {
		return "", err
	}
	if len(c.Parents) != 0 {
		return "", fmt.Errorf("ordinary commits derive their parent from the branch tip")
	}
	unlock, err := LockRepository(".")
	if err != nil {
		return "", err
	}
	defer unlock()
	return createCommitLocked(c)
}

func createCommitLocked(c Commit) (string, error) {
	current, err := currentBranch()
	if err != nil {
		return "", err
	}
	if c.Branch != current {
		return "", fmt.Errorf("commit branch does not match active branch")
	}
	history, err := readBranchLog(c.Branch)
	if err != nil {
		return "", err
	}
	c.Format = 2
	if len(history) > 0 {
		parent := history[len(history)-1]
		if _, err := ReadCommit(parent); err != nil {
			return "", fmt.Errorf("cannot read branch tip: %w", err)
		}
		c.Parents = []string{parent}
	}
	if _, err := readObject(c.Tree); err != nil {
		return "", fmt.Errorf("cannot read commit tree: %w", err)
	}
	data, err := encodeCommit(c)
	if err != nil {
		return "", err
	}
	hash := commitObjectID(data)
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(data); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	if err := atomicWrite(filepath.Join(".ssc", "objects", hash), compressed.Bytes()); err != nil {
		return "", err
	}
	history = append(history, hash)
	// Publish history only after the complete object is available.
	if err := atomicWrite(filepath.Join(".ssc", "branches", c.Branch, "commitlog"), []byte(strings.Join(history, "\n")+"\n")); err != nil {
		return "", err
	}
	return hash, nil
}

func atomicWrite(path string, data []byte) error {
	f, err := ioutil.TempFile(filepath.Dir(path), ".ssc-write-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// CommitWorkingTree holds the operation lock throughout snapshot creation.
func CommitWorkingTree(message, name, email string) error {
	if err := validateIdentity(name, email); err != nil {
		return err
	}
	return withRepositoryLock(func() error {
		branch, err := currentBranch()
		if err != nil {
			return err
		}
		c := Commit{Tree: CreateTree(), Date: time.Now().Format(time.RFC3339), Branch: branch, Message: message, AuthorName: name, AuthorEmail: email}
		hash, err := createCommitLocked(c)
		if err != nil {
			return err
		}
		println(hash)
		return nil
	})
}
