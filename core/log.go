package core

import (
	"fmt"
	"ssc/utils"
	"strings"
)

// Branch logs remain display indexes so pre-format-2 history stays visible.
func Log(commits int, reverse bool) {
	branch, err := currentBranch()
	if err != nil {
		utils.Exit(err)
	}
	history, err := readBranchLog(branch)
	if err != nil {
		utils.Exit(err)
	}
	if commits < 0 || commits > len(history) {
		utils.Exit("Requested commit count is outside the available history")
	}
	selected := append([]string(nil), history[len(history)-commits:]...)
	if !reverse {
		for left, right := 0, len(selected)-1; left < right; left, right = left+1, right-1 {
			selected[left], selected[right] = selected[right], selected[left]
		}
	}
	for _, hash := range selected {
		c, err := ReadCommit(hash)
		if err != nil {
			utils.Exit(err)
		}
		fmt.Println(formatLogEntry(hash, c))
	}
}

func formatLogEntry(hash string, c Commit) string {
	author := "unknown author (legacy commit)"
	if c.Format == 2 {
		author = c.AuthorName + " <" + c.AuthorEmail + ">"
	}
	subject := strings.SplitN(c.Message, "\n", 2)[0]
	return fmt.Sprintf("%s   %s   %s   %s", hash, c.Date, author, subject)
}

func MaxLog(reverse bool) {
	branch, err := currentBranch()
	if err != nil {
		utils.Exit(err)
	}
	history, err := readBranchLog(branch)
	if err != nil {
		utils.Exit(err)
	}
	Log(len(history), reverse)
}
