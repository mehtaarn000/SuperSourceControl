# SuperSourceControl
A local version control system written in Go. I built this as a way to introduce myself to Golang.

## Installation
```
git clone https://github.com/mehtaarn000/SuperSourceControl
go build ssc.go
```
Then, move the `ssc` executable to somewhere in your path.

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