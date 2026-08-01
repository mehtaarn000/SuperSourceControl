package core

import (
	"encoding/hex"
	"golang.org/x/crypto/ripemd160"
)

func hashObject(s string) string {
	stringtohash := "blob " + string(rune(len(s))) + s

	hasher := ripemd160.New()
	hasher.Write([]byte(stringtohash))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash
}

// PrintHash takes a string, turns it into a blob object, and hashes it
func PrintHash(s string) {
	value := hashObject(s)
	println(value)
}