/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package main

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"ssc/core"
	"ssc/remote"
	"ssc/server"
	"ssc/utils"
	"strconv"
	"syscall"
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

	if args[1] == "serve" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := server.Run(ctx, args[2:], os.Stdout); err != nil {
			utils.Exit(err)
		}
		return
	}

	if args[1] == "clone" || args[1] == "push" || args[1] == "pull" || args[1] == "remote" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := remote.Run(ctx, args[1:], os.Stdout); err != nil {
			utils.Exit(err)
		}
		return
	}

	core.EnsureConfig()

	// If the user runs 'init'
	if args[1] == "init" {
		switch {
		case len(args) == 2:
			core.Init(core.GetSetting("defaultBranch"))
		case len(args) == 4 && (args[2] == "-b" || args[2] == "--branch-name"):
			core.Init(args[3])
		default:
			utils.Exit("Usage: ssc init [-b | --branch-name <branch>]")
		}
		return
	}

	// If the .ssc directory does not exist
	if _, err := os.Stat(".ssc"); os.IsNotExist(err) && args[1] != "config" {
		utils.Exit("No .ssc directory found. Run  `ssc init`  to initilize the .ssc directory.")
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

		if len(args) < 3 {
			utils.Exit("Command 'config' requires a flag and an argument.")
		}

		switch args[2] {
		// Get a setting
		case "-s", "--setting":
			if len(args) != 4 {
				utils.Exit(core.ConfigUsage)
			}
			setting := core.GetSetting(args[3])
			println(setting)

		// Change a setting
		case "-c", "--change-setting":
			if len(args) != 5 {
				utils.Exit(core.ConfigUsage)
			}
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
			utils.Exit(core.CommitUsage)
		}
		if args[2] == "-h" || args[2] == "--help" {
			println(core.CommitUsage)
			return
		}
		authorName, authorEmail, err := core.ConfiguredAuthor()
		if err != nil {
			utils.Exit(err)
		}
		var message string
		switch args[2] {
		case "-m", "--message":
			if len(args) != 4 {
				utils.Exit(core.CommitUsage)
			}
			message = args[3]
		case "-p", "--prompt":
			if len(args) != 3 {
				utils.Exit(core.CommitUsage)
			}
			print(core.GetSetting("commitMessagePrompt"))
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					utils.Exit(err)
				}
				utils.Exit("No commit message received")
			}
			message = scanner.Text()
		case "-f", "--file":
			if len(args) != 4 {
				utils.Exit(core.CommitUsage)
			}
			data, err := ioutil.ReadFile(args[3])
			if err != nil {
				utils.Exit(err)
			}
			message = string(data)
		case "-e", "--editor":
			if len(args) > 4 {
				utils.Exit(core.CommitUsage)
			}
			editor := ""
			if len(args) == 4 {
				editor = args[3]
			} else {
				editor = core.GetSetting("editor")
			}
			cmd := exec.Command(editor, ".ssc/tmp/message.txt")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				utils.Exit(err)
			}
			data, err := ioutil.ReadFile(".ssc/tmp/message.txt")
			if err != nil {
				utils.Exit(err)
			}
			message = string(data)
		default:
			utils.Exit(core.CommitUsage)
		}
		if err := core.CommitWorkingTree(message, authorName, authorEmail); err != nil {
			utils.Exit(err)
		}

	case "log":

		if len(args) < 3 {
			utils.Exit("Command 'log' requires a flag and an argument.")
		}

		switch args[2] {
		// Log n number of commits
		case "-n", "--number":

			if len(args) != 4 {
				utils.Exit("Flag 'n' or 'number' requires a value.")
			}

			arg, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				utils.Exit(err)
			}
			core.Log(int(arg), false)

		case "-r", "--reverse":
			// Log n number of commits from first to last
			if len(args) != 4 {
				utils.Exit("Flag 'r' or 'reverse' requires a value.")
			}

			arg, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				utils.Exit(err)
			}
			core.Log(int(arg), true)

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
		if len(args) == 3 && (args[2] == "-h" || args[2] == "--help") {
			println(core.BranchUsage)
			return
		}
		if len(args) < 4 {
			utils.Exit(core.BranchUsage)
		}
		deleting := args[2] == "-d" || args[2] == "-D" || args[2] == "--delete"
		if len(args) != 4 && !(deleting && len(args) == 5 && args[4] == "--force") {
			utils.Exit(core.BranchUsage)
		}
		switch args[2] {
		case "-n", "--new":
			core.CreateBranch(args[3])
		case "-ns", "--new-switch":
			core.CreateBranch(args[3])
			core.SwitchBranch(args[3])
		case "-s", "--switch":
			core.SwitchBranch(args[3])
		case "-d", "-D", "--delete":
			force := len(args) == 5 || core.GetSetting("forceBranchDeletion") == "true"
			core.DeleteBranch(args[3], force)
		default:
			utils.Exit(core.BranchUsage)
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
