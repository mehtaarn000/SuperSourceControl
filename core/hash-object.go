/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"encoding/hex"
	"io/ioutil"
	"ssc/utils"
	"ssc/zlibutils"

	"golang.org/x/crypto/ripemd160"
)

// Utility function to turn a file/stdin into a blob and return the blob's hash
func hashObject(s string) string {
	stringtohash := "blob " + string(rune(len(s))) + s

	hasher := ripemd160.New()
	hasher.Write([]byte(stringtohash))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash
}

// PrintHash takes a string from stdin, turns it into a blob object, hashes it, and prints the generated hash
func PrintStdinHash(s string) {
	value := hashObject(s)
	println(value)
}

// WriteStdinHash takes a string from stdin, turns it into a blob object, hashes it, and writes the object to the ssc database
func WriteStdinHash(s string, quiet bool) {
	value := hashObject(s)
	ioutil.WriteFile(".ssc/objects/"+value, []byte(s), 0644)
	zlibutils.CompressFile(map[string]string{".ssc/objects/" + value: value})

	if !quiet {
		println(value)
	}
}

// PrintFileHash takes a filename, turns the file into a blob object, hashes it, and prints the generated hash
func PrintFileHash(filename string) {
	bytesfilecontent, err := ioutil.ReadFile(filename)
	filecontent := string(bytesfilecontent)

	println(hashObject(filecontent))

	if err != nil {
		utils.Exit(err)
	}
}

// WriteFileHash takes a filename, turns the file into a blob object, hashes it, and writes it to the ssc database
func WriteFileHash(filename string, quiet bool) {
	bytesfilecontent, err := ioutil.ReadFile(filename)
	filecontent := string(bytesfilecontent)

	hash := hashObject(filecontent)
	ioutil.WriteFile(".ssc/objects/"+hash, bytesfilecontent, 0644)

	zlibutils.CompressFile(map[string]string{".ssc/objects/" + hash: hash})

	if !quiet {
		println(hash)
	}

	if err != nil {
		utils.Exit(err)
	}
}
