/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package zlibutils

import (
	"compress/zlib"
	"io"
	"os"
	"ssc/utils"
)

// DecompressFile takes a hash object as an input and decodes it
func DecompressFile(hash string) {
	zlibfile, err := os.Open(".ssc/objects/" + hash)

	if err != nil {
		utils.Exit(err)
	}

	reader, err := zlib.NewReader(zlibfile)
	if err != nil {
		utils.Exit(err)
	}

	hashobject := ".ssc/tmp/" + hash
	writer, err := os.Create(hashobject)

	if _, err = io.Copy(writer, reader); err != nil {
		utils.Exit(err)
	}
}
