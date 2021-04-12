package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// Create creates files and directories recursively (and returns a write object)
func Create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 777); err != nil {
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

func RemoveEverything() {
	err := filepath.Walk("./", func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, ".ssc") {
			return nil
		}

		os.Remove(path)
		return nil
	})

	if err != nil {
		panic(err)
	}
}

// FileExists it used to check if file exists in cwd
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
