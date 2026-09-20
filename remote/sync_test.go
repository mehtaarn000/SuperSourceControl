package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"ssc/core"
	"ssc/server"
	"strings"
	"sync/atomic"
	"testing"
)

func syncServer(t *testing.T) (*server.Store, *Client, *httptest.Server) {
	t.Helper()
	s, err := server.NewStore(t.TempDir(), []string{"demo"})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("test-token"))
	cfg := server.Config{Repositories: []string{"demo"}, Tokens: []server.TokenConfig{{Name: "tester", SHA256: hex.EncodeToString(digest[:]), Repositories: map[string]string{"demo": "write"}}}}
	readDigest := sha256.Sum256([]byte("read-token"))
	cfg.Tokens = append(cfg.Tokens, server.TokenConfig{Name: "reader", SHA256: hex.EncodeToString(readDigest[:]), Repositories: map[string]string{"demo": "read"}})
	h, err := server.NewHandler(s, cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := NewClient(srv.URL+"/v1/repos/demo", "test-token")
	if err != nil {
		t.Fatal(err)
	}
	return s, c, srv
}
func addServerCommit(t *testing.T, s *server.Store, files map[string]string, parents ...string) string {
	t.Helper()
	var tree strings.Builder
	// Sort to make fixtures deterministic.
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data := []byte(files[name])
		hash, _, _ := core.InspectObject("blob", data)
		if err := s.PutObject("demo", hash, "blob", data); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&tree, "%s %s\n", name, hash)
	}
	treeID, _, err := core.InspectObject("tree", []byte(tree.String()))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PutObject("demo", treeID, "tree", []byte(tree.String())); err != nil {
		t.Fatal(err)
	}
	data := fmt.Sprintf("tree %s\ndate 2026-09-20T12:00:00Z\nbranch main\nformat 2\n", treeID)
	for _, parent := range parents {
		data += "parent " + parent + "\n"
	}
	data += "author-name Developer\nauthor-email dev@example.com\n\nmessage"
	hash, _, err := core.InspectObject("commit", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PutObject("demo", hash, "commit", []byte(data)); err != nil {
		t.Fatal(err)
	}
	return hash
}
func updateServer(t *testing.T, s *server.Store, old, next string) {
	t.Helper()
	if err := s.UpdateRef(context.Background(), "demo", "main", old, next); err != nil {
		t.Fatal(err)
	}
}

