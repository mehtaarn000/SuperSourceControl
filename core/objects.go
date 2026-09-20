/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"ssc/utils"
	"ssc/zlibutils"
	"strings"

	"golang.org/x/crypto/ripemd160"
)

// CreateTree creates a tree object
func CreateTree() string {
	// Get all file names in directory
	files := utils.GetFiles()

	// Create blobs using all file names
	rawtree := createBlobs(files)
	treelength := string(rune(len(rawtree)))

	/*Tree hash is calculated like:
	tree $LENGTHOFTREE $TREE
	*/

	tree := "tree " + treelength + rawtree

	// Calculate tree hash
	hasher := ripemd160.New()
	hasher.Write([]byte(tree))
	hash := hex.EncodeToString(hasher.Sum(nil))
	hashfile := ".ssc/objects/" + hash

	// Write tree to file
	ioutil.WriteFile(hashfile, []byte(rawtree), 0644)
	zlibutils.CompressFile(map[string]string{hashfile: hash})

	// Write tree hash to list of trees
	f, err := os.OpenFile(".ssc/trees", os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()

	if err != nil {
		utils.Exit(err)
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

		file = filepath.ToSlash(file)

		if strings.Contains(file, ".ssc") {
			continue
		}

		bloblength := len(string(raw))

		/*Blob hash is calculated like:
		blob $LENGTHOFBLOB $BLOB
		*/
		content := "blob " + string(rune(bloblength)) + string(raw)

		if err != nil {
			utils.Exit(err)
		}

		hasher := ripemd160.New()
		hasher.Write([]byte(content))

		hash := hex.EncodeToString(hasher.Sum(nil))
		hashfile := ".ssc/objects/" + hash

		// If the file doesn't exist, create it
		// Multiple files can point to the same object
		if !utils.FileExists(hashfile) {
			ioutil.WriteFile(hashfile, raw, 0644)
			zlibfiles[hashfile] = hash
		}

		// If last element in array, don't append string with newline
		if index != len(filenames)-1 {
			treestring += file + " " + hash + "\n"
		} else {
			treestring += file + " " + hash
		}
	}

	zlibutils.CompressFile(zlibfiles)
	return treestring
}
