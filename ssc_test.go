package main

import (
	"compress/zlib"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Run the real CLI in subprocesses so Exit and filesystem effects are exercised.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("SSC_TEST_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"ssc"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(2)
}

func TestCommitCLIIdentityParentsAndLog(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.Mkdir(repo, 0755); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	run := func(wantSuccess bool, input string, args ...string) string {
		t.Helper()
		command := exec.Command(executable, append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
		command.Dir = repo
		command.Env = append(os.Environ(), "SSC_TEST_PROCESS=1", "SSC_CONFIG_FILE="+filepath.Join(root, "config.json"))
		command.Stdin = strings.NewReader(input)
		data, err := command.CombinedOutput()
		if (err == nil) != wantSuccess {
			t.Fatalf("%v: %v\n%s", args, err, data)
		}
		return strings.TrimSpace(string(data))
	}
	// Configuration can be established outside a repository; legacy config works.
	run(true, "", "config", "-c", "authorName", "Example Developer")
	run(true, "", "config", "-c", "authorEmail", "dev@example.com")
	run(true, "", "init", "-b", "main")
	if err := ioutil.WriteFile(filepath.Join(repo, "file"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	readCommit := func(hash string) string {
		t.Helper()
		file, err := os.Open(filepath.Join(repo, ".ssc", "objects", hash))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		reader, err := zlib.NewReader(file)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		data, err := ioutil.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	first := run(true, "", "commit", "-m", "first")
	raw := readCommit(first)
	if strings.Contains(raw, "\nparent ") || !strings.Contains(raw, "\nauthor-name Example Developer\nauthor-email dev@example.com\n\nfirst") {
		t.Fatal(raw)
	}
	second := run(true, "second\n", "commit", "-p")
	second = second[len(second)-40:]
	if !strings.Contains(readCommit(second), "\nparent "+first+"\n") {
		t.Fatal("prompt commit has wrong parent")
	}
	messagePath := filepath.Join(root, "message.txt")
	if err := ioutil.WriteFile(messagePath, []byte("third\n\nbody"), 0644); err != nil {
		t.Fatal(err)
	}
	third := run(true, "", "commit", "-f", messagePath)
	if raw := readCommit(third); !strings.Contains(raw, "\nparent "+second+"\n") || !strings.HasSuffix(raw, "third\n\nbody") {
		t.Fatal(raw)
	}
	latest := third
	if runtime.GOOS != "windows" {
		editor := filepath.Join(root, "editor")
		if err := ioutil.WriteFile(editor, []byte("#!/bin/sh\nprintf 'fourth' > \"$1\"\n"), 0755); err != nil {
			t.Fatal(err)
		}
		latest = run(true, "", "commit", "-e", editor)
		if raw := readCommit(latest); !strings.Contains(raw, "\nparent "+third+"\n") || !strings.HasSuffix(raw, "fourth") {
			t.Fatal(raw)
		}
	}
	log := run(true, "", "log", "-m")
	if !strings.HasPrefix(log, latest) || !strings.Contains(log, "Example Developer <dev@example.com>") || !strings.Contains(log, "first") {
		t.Fatal(log)
	}
	reverse := run(true, "", "log", "-mr")
	if !strings.HasPrefix(reverse, first) {
		t.Fatal(reverse)
	}
	historyPath := filepath.Join(repo, ".ssc", "branches", "main", "commitlog")
	before, _ := ioutil.ReadFile(historyPath)
	run(false, "", "commit", "-f", filepath.Join(root, "missing.txt"))
	run(false, "", "log", "-n", "-1")
	run(false, "", "log", "-n")
	run(false, "", "log", "-n", "invalid")
	run(false, "", "config", "-c", "authorName")
	if err := ioutil.WriteFile(filepath.Join(root, "config.json"), []byte(`{"defaultBranch":"main"}`), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(false, "", "commit", "-m", "no identity")
	if !strings.Contains(out, "authorName") {
		t.Fatal(out)
	}
	after, _ := ioutil.ReadFile(historyPath)
	if string(before) != string(after) {
		t.Fatal("failed command advanced history")
	}
	run(true, "", "log", "-m") // Reading history does not require configured identity.
}

func TestServeCommandDoesNotRequireLocalRepository(t *testing.T) {
	root := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "must-not-be-created.json")
	command := exec.Command(executable, "-test.run=^TestCLIProcess$", "--", "serve", "--help")
	command.Dir = root
	command.Env = append(os.Environ(), "SSC_TEST_PROCESS=1", "SSC_CONFIG_FILE="+config)
	data, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(data), "listen") {
		t.Fatalf("%s %v", data, err)
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("serve touched author configuration")
	}
	if _, err := os.Stat(filepath.Join(root, ".ssc")); !os.IsNotExist(err) {
		t.Fatal("serve initialized a local repository")
	}
}
