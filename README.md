# SuperSourceControl
A local version control system written in Go. I built this as a way to introduce myself to Golang.

## Installation
```
git clone https://github.com/mehtaarn000/SuperSourceControl
go build ssc.go
```
Then, move the `ssc` executable to somewhere in your $PATH.

## Quickstart
To start a project:
```sh
mkdir testdir
cd testdir
ssc init
```

Then, create a file and commit:
```sh
touch "hello world" > helloworld.txt
ssc commit -m "Initial commit"
```

Log all commits:
```sh
ssc log -m
```

## Usage
Simpily run `ssc --help` to see usage.

## Update version
On Unix-like systems, two dependencies are required. `jq` and `curl`. You can use your operating systems package manager to install these.
When you are ready to update, run either of the following commands:

`ssc update`

or if you are using an older version of SuperSourceControl:

`curl https://raw.githubusercontent.com/mehtaarn000/SuperSourceControl/master/update.sh | sh`
