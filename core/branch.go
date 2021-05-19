package core

import (
	"bufio"
	"io/ioutil"
	"os"
	"ssc/utils"
	"strings"

	"github.com/glenn-brown/golang-pkg-pcre/src/pkg/pcre"
)

func validateBranchName(name string) bool {
	newmatcher, err := pcre.Compile("^(?!/|.*([/.]\\.|//|@\\{|\\\\))[^\040\177 ~^:?*\\[]+(?<!\\.lock|[/.])$", 0)
	match := newmatcher.Matcher([]byte(name), 0).MatchString(name, 0)

	if err != nil {
		utils.Exit(err)
	}

	return match
}

func CreateBranch(name string) {
	if _, err := os.Stat(".ssc/branches/" + name); err != nil {
		if os.IsExist(err) {
			utils.Exit("Branch '" + name + "' already exists.")
		}
	}

	currentbranch, err := ioutil.ReadFile(".ssc/branch")
	othercommitlog, err := ioutil.ReadFile(".ssc/branches/" + string(currentbranch) + "/commitlog")
	array := strings.Split(string(othercommitlog), "\n")
	head := array[0]

	if head == "" || head == "\n" {
		utils.Exit("At least 1 commit must be made on the default branch before new branches can be created.")
	}

	match := validateBranchName(name)

	if !match {
		utils.Exit("Invalid branch name: '" + name + "'")
	}

	err = os.Mkdir(".ssc/branches/"+name, 0777)
	f, err := os.Create(".ssc/branches/" + name + "/commitlog")
	defer f.Close()

	f.WriteString(head + "\n")

	if err != nil {
		utils.Exit(err)
	}
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
