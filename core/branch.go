package core

import (
	"github.com/glenn-brown/golang-pkg-pcre/src/pkg/pcre"
	"os"
)

func CreateBranch(name string) {
	if _, err := os.Stat(".ssc/branches/" + name); err != nil {
		if os.IsExist(err) {
			panic("Branch '" + name + "' already exists.")
		}
	}

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

func SwitchBranch(name string) {
	if _, err := os.Stat(".ssc/branches/" + name); err != nil {
		if os.IsNotExist(err) {
			panic("Branch '" + name + "' does not exist.")
		}
	}

	writer, err := os.Create(".ssc/branch")
	writer.WriteString(name)

	if err != nil {
		panic(err)
	}

	println("Switched to branch '" + name + "'")
}