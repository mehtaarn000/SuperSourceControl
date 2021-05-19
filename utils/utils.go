/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

// This module is used for small utility functions
// Not core functions such as `hashobject`

package utils

import (
	"sort"
	"fmt"
	"os"
)

func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

func Intersection(s1, s2 []string) (inter []string) {
	hash := make(map[string]bool)
	for _, e := range s1 {
		hash[e] = true
	}
	for _, e := range s2 {
		// If elements present in the hashmap then append intersection list.
		if !hash[e] {
			inter = append(inter, e)
		}
	}
	//Remove dups from slice.
	inter = removeDups(inter)
	return
}

//Remove dups from slice.
func removeDups(elements []string) (nodups []string) {
	encountered := make(map[string]bool)
	for _, element := range elements {
		if !encountered[element] {
			nodups = append(nodups, element)
			encountered[element] = true
		}
	}
	return
}

// ExistInArray is used to check if a hash is in the log
func ExistInArray(element string, array []string) bool {
	sort.Strings(array)
	i := sort.SearchStrings(array, element)
	if i < len(element) && array[i] == element {
		return true
	}
	return false
}

// ReverseArray is used to (duh) reverse an array
func ReverseArray(input []string) []string {
	if len(input) == 0 {
		return input
	}
	return append(ReverseArray(input[1:]), input[0])
}

// DeleteEmpty returns a new array without the empty strings/newline strings
func DeleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "\n" && str != "" {
			r = append(r, str)
		}
	}
	return r
}

func Exit(s interface{}) {
	fmt.Println(s)
	os.Exit(1)
}