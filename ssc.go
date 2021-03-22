package main

import (
	"io/ioutil"
	"os"
	"ssc/core"
	"time"
	"os/exec"
	"strconv"
)

func main() {
	args := os.Args

	if _, err := os.Stat(".ssc"); os.IsNotExist(err) {
		panic("No .ssc directory found. Run  `ssc init`  to initilize the .ssc directory.")
	}

	if len(args) < 2 {
		print(core.Usage)
		os.Exit(0)
	}

	switch args[1] {
	case "cat-file":
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
	
	case "commit":
		switch args[2] {
		case "-m", "--message":
			if len(args) < 4  {
				panic("Flag 'm' or 'message' requires a value.")
			}

			tree := core.CreateTree()
			file, err := ioutil.ReadFile(".ssc/branch")

			if err != nil {
				panic(err)
			}

			commit := core.Commit{Tree:tree, Date:time.Now().Format(time.RFC3339), Message:args[3], Branch:string(file)}
			core.CreateCommit(commit)
		
	
		case "-e", "--editor":
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
			commit := core.Commit{Tree:tree, Date:time.Now().Format(time.RFC3339), Message:string(message), Branch:string(branch)}
			core.CreateCommit(commit)
			
		case "-f", "--file":
			if len(args) < 4  {
				panic("Flag 'f' requires a value.")
			}
			
			message, err := ioutil.ReadFile(args[3])
			branch, err := ioutil.ReadFile(".ssc/branch")

			if err != nil {
				panic(err)
			}

			tree := core.CreateTree()
			commit := core.Commit{Tree:tree, Date:time.Now().Format(time.RFC3339), Message:string(message), Branch:string(branch)}
			core.CreateCommit(commit)

		case "-h", "--help":
			print(core.CommitUsage)
		}

	case "log":
		if len(args) < 3 {
			panic("'log' command requires an int argument")
		}

		arg, err := strconv.ParseInt(args[2], 10, 64)

		if err != nil {
			panic(err)
		}

		core.Log(int(arg))
	
	case "help", "-h", "--help":
		print(core.Usage)

	default:
		print(core.Usage)
	}

}

