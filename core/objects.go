/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"encoding/hex"
	"golang.org/x/crypto/ripemd160"
	"io/ioutil"
	"os"
	"path/filepath"
	"ssc/zlibutils"
	"strings"
)

func get_files() []string {
	var files []string

	err := filepath.Walk("./", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.Contains(path, ".ssc") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		panic(err)
	}

	return files
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// CreateCommit creates a commit, compresses it, and writes it to a file.
func CreateCommit(c Commit) {
	commit := ""
	commit += "tree " + c.Tree + "\n"
	commit += "date " + c.Date + "\n"
	commit += "branch " + c.Branch + "\n\n"
	commit += c.Message

	commitlen := string(rune(len(commit)))
	lenoflen := len(commitlen)
	commit = "commit " + commitlen + commit

	hasher := ripemd160.New()
	hasher.Write([]byte(commit))
	hash := hex.EncodeToString(hasher.Sum(nil))
	filename := ".ssc/objects/" + hash
	writer, err := os.Create(filename)

	if err != nil {
		panic(err)
	}

	writer.WriteString(commit[7+lenoflen:])
	zlibutils.CompressFile(map[string]string{filename: hash})

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()

	f.WriteString(hash + "\n")

	writeToLog, err := os.OpenFile(".ssc/commitlog", os.O_APPEND|os.O_WRONLY, 0644)
	defer writeToLog.Close()

	writeToLog.WriteString(hash + "\n")

	println(hash)
}

// CreateTree creates a tree object
func CreateTree() string {
	files := get_files()

	rawtree := createBlobs(files)
	treelength := string(rune(len(rawtree)))
	tree := "tree " + treelength + rawtree

	hasher := ripemd160.New()
	hasher.Write([]byte(tree))
	hash := hex.EncodeToString(hasher.Sum(nil))
	hashfile := ".ssc/objects/" + hash

	ioutil.WriteFile(hashfile, []byte(rawtree), 0644)
	zlibutils.CompressFile(map[string]string{hashfile: hash})

	f, err := os.OpenFile(".ssc/trees", os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()

	if err != nil {
		panic(err)
	}

	f.WriteString(hash + "\n")

	return hash
}

func createBlobs(filenames []string) string {
	// init
	zlibfiles := map[string]string{}
	treestring := ""

	// Creates blobs and string for tree object
	for index, file := range filenames {
		raw, err := ioutil.ReadFile(file)

		if strings.Contains(file, ".ssc") {
			continue
		}

		bloblength := len(string(raw))
		content := "blob " + string(rune(bloblength)) + string(raw) // Hash = string "blob" + space + len of blob + raw content

		if err != nil {
			panic(err)
		}

		hasher := ripemd160.New()
		hasher.Write([]byte(content))

		hash := hex.EncodeToString(hasher.Sum(nil))
		hashfile := ".ssc/objects/" + hash

		if !fileExists(hashfile) {
			ioutil.WriteFile(hashfile, raw, 0644)
			zlibfiles[hashfile] = hash
		}

		if index != len(filenames)-1 {
			treestring += file + " " + hash + "\n"
		} else {
			treestring += file + " " + hash
		}
	}

	zlibutils.CompressFile(zlibfiles)
	return treestring
}
