/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"ssc/utils"
	"strings"
)

// RevertTo reverts the CWD to the tree of the hash passed
// any uncommitted changes will be lost
func RevertTo(hash string) {
	// Get content of the commit
	get_commit := getContent(hash)
	get_tree := strings.Split(get_commit, "\n")[0][5:]

	// Get content of tree and split it into an array
	tree := getContent(get_tree)
	arr := strings.Split(tree, "\n")

	for _, content := range arr {
		// filenameToHash[0] = filename
		// filenameToHash[1] = filename
		filenameToHash := strings.Split(content, " ")

		// Get content of each object and write it to a new file in the CWD
		filecontent := getContent(filenameToHash[1])
		writer, err := utils.Create(string(filenameToHash[0]))

		writer.WriteString(filecontent)

		if err != nil {
			panic(err)
		}
	}
}
