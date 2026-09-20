package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func commitBody(tree, message string, parents ...string) []byte {
	body := fmt.Sprintf("tree %s\ndate 2026-09-19T12:00:00Z\nbranch main\nformat 2\n", tree)
	for _, p := range parents {
		body += "parent " + p + "\n"
	}
	return []byte(body + "author-name Example Developer\nauthor-email dev@example.com\n\n" + message)
}
func testCommit(t *testing.T, s *Store, message string, parents ...string) string {
	t.Helper()
	blob := putTestObject(t, s, "blob", []byte(message))
	tree := putTestObject(t, s, "tree", []byte("file "+blob))
	return putTestObject(t, s, "commit", commitBody(tree, message, parents...))
}
func TestRefFastForwardAndCompleteGraph(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	root := testCommit(t, s, "root")
	next := testCommit(t, s, "next", root)
	fork := testCommit(t, s, "fork", root)
	if err := s.UpdateRef(ctx, "demo", "feature/main", "", root); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateRef(ctx, "demo", "feature/main", root, next); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{root, fork}, {next, fork}, {next, root}} {
		if err := s.UpdateRef(ctx, "demo", "feature/main", pair[0], pair[1]); !errors.Is(err, ErrConflict) {
			t.Fatal("unsafe update accepted:", err)
		}
	}
	missing := putTestObject(t, s, "commit", commitBody(strings.Repeat("a", 40), "missing"))
	if err := s.UpdateRef(ctx, "demo", "missing", "", missing); !errors.Is(err, ErrIncomplete) {
		t.Fatal(err)
	}
	blob := putTestObject(t, s, "blob", []byte("not a commit"))
	if err := s.UpdateRef(ctx, "demo", "blob", "", blob); !errors.Is(err, ErrIncomplete) {
		t.Fatal(err)
	}
	refs, err := s.ListRefs("demo")
	if err != nil || len(refs) != 1 || refs["feature/main"] != next {
		t.Fatalf("%v %v", refs, err)
	}
}
func TestConcurrentRefUpdatesHaveOneWinner(t *testing.T) {
	s := testStore(t)
	root := testCommit(t, s, "root")
	if err := s.UpdateRef(context.Background(), "demo", "main", "", root); err != nil {
		t.Fatal(err)
	}
	tips := []string{testCommit(t, s, "one", root), testCommit(t, s, "two", root)}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, tip := range tips {
		wg.Add(1)
		go func(tip string) {
			defer wg.Done()
			results <- s.UpdateRef(context.Background(), "demo", "main", root, tip)
		}(tip)
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("%d successes, %d conflicts", successes, conflicts)
	}
}
func TestRefHTTPRequiresExpectedTipAndWriteAccess(t *testing.T) {
	h, s := testHandler(t)
	tip := testCommit(t, s, "root")
	update := func(token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("PUT", "/v1/repos/demo/refs/main", bytes.NewBufferString(body))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	body := fmt.Sprintf(`{"old":"","new":%q}`, tip)
	if w := update("read-secret", body); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := update("write-secret", fmt.Sprintf(`{"new":%q}`, tip)); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := update("write-secret", body); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := update("write-secret", body); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if w := request(h, "GET", "/v1/repos/demo/refs", "read-secret", "", nil); w.Code != 200 || !strings.Contains(w.Body.String(), tip) {
		t.Fatal(w)
	}
}

func TestRefValidationRejectsNamespaceCollisionsAndWrongTypes(t *testing.T) {
	s := testStore(t)
	tip := testCommit(t, s, "root")
	ctx := context.Background()
	if err := s.UpdateRef(ctx, "demo", "feature/one", "", tip); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateRef(ctx, "demo", "feature", "", tip); !errors.Is(err, ErrConflict) {
		t.Fatal("namespace collision accepted", err)
	}
	if err := s.UpdateRef(ctx, "demo", "feature/one/child", "", tip); !errors.Is(err, ErrConflict) {
		t.Fatal("namespace collision accepted", err)
	}
	blob := putTestObject(t, s, "blob", []byte("not a tree"))
	wrong := putTestObject(t, s, "commit", commitBody(blob, "wrong type"))
	if err := s.UpdateRef(ctx, "demo", "bad", "", wrong); !errors.Is(err, ErrIncomplete) {
		t.Fatal("wrong object type accepted", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.UpdateRef(canceled, "demo", "canceled", "", tip); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestGraphReferenceLimit(t *testing.T) {
	s := testStore(t)
	blob := putTestObject(t, s, "blob", []byte("shared"))
	var tree strings.Builder
	for i := 0; i < MaxGraphObjects; i++ {
		fmt.Fprintf(&tree, "file%d %s\n", i, blob)
	}
	treeID := putTestObject(t, s, "tree", []byte(tree.String()))
	commit := putTestObject(t, s, "commit", commitBody(treeID, "too many references"))
	if err := s.UpdateRef(context.Background(), "demo", "main", "", commit); !errors.Is(err, ErrGraphLimit) {
		t.Fatal("graph limit not enforced", err)
	}
	refs, err := s.ListRefs("demo")
	if err != nil || len(refs) != 0 {
		t.Fatal("limit failure changed refs", refs, err)
	}
}
