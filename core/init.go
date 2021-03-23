package core

import "os"

func Init() {
	os.MkdirAll(".ssc/branches", 0644)
	os.MkdirAll(".ssc/objects", 0644)
	os.MkdirAll(".ssc/tmp", 0644)
	os.Create(".ssc/branch")
	os.Create(".ssc/commitlog")
	os.Create(".ssc/trees")
}