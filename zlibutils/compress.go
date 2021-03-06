/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package zlibutils

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"io/ioutil"
	"os"
	"ssc/utils"
)

// CompressFile takes a filename list of strings and an outhash list of strings
func CompressFile(fileToHash map[string]string) {
	for file, hash := range fileToHash {
		rawfile, err := os.Open(file)

		if err != nil {
			utils.Exit(err)
		}
		defer rawfile.Close()

		// calculate the buffer size for rawfile
		info, _ := rawfile.Stat()
		var size int64 = info.Size()
		rawbytes := make([]byte, size)

		// read rawfile content into buffer
		buffer := bufio.NewReader(rawfile)
		_, err = buffer.Read(rawbytes)

		if err != nil {
			utils.Exit(err)
		}

		var buf bytes.Buffer
		writer := zlib.NewWriter(&buf)
		writer.Write(rawbytes)
		writer.Close()

		ioutil.WriteFile(".ssc/objects/"+hash, buf.Bytes(), info.Mode())
	}
}
