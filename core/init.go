/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import "os"

func Init() {
	// If .ssc repo already exists
	if _, err := os.Stat(".ssc"); err != nil {
		if os.IsExist(err) {
			print("Repository already exists.")
			os.Exit(0)
		}
	}

	/*Repository file structure:
	.ssc/
	|-- branches/
	|-- objects/
	|-- tmp/
	|-- branch (default = master)
	|-- commitlog
	|-- trees	
	*/

	// Create dirs
	err := os.MkdirAll(".ssc/branches", 0777)
	err = os.MkdirAll(".ssc/objects", 0777)
	err = os.MkdirAll(".ssc/tmp", 0777)

	// Create files
	f, err := os.Create(".ssc/branch")
	f.WriteString("master")

	f, err = os.Create(".ssc/commitlog")
	f, err = os.Create(".ssc/trees")

	if err != nil {
		panic(err)
	}
}
