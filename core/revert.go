/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"os"
	"path/filepath"
	"strings"
)

func create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}
	return os.Create(p)
}

func RevertTo(hash string) {
	get_commit := getContent(hash)
	get_tree := strings.Split(get_commit, "\n")[0][5:]
	tree := getContent(get_tree)

	arr := strings.Split(tree, "\n")

	for _, content := range arr {
		// hash[0] = filename
		// hash[1] = object hash
		filenameToHash := strings.Split(content, " ")

		filecontent := getContent(filenameToHash[1])
		writer, err := create(string(filenameToHash[0]))

		writer.WriteString(filecontent)

		if err != nil {
			panic(err)
		}
	}
}
