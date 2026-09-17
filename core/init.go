/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"ssc/utils"
)

func Init(branch string) {
	if err := initRepository(branch); err != nil {
		utils.Exit(err)
	}
}

func initRepository(branch string) error {
	if !validateBranchName(branch) {
		return fmt.Errorf("invalid branch name: %q", branch)
	}
	// Exclusive creation prevents reinitialization from truncating history.
	if err := os.Mkdir(".ssc", 0755); err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			os.RemoveAll(".ssc")
		}
	}()
	for _, dir := range []string{filepath.Join(".ssc", "branches", branch), ".ssc/objects", ".ssc/tmp"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	for path, content := range map[string]string{
		".ssc/branch": branch,
		filepath.Join(".ssc", "branches", branch, "commitlog"): "",
		".ssc/trees": "",
	} {
		if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}
	complete = true
	return nil
}
