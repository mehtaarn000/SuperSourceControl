package core

import (
	"io/ioutil"
	"strings"
)

func delete_empty (s []string) []string {
	var r []string
	for _, str := range s {
        if str != "\n" {
            r = append(r, str)
        }
	}
	return r
}

func reverseArray(input []string) []string {
    if len(input) == 0 {
        return input
    }
    return append(reverseArray(input[1:]), input[0]) 
}

// Log lists the commits from the commitlog
func Log(commits int, reverse bool) {
	bytescommitlog, err := ioutil.ReadFile(".ssc/commitlog")
	commitlog := strings.Split(string(bytescommitlog), "\n")
	commitlog = commitlog[:len(commitlog)-1]

	if commits > len(commitlog) {
		panic("Number of requested commits is too large")
	}

	requested_commits := commitlog[len(delete_empty(commitlog))-commits:]
	if err != nil {
		panic(err)
	}

	if !reverse {
		requested_commits = reverseArray(requested_commits)
	}

	for _, commit := range requested_commits {
		content := getContent(commit)
		split_content := strings.Split(content, "\n")

		date := split_content[1][5:]
		message := split_content[4]
		print(commit, "   ", date, "   ", message, "\n")
	}

	if err != nil {
		panic(err)
	}
}

