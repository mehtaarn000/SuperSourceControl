/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package main

import (
	"bufio"
	"io/ioutil"
	"os"
	"os/exec"
	"ssc/core"
	"ssc/utils"
	"strconv"
	"time"
)

func main() {
	args := os.Args

	// Metadata
	__version__ := "2.1"
	__author__ := "mehtaarn000"
	__github__ := "https://github.com/mehtaarn000/SuperSourceControl"

	// If the user runs 'ssc'
	if len(args) < 2 {
		println(core.Usage)
		os.Exit(0)
	}

	if args[1] == "-h" || args[1] == "--help" {
		println(core.Usage)
		os.Exit(0)
	}

	// If the user runs 'init'
	if args[1] == "init" {
		if len(args) > 3 {

			switch args[2] {
			case "-b", "--branch-name":
				core.Init(args[3])
				os.Exit(0)
			}
		} else {
			default_branch := core.GetSetting("defaultBranch")
			core.Init(default_branch)
			os.Exit(0)
		}
	}

	// If the .ssc directory does not exist
	if _, err := os.Stat(".ssc"); os.IsNotExist(err) {
		utils.Exit("No .ssc directory found. Run  `ssc init`  to initilize the .ssc directory.")
	}

	homedir, err := os.UserHomeDir()
	if !utils.FileExists(homedir + "/.sscconfig.json") {
		println("Creating the missing $HOME/.sscconfig.json file with default settings.")
		core.DefaultSettings(true)
	}

	if err != nil {
		utils.Exit(err)
	}

	switch args[1] {
	case "cat-file":

		if len(args) < 3 {
			utils.Exit("Command 'cat-file' requires a flag and an argument.")
		}

		// Print function's output and exits
		switch args[2] {
		case "-s", "--size":
			core.PrintSize(args[3])
		case "-c", "--content":
			core.PrintContent(args[3])
		case "-t", "--type":
			core.PrintType(args[3])
		case "-z", "--zlib-size":
			core.PrintZlibSize(args[3])
		case "-h", "--help":
			println(core.CatFileUsage)
		default:
			println(core.CatFileUsage)
		}

	case "config":

		if len(args) < 3 && args[2] != "-c" && args[2] != "--change-setting" {
			utils.Exit("Command 'config' requires a flag and an argument.")
		}

		switch args[2] {
		// Get a setting
		case "-s", "--setting":
			setting := core.GetSetting(args[3])
			println(setting)

		// Change a setting
		case "-c", "--change-setting":
			core.ChangeSetting(args[3], args[4])

		// Restore settings to default
		case "-d", "--default":
			if len(args) == 4 && args[3] == "--force" {
				core.DefaultSettings(true)
			} else {
				core.DefaultSettings(false)
			}

		case "-h", "--help":
			println(core.ConfigUsage)

		default:
			println(core.ConfigUsage)
		}

	case "revert":
		// Revert CWD to a previous commit

		if len(args) < 3 {
			utils.Exit("Command 'revert' requires a flag and an argument.")
		}

		switch args[2] {
		case "-n":
			core.RevertTo(string(args[3]))

		case "-h", "--help":
			println(core.RevertUsage)

		default:
			println(core.RevertUsage)
		}

	case "commit":

		if len(args) < 3 {
			utils.Exit("Command 'commit' requires a flag and an argument.")
		}

		switch args[2] {
		// Specify a message, create a commit with given message, and output the new commit hash
		case "-m", "--message":
			if len(args) < 4 {
				utils.Exit("Flag 'm' or 'message' requires a value.")
			}

			tree := core.CreateTree()
			file, err := ioutil.ReadFile(".ssc/branch")

			if err != nil {
				utils.Exit(err)
			}

			commit := core.Commit{Tree: tree, Date: time.Now().Format(time.RFC3339), Message: args[3], Branch: string(file)}
			core.CreateCommit(commit)

		case "-p", "--prompt":
			// Input a message
			scanner := bufio.NewScanner(os.Stdin)

			prompt_message := core.GetSetting("commitMessagePrompt")

			print(prompt_message)
			scanner.Scan()

			input := scanner.Text()

			tree := core.CreateTree()
			file, err := ioutil.ReadFile(".ssc/branch")

			if err != nil {
				utils.Exit(err)
			}

			commit := core.Commit{Tree: tree, Date: time.Now().Format(time.RFC3339), Message: input, Branch: string(file)}
			core.CreateCommit(commit)

		case "-e", "--editor":
			// Open editor with file: .ssc/tmp/message.txt
			// Message is read from file when editor is exited
			// Create a commit with this message

			editor := ""
			if len(args) < 4 {
				editor = core.GetSetting("editor")
			} else {
				editor = args[3]
			}

			cmd := exec.Command(editor, ".ssc/tmp/message.txt")
			err := cmd.Run()

			branch, err := ioutil.ReadFile(".ssc/branch")
			message, err := ioutil.ReadFile(".ssc/tmp/message.txt")

			if err != nil {
				utils.Exit(err)
			}

			tree := core.CreateTree()
			commit := core.Commit{Tree: tree, Date: time.Now().Format(time.RFC3339), Message: string(message), Branch: string(branch)}
			core.CreateCommit(commit)

		case "-f", "--file":
			// Read commit message from file
			if len(args) < 4 {
				utils.Exit("Flag 'f' requires a value.")
			}

			message, err := ioutil.ReadFile(args[3])
			branch, err := ioutil.ReadFile(".ssc/branch")

			if err != nil {
				utils.Exit(err)
			}

			tree := core.CreateTree()
			commit := core.Commit{Tree: tree, Date: time.Now().Format(time.RFC3339), Message: string(message), Branch: string(branch)}
			core.CreateCommit(commit)

		case "-h", "--help":
			println(core.CommitUsage)

		default:
			println(core.CommitUsage)
		}

	case "log":

		if len(args) < 3 {
			utils.Exit("Command 'log' requires a flag and an argument.")
		}

		switch args[2] {
		// Log n number of commits
		case "-n", "--number":

			if args[3] == "" {
				utils.Exit("Flag 'n' or 'number' requires a value.")
			}

			arg, err := strconv.ParseInt(args[3], 10, 64)
			core.Log(int(arg), false)

			if err != nil {
				utils.Exit(err)
			}

		case "-r", "--reverse":
			// Log n number of commits from first to last
			if args[3] == "" {
				utils.Exit("Flag 'r' or 'reverse' requires a value.")
			}

			arg, err := strconv.ParseInt(args[3], 10, 64)
			core.Log(int(arg), true)

			if err != nil {
				utils.Exit(err)
			}

		case "-m", "--max":
			// Log all commits
			core.MaxLog(false)

		case "-mr", "--max-reverse":
			// Log all commits backwards
			core.MaxLog(true)

		case "-h", "--help":
			println(core.LogUsage)

		default:
			println(core.LogUsage)
		}

	case "hash-object":

		if len(args) < 3 {
			utils.Exit("Command 'hash-object' requires a flag and an argument.")
		}

		switch args[2] {
		// Create hash from stdin
		case "-s", "--stdin":
			core.PrintStdinHash(string(args[3]))

		// Create object from stdin
		case "-ws", "--write-stdin":
			if len(args) == 5 && args[4] == "--quiet" {
				core.WriteStdinHash(string(args[3]), true)
			} else {
				core.WriteStdinHash(string(args[3]), false)
			}

		// Create hash from file
		case "-f", "--file":
			core.PrintFileHash(string(args[3]))

		// Create object from file
		case "-wf", "--write-file":
			if args[4] == "--quiet" {
				core.WriteFileHash(string(args[3]), true)
			} else {
				core.WriteFileHash(string(args[3]), false)
			}

		case "-h", "--help":
			println(core.HashObjectUsage)

		default:
			println(core.HashObjectUsage)
		}

	case "branch":

		if len(args) < 3 {
			utils.Exit("Command 'branch' requires a flag and an argument.")
		}

		switch args[2] {
		// Create a new branch
		case "-n", "--new":
			core.CreateBranch(args[3])

		case "-ns", "--new-switch":
			core.CreateBranch(args[3])
			core.SwitchBranch(args[3])

		case "-s", "--switch":
			core.SwitchBranch(args[3])

		case "-d", "-D", "--delete":
			force_deletion_setting := core.GetSetting("forceBranchDeletion")

			if len(args) == 5 && args[4] == "--force" {
				core.DeleteBranch(args[3], true)
			} else if force_deletion_setting == "true" {
				core.DeleteBranch(args[3], true)
			} else {
				core.DeleteBranch(args[3], false)
			}

		case "-h", "--help":
			println(core.BranchUsage)

		default:
			println(core.BranchUsage)
		}

	case "update":
		core.Update()

	case "help", "-h", "--help":
		println(core.Usage)

	case "-v", "--version":
		println(__version__)

	case "author":
		println(__author__)

	case "github":
		println(__github__)

	default:
		print(core.Usage)
	}

}
