/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"os/exec"
	"runtime"
	"io/ioutil"
	"ssc/utils"
)

const bash_script = `
JSON=$(curl -sL https://api.github.com/repos/mehtaarn000/SuperSourceControl/releases)
SSCPATH=$(which ssc)

ZIPBALL=$(echo "$JSON" | tr '\r\n' ' ' | jq -r '.[0] | .zipball_url')
curl -L "$ZIPBALL" > /tmp/ssc.zip
unzip /tmp/ssc.zip -d /tmp/SuperSourceControl

ZIPDIR=$(ls /tmp/SuperSourceControl)
FULLPATH="/tmp/SuperSourceControl/$ZIPDIR/ssc.go"
go build -o "/tmp/SuperSourceControl/$ZIPDIR/ssc" "$FULLPATH"
mv "/tmp/SuperSourceControl/$ZIPDIR/ssc" "$SSCPATH"
rm -rf "/tmp/SuperSourceControl"
`

// Update ssc on a Unix-like system
func Update() {
	switch runtime.GOOS {
	case "darwin", "freebsd", "openbsd", "linux", "netbsd":
		err := ioutil.WriteFile("./.ssc/tmp/update.sh", []byte(bash_script), 0777)
		cmd := exec.Command("sh", "./.ssc/tmp/update.sh")
		_, err = cmd.Output()

		if err != nil {
			utils.Exit(err)
		}
	
	default:
		utils.Exit("Sorry. Updates on your OS are not supported as of right now.")
	}
}