package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"ssc/server"
	"strings"
	"testing"
)

func TestTwoDevelopersClonePushAndPull(t *testing.T) {
	root := t.TempDir()
	store, err := server.NewStore(filepath.Join(root, "server"), []string{"demo"})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("team-token"))
	config := server.Config{Repositories: []string{"demo"}, Tokens: []server.TokenConfig{{Name: "team", SHA256: hex.EncodeToString(digest[:]), Repositories: map[string]string{"demo": "write"}}}}
	handler, err := server.NewHandler(store, config)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(handler)
	defer srv.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	run := func(dir string, wantSuccess bool, args ...string) string {
		t.Helper()
		identity := filepath.Join(root, "identity-"+filepath.Base(dir)+".json")
		settings, _ := json.Marshal(map[string]string{"authorName": filepath.Base(dir), "authorEmail": filepath.Base(dir) + "@example.com", "defaultBranch": "main"})
		if err := os.WriteFile(identity, settings, 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(executable, append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "SSC_TEST_PROCESS=1", "SSC_TOKEN=team-token", "SSC_CONFIG_FILE="+identity)
		data, err := cmd.CombinedOutput()
		if (err == nil) != wantSuccess {
			t.Fatalf("%s %v: %v\n%s", dir, args, err, data)
		}
		return strings.TrimSpace(string(data))
	}
	write := func(dir, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(dir string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, "file.txt"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	alice, bob := filepath.Join(root, "alice"), filepath.Join(root, "bob")
	endpoint := srv.URL + "/v1/repos/demo"
	run(root, true, "clone", endpoint, alice)
	write(alice, "first")
	first := run(alice, true, "commit", "-m", "first")
	run(alice, true, "push")
	run(root, true, "clone", endpoint, bob)
	if read(bob) != "first" {
		t.Fatal("clone did not restore snapshot")
	}
	write(alice, "second")
	second := run(alice, true, "commit", "-m", "second")
	run(alice, true, "push")
	run(bob, true, "pull")
	if read(bob) != "second" {
		t.Fatal("pull did not advance working files")
	}
	log := run(bob, true, "log", "-m")
	if !strings.Contains(log, "alice <alice@example.com>") || !strings.Contains(log, first) || !strings.Contains(log, second) {
		t.Fatal(log)
	}
	write(alice, "third")
	run(alice, true, "commit", "-m", "third")
	run(alice, true, "push")
	write(bob, "uncommitted")
	run(bob, false, "pull")
	if read(bob) != "uncommitted" {
		t.Fatal("dirty work lost")
	}
	write(bob, "second")
	run(bob, true, "pull")
	if read(bob) != "third" {
		t.Fatal("retry failed")
	}
	write(alice, "alice fourth")
	aliceTip := run(alice, true, "commit", "-m", "alice fourth")
	write(bob, "bob fourth")
	bobTip := run(bob, true, "commit", "-m", "bob fourth")
	run(bob, true, "push")
	run(alice, false, "push")
	run(alice, false, "pull")
	if read(alice) != "alice fourth" {
		t.Fatal("divergence overwrote local work")
	}
	localLog := run(alice, true, "log", "-m")
	if !strings.Contains(localLog, aliceTip) {
		t.Fatal(localLog)
	}
	refs, err := store.ListRefs("demo")
	if err != nil || refs["main"] != bobTip {
		t.Fatal(refs, err)
	}
	// Failed branch commands release the same operation lock used by sync.
	run(alice, false, "branch", "-s", "missing")
	run(alice, true, "remote")
	if _, err := os.Stat(filepath.Join(alice, ".ssc", "commit.lock")); !os.IsNotExist(err) {
		t.Fatal("failed command left lock")
	}
	for _, dir := range []string{alice, bob} {
		data, err := os.ReadFile(filepath.Join(dir, ".ssc", "remote.json"))
		if err != nil || strings.Contains(string(data), "team-token") {
			t.Fatal(fmt.Sprintf("credentials leaked: %v", err))
		}
	}
}
