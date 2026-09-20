package remote

import (
	"context"
	"os"
	"path/filepath"
	"ssc/core"
	"testing"
)

func TestLocalObjectVerification(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".ssc/objects", ".ssc/branches", ".ssc/tmp"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	r, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("hello")
	hash, _, _ := core.InspectObject("blob", raw)
	if err := r.saveObject(hash, "blob", raw); err != nil {
		t.Fatal(err)
	}
	got, err := r.readObject(hash, "blob")
	if err != nil || string(got) != "hello" {
		t.Fatal(string(got), err)
	}
	if _, err := r.readObject(hash, "tree"); err == nil {
		t.Fatal("wrong type accepted")
	}
	if err := os.WriteFile(r.metadata("objects", hash), []byte("corrupted"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.readObject(hash, "blob"); err == nil {
		t.Fatal("corrupt object accepted")
	}
	if _, err := loadGraph(context.Background(), []string{hash}, r.readObject); err == nil {
		t.Fatal("blob used as commit")
	}
}
func TestRejectBranchNamespaceCollisions(t *testing.T) {
	for _, refs := range []map[string]string{{"feature": "x", "feature/one": "x"}, {"Main": "x", "main": "x"}} {
		if err := validateBranches(refs); err == nil {
			t.Fatal("invalid branch namespace accepted")
		}
	}
}
