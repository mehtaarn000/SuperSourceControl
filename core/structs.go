/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

// Commit is a commit object
type Commit struct {
	Format      int
	Parents     []string
	AuthorName  string
	AuthorEmail string
	Tree        string
	Date        string
	Message     string
	Branch      string
}
