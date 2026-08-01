package core

import "os"

func Init() {
	if _, err := os.Stat("./conf/app.ini"); err != nil {
		if os.IsExist(err) {
			print("Repository already exists.")
			os.Exit(0)
		}
	}
	os.MkdirAll(".ssc/branches", 0777)
	os.MkdirAll(".ssc/objects", 0777)
	os.MkdirAll(".ssc/tmp", 0777)

	f, err := os.Create(".ssc/branch")
	f.WriteString("master")

	if err != nil {
		panic(err)
	}

	os.Create(".ssc/commitlog")
	os.Create(".ssc/trees")
}