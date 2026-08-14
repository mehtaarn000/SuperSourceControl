/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"io/ioutil"
	"ssc/utils"
	"strings"
)

// Log lists the commits from the commitlog
func Log(commits int, reverse bool) {
	// Get commits
	currentbranch, err := ioutil.ReadFile(".ssc/branch")

	bytescommitlog, err := ioutil.ReadFile(".ssc/branches/" + string(currentbranch) + "/commitlog")
	commitlog := strings.Split(string(bytescommitlog), "\n")
	commitlog = commitlog[:len(commitlog)-1]

	if commits > len(commitlog) {
		utils.Exit("Number of requested commits is too large")
	}

	//Get number of commits passed to function
	requested_commits := commitlog[len(utils.DeleteEmpty(commitlog))-commits:]
	if err != nil {
		utils.Exit(err)
	}

	//If the user doesn't specify the reverse option
	if !reverse {
		requested_commits = utils.ReverseArray(requested_commits)
	}

	for _, commit := range requested_commits {
		//Get content of each commit
		content := getContent(commit)
		split_content := strings.Split(content, "\n")

		//Slice the string to get the date and message
		date := split_content[1][5:]
		message := split_content[4]

		/*Output looks like:
		$COMMITHASH   $COMMITDATEANDTIME   $COMMITMESSAGE
		*/
		print(commit, "   ", date, "   ", message, "\n")
	}

	if err != nil {
		utils.Exit(err)
	}
}

// MaxLog gets all the commits and logs them
func MaxLog(reverse bool) {

	currentbranch, err := ioutil.ReadFile(".ssc/branch")

	bytescommitlog, err := ioutil.ReadFile(".ssc/branches/" + string(currentbranch) + "/commitlog")
	commits := strings.Split(string(bytescommitlog), "\n")
	commits = utils.DeleteEmpty(commits)
	numofcommits := len(commits)

	if reverse {
		Log(numofcommits, true)
		return
	}

	Log(numofcommits, false)

	if err != nil {
		utils.Exit(err)
	}
}
