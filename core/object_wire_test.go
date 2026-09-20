package core

import (
	"strings"
	"testing"
)

func TestWireObjectsMatchLocalStorage(t *testing.T) {
	inRepository(t, "main")
	writeTestFile(t, "file with spaces", "hello")
	hash := commitForTest(t, "wire")
	data, err := readObject(hash)
	if err != nil {
		t.Fatal(err)
	}
	actual, refs, err := InspectObject("commit", data)
	if err != nil || actual != hash || len(refs) != 1 || refs[0].Type != "tree" {
		t.Fatalf("%s %+v %v", actual, refs, err)
	}
	tree, err := readObject(refs[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	id, blobs, err := InspectObject("tree", tree)
	if err != nil || id != refs[0].Hash || len(blobs) != 1 {
		t.Fatalf("%s %+v %v", id, blobs, err)
	}
	blob, err := readObject(blobs[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err = InspectObject("blob", blob)
	if err != nil || id != blobs[0].Hash {
		t.Fatalf("%s %v", id, err)
	}
	for _, name := range []string{"../escape", "/absolute", ".ssc/branch", ".git/config", "a\\b", "a/../b", "a:b"} {
		if _, _, err := InspectObject("tree", []byte(name+" "+strings.Repeat("a", 40))); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestWireTreeRejectsMetadataAndCaseAliases(t *testing.T) {
	hash := strings.Repeat("a", 40)
	for _, tree := range []string{
		".SSC/branch " + hash, ".Git/config " + hash, "nested/.git/config " + hash,
		"File " + hash + "\nfile " + hash, "Dir/a " + hash + "\ndir/b " + hash,
		"trailing. " + hash, "trailing  " + hash,
	} {
		if _, _, err := InspectObject("tree", []byte(tree)); err == nil {
			t.Fatalf("accepted unsafe tree %q", tree)
		}
	}
}

func TestSnapshotsExcludeGitMetadata(t *testing.T) {
	inRepository(t, "main")
	writeTestFile(t, "source.go", "package main")
	writeTestFile(t, ".git/config", "must stay local")
	tree := CreateTree()
	data, err := readObject(tree)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), ".git") {
		t.Fatal("Git metadata entered the snapshot")
	}
	if _, refs, err := InspectObject("tree", data); err != nil || len(refs) != 1 {
		t.Fatal(refs, err)
	}
}
