/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"io/ioutil"
	"strings"
	"ssc/utils"
)

// Log lists the commits from the commitlog
func Log(commits int, reverse bool) {
	// Get commits
	bytescommitlog, err := ioutil.ReadFile(".ssc/commitlog")
	commitlog := strings.Split(string(bytescommitlog), "\n")
	commitlog = commitlog[:len(commitlog)-1]

	if commits > len(commitlog) {
		panic("Number of requested commits is too large")
	}

	// Get number of commits passed to function
	requested_commits := commitlog[len(utils.DeleteEmpty(commitlog))-commits:]
	if err != nil {
		panic(err)
	}

	// If the user doesn't specify the reverse option
	if !reverse {
		requested_commits = utils.ReverseArray(requested_commits)
	}

	for _, commit := range requested_commits {
		// Get content of each commit
		content := getContent(commit)
		split_content := strings.Split(content, "\n")

		// Slice the string to get the date and message
		date := split_content[1][5:]
		message := split_content[4]

		/*Output looks like:
		$COMMITHASH   $COMMITDATEANDTIME   $COMMITMESSAGE
		*/
		print(commit, "   ", date, "   ", message, "\n")
	}

	if err != nil {
		panic(err)
	}
}

// MaxLog gets all the commits and logs them
func MaxLog(reverse bool) {
	bytescommitlog, err := ioutil.ReadFile(".ssc/commitlog")
	numofcommits := len(strings.Split(string(bytescommitlog), "\n"))

	if reverse {
		Log(numofcommits-1, true)
		return
	}

	Log(numofcommits-1, false)

	if err != nil {
		panic(err)
	}
}
