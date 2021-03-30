/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

// This module is used for small utility functions
// Not core functions such as `hashobject`

package utils

import (
	"sort"
	"os"
	"path/filepath"
	"strings"
)

// ExistInArray is used to check if a hash is in the log
func ExistInArray(hash string, log []string) bool {
	sort.Strings(log)
	i := sort.SearchStrings(log, hash)
	if i < len(log) && log[i] == hash {
		return true
	}
	return false
}

// ReverseArray is used to (duh) reverse an array
func ReverseArray(input []string) []string {
	if len(input) == 0 {
		return input
	}
	return append(ReverseArray(input[1:]), input[0])
}

// DeleteEmpty returns a new array without the empty strings/newline strings
func DeleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "\n" && str != "" {
			r = append(r, str)
		}
	}
	return r
}

// Create creates files and directories recursively (and returns a write object)
func Create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}
	return os.Create(p)
}

// GetFiles is used to get all file names (not full paths) in cwd
func GetFiles() []string {
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

// FileExists it used to check if file exists in cwd
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
