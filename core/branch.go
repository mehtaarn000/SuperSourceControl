package core

import (
	"os"
	"os/exec"
)

func CreateBranch(name string) {
	// Perl script validates branch name using regex
	out, err := exec.Command("perl", "core/branch_name.pl", name).Output()

	if string(out) == "true" {
		os.Mkdir(".ssc/branches/" + name, 0777)
	} else {
		panic("Invalid branch name: " + name)
	}

	if err != nil {
		panic(err)
	}
}