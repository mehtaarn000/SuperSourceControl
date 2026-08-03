package core

import (
	"github.com/glenn-brown/golang-pkg-pcre/src/pkg/pcre"
	"os"
)

func CreateBranch(name string) {
	// Perl script validates branch name using regex
	newmatcher, err := pcre.Compile("^(?!/|.*([/.]\\.|//|@\\{|\\\\))[^\040\177 ~^:?*\\[]+(?<!\\.lock|[/.])$", 0)
	match := newmatcher.Matcher([]byte(name), 0).MatchString(name, 0)

	if !match {
		panic("Invalid branch name: '" + name + "'")
	}

	mkerr := os.Mkdir(".ssc/branches/" + name, 0777)
	_, mkerr = os.Create(".ssc/branches/" + name + "/commitlog")

	if err != nil {
		panic(err)
	} 

	if mkerr != nil {
		panic(mkerr)
	}
}