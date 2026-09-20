package core

import (
	"reflect"
	"strings"
	"testing"
)

func exampleCommit() Commit {
	return Commit{Format: 2, Tree: strings.Repeat("a", 40), Date: "2026-09-19T12:00:00Z", Branch: "main", AuthorName: "Example Developer", AuthorEmail: "dev@example.com", Message: "subject\n\nbody\n"}
}

func TestCommitFormatRoundTrip(t *testing.T) {
	for _, parents := range [][]string{nil, {strings.Repeat("b", 40)}, {strings.Repeat("b", 40), strings.Repeat("c", 40)}} {
		want := exampleCommit()
		want.Parents = parents
		data, err := encodeCommit(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeCommit(data)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}

func TestLegacyCommitDecoding(t *testing.T) {
	data := "tree " + strings.Repeat("a", 40) + "\ndate 2021-03-22T17:22:04-05:00\nbranch master\n\nold message\nsecond line"
	got, err := decodeCommit([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != 1 || len(got.Parents) != 0 || got.AuthorName != "" || got.Message != "old message\nsecond line" {
		t.Fatal(got)
	}
}

func TestCommitMetadataValidationAndHashing(t *testing.T) {
	original := exampleCommit()
	raw, _ := encodeCommit(original)
	for _, mutate := range []func(*Commit){
		func(c *Commit) { c.AuthorName = "different" },
		func(c *Commit) { c.AuthorEmail = "other@example.com" },
		func(c *Commit) { c.Parents = []string{strings.Repeat("b", 40)} },
	} {
		changed := original
		mutate(&changed)
		data, err := encodeCommit(changed)
		if err != nil {
			t.Fatal(err)
		}
		if commitObjectID(data) == commitObjectID(raw) {
			t.Fatal("metadata not included in hash")
		}
	}
	for _, mutate := range []func(*Commit){
		func(c *Commit) { c.AuthorName = "\nparent injected" },
		func(c *Commit) { c.AuthorEmail = "" },
		func(c *Commit) { c.Parents = []string{"../outside"} },
		func(c *Commit) { c.Parents = []string{strings.Repeat("b", 40), strings.Repeat("b", 40)} },
	} {
		changed := original
		mutate(&changed)
		if _, err := encodeCommit(changed); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
	for _, bad := range []string{strings.Replace(string(raw), "format 2", "format 3", 1), strings.Replace(string(raw), "format 2", "format 2\nformat 2", 1), "not a commit"} {
		if _, err := decodeCommit([]byte(bad)); err == nil {
			t.Fatal("malformed commit accepted")
		}
	}
}
