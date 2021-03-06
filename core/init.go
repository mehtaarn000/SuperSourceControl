/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"os"
	"ssc/utils"
)

func Init(branch string) {
	// If .ssc repo already exists
	if _, err := os.Stat(".ssc"); err != nil {
		if os.IsExist(err) {
			print("Repository already exists.")
			os.Exit(0)
		}
	}

	/* Initial repository file structure:
	.ssc/
	|-- branches/
	    |-- master/
		    |-- commitlog
	|-- objects/
	|-- tmp/
	|-- branch (default = master)
	|-- trees
	*/

	// Create dirs
	err := os.MkdirAll(".ssc/branches/"+branch, 0777)
	err = os.MkdirAll(".ssc/objects", 0777)
	err = os.MkdirAll(".ssc/tmp", 0777)

	// Create files
	f, err := os.Create(".ssc/branch")
	f.WriteString("master")

	f, err = os.Create(".ssc/branches/" + branch + "/commitlog")
	f, err = os.Create(".ssc/trees")

	if err != nil {
		utils.Exit(err)
	}
}
