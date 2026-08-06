/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"io/ioutil"
	"os"
	"ssc/utils"
	"ssc/zlibutils"
	"strings"
)

// PrintContent prints an objects content as a string
func PrintContent(hash string) {
	content := getContent(hash)
	println(content)
}

// Utility function to get the content of an object
func getContent(hash string) string {
	zlibutils.DecompressFile(hash)
	content, err := ioutil.ReadFile(".ssc/tmp/" + hash)

	if err != nil {
		panic(err)
	}

	os.Remove(".ssc/tmp/" + hash)

	return (string(content))
}

// PrintType prints an objects type (Broken as of now)
func PrintType(hash string) {

	// If the hash is in commitlog, then it is a commit
	// If the hash is in the trees file, then it is a tree
	// Else it is a blob
	bytedata, err := ioutil.ReadFile(".ssc/commitlog")

	if err != nil {
		panic(err)
	}

	commitlog := strings.Split(string(bytedata), "\n")
	check := utils.ExistInArray(hash, commitlog)

	if check == true {
		println("commit")
		os.Exit(0)
	}

	blobbytes, err := ioutil.ReadFile(".ssc/blobs")

	blobs := strings.Split(string(blobbytes), "\n")
	check2 := utils.ExistInArray(hash, blobs)

	if check2 == true {
		println("blob")
		os.Exit(0)
	}

	treebytes, err := ioutil.ReadFile(".ssc/trees")

	trees := strings.Split(string(treebytes), "\n")
	check3 := utils.ExistInArray(hash, trees)

	if check3 == true {
		println("tree")
		os.Exit(0)
	}

	//panic("Object with hash '" + hash + "' does not exist.")
}

// PrintSize prints an decoded objects size (basically the file itself)
func PrintSize(hash string) {
	zlibutils.DecompressFile(hash)
	file, err := os.Stat(".ssc/tmp/" + hash)

	if err != nil {
		panic(err)
	}

	println(file.Size())
	os.Remove(".ssc/tmp/" + hash)
}

// PrintZlibSize prints a zlib encoded objects size
func PrintZlibSize(hash string) {
	file, err := os.Stat(".ssc/objects/" + hash)

	if err != nil {
		panic(err)
	}

	println(file.Size())
}
