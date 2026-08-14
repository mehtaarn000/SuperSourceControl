/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"os"
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

	// cwdfiles = files in current working directory (only names not full paths)
	cwdfiles := utils.GetFiles()
	filesintree := []string{}
	hashes := []string{}

	// filesintree and hashes arrays match indexes for files to hashes
	for _, filehash := range arr {
		items := strings.Split(filehash, " ")
		filesintree = append(filesintree, items[0])
		hashes = append(hashes, items[1])
	}

	// Get all files not in currentworking directory
	notincwd := utils.Intersection(filesintree, cwdfiles)

	for _, file := range notincwd {
		// Remove the file, else create it
		err := os.Remove(file)

		if err != nil {
			/*Index in filesintree
			Get object hash using the index from hashes array
			Create and write the data to the file*/
			index := utils.Find(filesintree, file)
			object := hashes[index]

			filecontent := getContent(object)
			writer, err := utils.Create(file)

			writer.WriteString(filecontent)

			if err != nil {
				utils.Exit(err)
			}
		}
	}

	// Create needed files
	for i, hash := range hashes {
		filecontent := getContent(hash)
		writer, err := utils.Create(string(filesintree[i]))

		writer.WriteString(filecontent)

		if err != nil {
			utils.Exit(err)
		}
	}
}
