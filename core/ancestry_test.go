package core

import (
	"bytes"
	"compress/zlib"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func commitForTest(t *testing.T, message string) string {
	t.Helper()
	branch, err := currentBranch()
	if err != nil {
		t.Fatal(err)
	}
	c := exampleCommit()
	c.Branch = branch
	c.Tree = CreateTree()
	c.Message = message
	hash, err := createCommit(c)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func writeObjectForTest(t *testing.T, hash string, data []byte) {
	t.Helper()
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(filepath.Join(".ssc", "objects", hash), b.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestParentLinksAcrossBranches(t *testing.T) {
	inRepository(t, "main")
	writeTestFile(t, "file", "root")
	root := commitForTest(t, "root")
	rootCommit, err := ReadCommit(root)
	if err != nil || len(rootCommit.Parents) != 0 || rootCommit.AuthorEmail != "dev@example.com" {
		t.Fatalf("%+v %v", rootCommit, err)
	}
	if err := createBranch("feature"); err != nil {
		t.Fatal(err)
	}
	main := commitForTest(t, "main")
	if err := switchBranch("feature"); err != nil {
		t.Fatal(err)
	}
	feature := commitForTest(t, "feature")
	for _, hash := range []string{main, feature} {
		c, err := ReadCommit(hash)
		if err != nil {
			t.Fatal(err)
		}
		if len(c.Parents) != 1 || c.Parents[0] != root {
			t.Fatal("incorrect branch parent:", c.Parents)
		}
	}
	for _, pair := range []struct {
		a, b string
		want bool
	}{{root, main, true}, {root, feature, true}, {main, feature, false}, {feature, main, false}, {root, root, true}} {
		got, err := IsAncestor(pair.a, pair.b)
		if err != nil || got != pair.want {
			t.Fatalf("%+v: %v %v", pair, got, err)
		}
	}
	// Prove traversal follows objects rather than branch log positions.
	writeTestFile(t, ".ssc/branches/main/commitlog", "")
	if got, err := IsAncestor(root, main); !got || err != nil {
		t.Fatalf("%v %v", got, err)
	}
	// The schema can represent both parents of a future merge commit.
	merge := exampleCommit()
	merge.Tree = rootCommit.Tree
	merge.Parents = []string{main, feature}
	data, _ := encodeCommit(merge)
	hash := commitObjectID(data)
	writeObjectForTest(t, hash, data)
	if got, err := IsAncestor(main, hash); !got || err != nil {
		t.Fatalf("first parent: %v %v", got, err)
	}
	if got, err := IsAncestor(feature, hash); !got || err != nil {
		t.Fatalf("second parent: %v %v", got, err)
	}
}

func TestLegacyTipAndIncompleteAncestry(t *testing.T) {
	inRepository(t, "main")
	tree := CreateTree()
	legacy := []byte("tree " + tree + "\ndate 2021-03-22T17:22:04-05:00\nbranch main\n\nlegacy")
	old := strings.Repeat("b", 40)
	writeObjectForTest(t, old, legacy)
	writeTestFile(t, ".ssc/branches/main/commitlog", old+"\n")
	next := commitForTest(t, "new")
	if got, err := IsAncestor(old, next); !got || err != nil {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := IsAncestor(next, old); !errors.Is(err, ErrLegacyHistory) {
		t.Fatalf("expected incomplete ancestry: %v", err)
	}
	if stringMustReadObject(t, old) != string(legacy) {
		t.Fatal("legacy object rewritten")
	}
}

func stringMustReadObject(t *testing.T, hash string) string {
	t.Helper()
	data, err := readObject(hash)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFailedCommitDoesNotAdvanceHistory(t *testing.T) {
	inRepository(t, "main")
	root := commitForTest(t, "root")
	before := contents(t, ".ssc/branches/main/commitlog")
	c := exampleCommit()
	c.Tree = CreateTree()
	c.AuthorName = ""
	if _, err := createCommit(c); err == nil {
		t.Fatal("missing identity accepted")
	}
	c = exampleCommit()
	c.Tree = CreateTree()
	writeTestFile(t, ".ssc/commit.lock", "")
	if _, err := createCommit(c); err == nil {
		t.Fatal("concurrent writer accepted")
	}
	if contents(t, ".ssc/branches/main/commitlog") != before {
		t.Fatal("failed commit advanced tip")
	}
	raw, _ := readObject(root)
	raw = bytes.Replace(raw, []byte("Example Developer"), []byte("Other Developer"), 1)
	writeObjectForTest(t, root, raw)
	if _, err := ReadCommit(root); err == nil {
		t.Fatal("modified metadata accepted")
	}
}