func TestClonePullAndDirtyProtection(t *testing.T) {
	s, c, _ := syncServer(t)
	ctx := context.Background()
	first := addServerCommit(t, s, map[string]string{"file": "one", "nested/old": "old"})
	updateServer(t, s, "", first)
	if err := s.UpdateRef(ctx, "demo", "feature/login", "", first); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "clone")
	if err := Clone(ctx, c, root, ""); err != nil {
		t.Fatal(err)
	}
	r, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := r.Branch(); got != "main" {
		t.Fatal(got)
	}
	if _, err := r.History("feature/login"); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(r.metadata("remote.json")); strings.Contains(string(raw), "test-token") {
		t.Fatal("token persisted")
	}
	if err := Clone(ctx, c, root, ""); err == nil {
		t.Fatal("existing destination overwritten")
	}
	second := addServerCommit(t, s, map[string]string{"file/child": "two", "nested": "now a file"}, first)
	updateServer(t, s, first, second)
	if err := os.WriteFile(filepath.Join(root, "untracked"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Pull(ctx, r, c); err == nil {
		t.Fatal("untracked files overwritten")
	}
	if history, _ := r.History("main"); tip(history) != first {
		t.Fatal(history)
	}
	os.Remove(filepath.Join(root, "untracked"))
	if err := os.WriteFile(filepath.Join(root, "file"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Pull(ctx, r, c); err == nil {
		t.Fatal("dirty file overwritten")
	}
	os.WriteFile(filepath.Join(root, "file"), []byte("one"), 0644)
	if err := Pull(ctx, r, c); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "file", "child")); string(content) != "two" {
		t.Fatal(string(content))
	}
	if content, _ := os.ReadFile(filepath.Join(root, "nested")); string(content) != "now a file" {
		t.Fatal(string(content))
	}
	if history, _ := r.History("main"); tip(history) != second {
		t.Fatal(history)
	}
}

func TestPullRollbackRestoresFilesAndTip(t *testing.T) {
	s, c, _ := syncServer(t)
	first := addServerCommit(t, s, map[string]string{"file": "old"})
	updateServer(t, s, "", first)
	root := filepath.Join(t.TempDir(), "clone")
	if err := Clone(context.Background(), c, root, ""); err != nil {
		t.Fatal(err)
	}
	r, _ := Open(root)
	history, _ := r.History("main")
	calls := 0
	err := r.applyPull("main", history, map[string][]byte{"file": []byte("old")}, map[string][]byte{"file": []byte("new")}, func(a, b string) error {
		calls++
		if calls == 2 {
			return fmt.Errorf("injected disk failure")
		}
		return os.Rename(a, b)
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	content, _ := os.ReadFile(filepath.Join(root, "file"))
	if string(content) != "old" {
		t.Fatal("rollback lost original")
	}
	if now, _ := r.History("main"); tip(now) != first {
		t.Fatal(now)
	}
	if _, err := os.Stat(r.metadata("sync-pending")); !os.IsNotExist(err) {
		t.Fatal("rollback left pending marker")
	}
}

func TestFailedCloneDoesNotPublishDestination(t *testing.T) {
	s, c, srv := syncServer(t)
	first := addServerCommit(t, s, map[string]string{"file": "one"})
	updateServer(t, s, "", first)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/objects/") {
			w.Header().Set("X-SSC-Object-Type", "commit")
			w.Write([]byte("tampered"))
			return
		}
		srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer bad.Close()
	// A valid ref advertisement followed by corrupt object data must publish nothing.
	c, _ = NewClient(bad.URL+"/v1/repos/demo", "test-token")
	dest := filepath.Join(t.TempDir(), "missing")
	if err := Clone(context.Background(), c, dest, ""); err == nil {
		t.Fatal("bad clone succeeded")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("failed clone published files")
	}
}

func TestPushUploadsOnlyMissingObjects(t *testing.T) {
	s, c, srv := syncServer(t)
	first := addServerCommit(t, s, map[string]string{"file": "one"})
	updateServer(t, s, "", first)
	root := filepath.Join(t.TempDir(), "clone")
	if err := Clone(context.Background(), c, root, ""); err != nil {
		t.Fatal(err)
	}
	r, _ := Open(root)
	// A new branch can reuse all existing objects.
	branchDir := r.metadata("branches", "new")
	os.Mkdir(branchDir, 0755)
	os.WriteFile(filepath.Join(branchDir, "commitlog"), []byte(first+"\n"), 0644)
	os.WriteFile(r.metadata("branch"), []byte("new"), 0644)
	var uploads atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "PUT" && strings.Contains(req.URL.Path, "/objects/") {
			uploads.Add(1)
		}
		target, _ := http.NewRequestWithContext(req.Context(), req.Method, srv.URL+req.URL.Path, req.Body)
		target.Header = req.Header.Clone()
		resp, err := http.DefaultClient.Do(target)
		if err != nil {
			t.Error(err)
			w.WriteHeader(502)
			return
		}
		defer resp.Body.Close()
		for k, values := range resp.Header {
			w.Header()[k] = values
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
	defer proxy.Close()
	c, _ = NewClient(proxy.URL+"/v1/repos/demo", "test-token")
	if err := Push(context.Background(), r, c); err != nil {
		t.Fatal(err)
	}
	if uploads.Load() != 0 {
		t.Fatal("existing objects reuploaded")
	}
	refs, _ := s.ListRefs("demo")
	if refs["new"] != first {
		t.Fatal(refs)
	}
}

func TestReadOnlyClientCanCloneAndPullButNotPush(t *testing.T) {
	s, _, srv := syncServer(t)
	ctx := context.Background()
	first := addServerCommit(t, s, map[string]string{"file": "one"})
	updateServer(t, s, "", first)
	reader, err := NewClient(srv.URL+"/v1/repos/demo", "read-token")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "reader")
	if err := Clone(ctx, reader, root, ""); err != nil {
		t.Fatal(err)
	}
	r, _ := Open(root)
	second := addServerCommit(t, s, map[string]string{"file": "two"}, first)
	updateServer(t, s, first, second)
	if err := Pull(ctx, r, reader); err != nil {
		t.Fatal(err)
	}
	dir := r.metadata("branches", "new")
	os.Mkdir(dir, 0755)
	os.WriteFile(filepath.Join(dir, "commitlog"), []byte(second+"\n"), 0644)
	os.WriteFile(r.metadata("branch"), []byte("new"), 0644)
	if err := Push(ctx, r, reader); err == nil {
		t.Fatal("read-only push accepted")
	}
	refs, _ := s.ListRefs("demo")
	if refs["new"] != "" {
		t.Fatal("unauthorized branch created")
	}
}

func TestPullRejectsSymlinksAndPreservesGitDirectory(t *testing.T) {
	s, c, _ := syncServer(t)
	ctx := context.Background()
	first := addServerCommit(t, s, map[string]string{"file": "one"})
	updateServer(t, s, "", first)
	root := filepath.Join(t.TempDir(), "clone")
	if err := Clone(ctx, c, root, ""); err != nil {
		t.Fatal(err)
	}
	r, _ := Open(root)
	external := filepath.Join(t.TempDir(), "external")
	os.WriteFile(external, []byte("outside"), 0644)
	os.Remove(filepath.Join(root, "file"))
	if err := os.Symlink(external, filepath.Join(root, "file")); err != nil {
		t.Skip(err)
	}
	second := addServerCommit(t, s, map[string]string{"file": "two"}, first)
	updateServer(t, s, first, second)
	if err := Pull(ctx, r, c); err == nil {
		t.Fatal("symlink accepted")
	}
	if data, _ := os.ReadFile(external); string(data) != "outside" {
		t.Fatal("external file changed")
	}
	os.Remove(filepath.Join(root, "file"))
	os.WriteFile(filepath.Join(root, "file"), []byte("one"), 0644)
	os.Mkdir(filepath.Join(root, ".git"), 0755)
	os.WriteFile(filepath.Join(root, ".git", "config"), []byte("preserve"), 0644)
	if err := Pull(ctx, r, c); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(root, ".git", "config")); string(data) != "preserve" {
		t.Fatal("Git metadata overwritten")
	}
}
