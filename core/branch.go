/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"bufio"
	"fmt"
	"path/filepath"
	"io/ioutil"
	"os"
	"ssc/utils"
	"strings"
)

// Branch names are also paths beneath .ssc/branches.
func validateBranchName(name string) bool {
	if name == "" || name == "@" || strings.Contains(name, "..") || strings.Contains(name, "@{") {
		return false
	}
	for _, ch := range name {
		if ch <= ' ' || ch == 127 || strings.ContainsRune("~^:?*[\\", ch) {
			return false
		}
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".") || strings.HasSuffix(part, ".lock") || part == "commitlog" {
			return false
		}
	}
	return true
}

func CreateBranch(name string) {
	if err := createBranch(name); err != nil { utils.Exit(err) }
}

func readBranchLog(name string) ([]string, error) {
	if !validateBranchName(name) { return nil, fmt.Errorf("invalid branch name: %q", name) }
	data, err := ioutil.ReadFile(filepath.Join(".ssc", "branches", name, "commitlog"))
	if err != nil { return nil, err }
	return strings.Fields(string(data)), nil
}

func currentBranch() (string, error) {
	data, err := ioutil.ReadFile(".ssc/branch")
	if err != nil { return "", err }
	name := string(data)
	if !validateBranchName(name) { return "", fmt.Errorf("invalid current branch: %q", name) }
	return name, nil
}

func createBranch(name string) error {
	if !validateBranchName(name) { return fmt.Errorf("invalid branch name: %q", name) }
	current, err := currentBranch()
	if err != nil { return err }
	commits, err := readBranchLog(current)
	if err != nil { return err }
	if len(commits) == 0 { return fmt.Errorf("make at least one commit before creating a branch") }
	dir := filepath.Join(".ssc", "branches", name)
	// A branch cannot be nested beneath another branch's storage directory.
	for parent := filepath.Dir(dir); parent != filepath.Join(".ssc", "branches"); parent = filepath.Dir(parent) {
		if _, err := os.Stat(filepath.Join(parent, "commitlog")); err == nil {
			return fmt.Errorf("branch name conflicts with an existing branch: %q", name)
		} else if !os.IsNotExist(err) { return err }
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil { return err }
	if err := os.Mkdir(dir, 0755); err != nil { return err }
	// Inherit the full history, whose final entry is the current tip.
	err = ioutil.WriteFile(filepath.Join(dir, "commitlog"), []byte(strings.Join(commits, "\n") + "\n"), 0644)
	if err != nil { os.RemoveAll(dir) }
	return err
}

func SwitchBranch(name string) {
	if err := switchBranch(name); err != nil { utils.Exit(err) }
	println("Switched to branch '" + name + "'")
}

func switchBranch(name string) error {
	commits, err := readBranchLog(name)
	if err != nil { return err }
	current, err := currentBranch()
	if err != nil { return err }
	if current == name { return nil }
	if len(commits) == 0 { return fmt.Errorf("branch %q has no commits", name) }
	oldCommits, err := readBranchLog(current)
	if err != nil { return err }
	tip := commits[len(commits)-1]
	if len(oldCommits) == 0 || oldCommits[len(oldCommits)-1] != tip {
		if err := restoreSnapshot(tip); err != nil { return err }
	}
	// Only change the active branch after its snapshot has been restored.
	return ioutil.WriteFile(".ssc/branch", []byte(name), 0644)
}

var confirm string

func DeleteBranch(name string, force bool) {
	if err := checkBranchDeletion(name); err != nil { utils.Exit(err) }
	if !force {
		print("Are you sure you want to delete branch: " + name + " [y/n]?")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() || !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") { return }
	}
	if err := deleteBranch(name); err != nil { utils.Exit(err) }
}

func checkBranchDeletion(name string) error {
	if _, err := readBranchLog(name); err != nil { return err }
	current, err := currentBranch()
	if err != nil { return err }
	if current == name { return fmt.Errorf("cannot delete the current branch: %q", name) }
	entries, err := ioutil.ReadDir(filepath.Join(".ssc", "branches", name))
	if err != nil { return err }
	if len(entries) != 1 || entries[0].Name() != "commitlog" {
		return fmt.Errorf("branch %q contains unexpected entries; refusing deletion", name)
	}
	return nil
}

func deleteBranch(name string) error {
	if err := checkBranchDeletion(name); err != nil { return err }
	return os.RemoveAll(filepath.Join(".ssc", "branches", name))
}
