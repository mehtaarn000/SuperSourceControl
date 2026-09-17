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
	// ALL UNCOMMITTED CHANGES WILL BE LOST
	// TODO Add feature that stores uncommitted changes when switching branches
	if _, err := os.Stat(".ssc/branches/" + name); err != nil {
		if os.IsNotExist(err) {
			utils.Exit("Branch '" + name + "' does not exist.")
		}
	}

	currentbranch, err := ioutil.ReadFile(".ssc/branch")

	writer, err := os.Create(".ssc/branch")
	writer.WriteString(name)

	othercommitlog, err := ioutil.ReadFile(".ssc/branches/" + name + "/commitlog")
	array1 := strings.Split(string(othercommitlog), "\n")
	head1 := array1[0]

	thiscommitlog, err := ioutil.ReadFile(".ssc/branches/" + string(currentbranch) + "/commitlog")
	array2 := strings.Split(string(thiscommitlog), "\n")
	head2 := array2[0]

	if head1 != head2 {
		RevertTo(head1)
	}

	if err != nil {
		utils.Exit(err)
	}

	println("Switched to branch '" + name + "'")
}

var confirm string

func DeleteBranch(name string, force bool) {
	if _, err := os.Stat(".ssc/branches/" + name); err != nil {
		if os.IsNotExist(err) {
			utils.Exit("Branch '" + name + "' does not exist.")
		}
	}

	branch, err := ioutil.ReadFile(".ssc/branch")
	if name == string(branch) {
		utils.Exit("Cannot delete current branch. Run   ssc branch -s [branch name]  to move to another branch or run   ssc branch -ns [branch name] to create and switch to a new branch.")
	}

	if !force {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			print("Are you sure you want to delete branch: " + name + " [y/n]?")
			scanner.Scan()

			confirm = scanner.Text()
			if confirm == "Y" || confirm == "N" || confirm == "y" || confirm == "n" {
				break
			}
		}

		if confirm == "Y" || confirm == "y" {
			err := os.RemoveAll(".ssc/branches/" + name)

			if err != nil {
				utils.Exit(err)
			}

		} else {
			return
		}
	}

	err = os.RemoveAll(".ssc/branches/" + name)

	if err != nil {
		utils.Exit(err)
	}
}
