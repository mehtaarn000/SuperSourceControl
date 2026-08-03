/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

const Usage = `Usage of ssc:
Standard:
	commit [flags] (argument) [options] Create a commit
	log [flags] (argument) List recent commits
	init Initilize a repository
	revert [flags] (argument) Revert repository to previous commit

Inner:
	cat-file [flags] (argument) Get information about ssc objects
	hash-object [flags] (argument) Create ssc objects`

const CommitUsage = `Usage of ssc commit:
Flags:
	-m [message] Write a commit message
	-p Input a commit message
	-e [editor] Specify an editor to write your commit message with
	-f [file] Read a commit message from a file
	-h Print this message`

const CatFileUsage = `Usage of ssc cat-file:
Flags:
	-s, --size [object hash] Print size of a decoded object
	-c, --content [object hash] Print an decoded object's content
	-t, --type [object hash] Print an objects type
	-z, --zlib-size [object hash] Print size of an encoded object
	-h, --help Print this message`

const LogUsage = `Usage of ssc log:
Flags:
	-n, --number [number of commits (int)] Print x many commits with the most recent being on the top
	-r, --reverse [number of commits (int)] Print x many commits with the most recent being on the bottom
	-h, --help Print this message`

const HashObjectUsage = `Usage of ssc hash-object:
Flags:
	-s, --stdin [string to hash] Print hash of stdin
	-ws, --write-stdin [string to hash and write to db] Write hash of stdin to ssc database
	-f, --file [filename] Print hash of file
	-wf, --write-file [filename] Write hash of file to ssc database
	-h, --help Print this message`

const RevertUsage = `Usage of ssc revert:
Flags:
	-n [commit hash] Revert to hash
	-h, --help Print this message`

const BranchUsage = `Usage of ssc branch:
Flags:
	-n, --new [branch name] Create branch with argument as name
	-ns, --new-switch [branch name] Create and switch to branch with argument as name
	-s, --switch [branch name] Switch branches
	-d, -D, --delete [branch name] [--force] Delete a branch (without the force option, ssc prompts the user if they want to proceed with the action)
	-h, --help Print this message`