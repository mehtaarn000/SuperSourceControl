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
	"strconv"
	"time"
)

func main() {
	args := os.Args

	// Metadata
	__version__ := "1.0"
	__author__ := "mehtaarn000"
	__github__ := "https://github.com/mehtaarn000/SuperSourceControl"

	// If the user runs 'init'
	if args[1] == "init" {
		core.Init()
		os.Exit(0)
	}

	// If the .ssc directory does not exist
	if _, err := os.Stat(".ssc"); os.IsNotExist(err) {
		panic("No .ssc directory found. Run  `ssc init`  to initilize the .ssc directory.")
	}

	// If the user runs 'ssc'
	if len(args) < 2 {
		print(core.Usage)
		os.Exit(0)
	}

	switch args[1] {
	case "cat-file":

		if len(args) < 3 {
			panic("Command 'cat-file' requires a flag and an argument.")
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

	case "revert":
		// Revert CWD to a previous commit

		if len(args) < 3 {
			panic("Command 'revert' requires a flag and an argument.")
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
			panic("Command 'commit' requires a flag and an argument.")
		}

		switch args[2] {
		// Specify a message, create a commit with given message, and output the new commit hash
		case "-m", "--message":
			if len(args) < 4 {
				panic("Flag 'm' or 'message' requires a value.")
			}

			tree := core.CreateTree()
			file, err := ioutil.ReadFile(".ssc/branch")

			if err != nil {
				panic(err)
			}

			commit := core.Commit{Tree: tree, Date: time.Now().Format(time.RFC3339), Message: args[3], Branch: string(file)}
			core.CreateCommit(commit)

		case "-p", "--prompt":
			// Input a message
			scanner := bufio.NewScanner(os.Stdin)

			print("Input a commit message: ")
			scanner.Scan()

			input := scanner.Text()

			tree := core.CreateTree()
			file, err := ioutil.ReadFile(".ssc/branch")

			if err != nil {
				panic(err)
			}

			commit := core.Commit{Tree: tree, Date: time.Now().Format(time.RFC3339), Message: input, Branch: string(file)}
			core.CreateCommit(commit)

		case "-e", "--editor":
			// Open editor with file: .ssc/tmp/message.txt
			// Message is read from file when editor is exited
			// Create a commit with this message
			//TODO add default editor
			if len(args) < 4 {
				panic("Flag 'e' or 'editor' requires a value.")
			}

			cmd := exec.Command(args[3], ".ssc/tmp/message.txt")
			err := cmd.Run()

			branch, err := ioutil.ReadFile(".ssc/branch")
			message, err := ioutil.ReadFile(".ssc/tmp/message.txt")

			if err != nil {
				panic(err)
			}

			tree := core.CreateTree()
			commit := core.Commit{Tree: tree, Date: time.Now().Format(time.RFC3339), Message: string(message), Branch: string(branch)}
			core.CreateCommit(commit)

		case "-f", "--file":
			// Read commit message from file
			if len(args) < 4 {
				panic("Flag 'f' requires a value.")
			}

			message, err := ioutil.ReadFile(args[3])
			branch, err := ioutil.ReadFile(".ssc/branch")

			if err != nil {
				panic(err)
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
			panic("Command 'log' requires a flag and an argument.")
		}

		switch args[2] {
		// Log n number of commits
		case "-n", "--number":

			if args[3] == "" {
				panic("Flag 'n' or 'number' requires a value.")
			}

			arg, err := strconv.ParseInt(args[3], 10, 64)
			core.Log(int(arg), false)

			if err != nil {
				panic(err)
			}

		case "-r", "--reverse":
			// Log n number of commits from first to last
			if args[3] == "" {
				panic("Flag 'r' or 'reverse' requires a value.")
			}

			arg, err := strconv.ParseInt(args[3], 10, 64)
			core.Log(int(arg), true)

			if err != nil {
				panic(err)
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
			panic("Command 'hash-object' requires a flag and an argument.")
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

	case "help", "-h", "--help":
		print(core.Usage)
	
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
