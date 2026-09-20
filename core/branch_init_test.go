package core

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func inRepository(t *testing.T, branch string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Error(err)
		}
	})
	if err := initRepository(branch); err != nil {
		t.Fatal(err)
	}
}

func writeTestFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path, []byte(value), 0644); err != nil {
		t.Fatal(err)
	}
}

func contents(t *testing.T, path string) string {
	t.Helper()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func snapshot(t *testing.T, message string) {
	t.Helper()
	branch, err := currentBranch()
	if err != nil {
		t.Fatal(err)
	}
	CreateCommit(Commit{AuthorName: "Test Developer", AuthorEmail: "test@example.com", Tree: CreateTree(), Date: "2026-09-17T12:00:00Z", Message: message, Branch: branch})
}

func TestBranchNames(t *testing.T) {
	for _, name := range []string{"master", "main", "feature/login", "release-2.1"} {
		if !validateBranchName(name) {
			t.Errorf("rejected valid name %q", name)
		}
	}
	for _, name := range []string{"", ".", "..", "../outside", "/absolute", "a/../b", "a//b", "a/", ".hidden", "a.lock", "a/b.lock", "a b", "a\nb", "a\\b", "a@{b", "a*", "a?", "a[", "a:", "commitlog", "a/commitlog"} {
		if validateBranchName(name) {
			t.Errorf("accepted invalid name %q", name)
		}
	}
}

func TestConfigurationBootstrapPreservesSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	t.Setenv("EDITOR", "editor with \"quotes\"")
	if err := ensureConfig(path); err != nil {
		t.Fatal(err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(contents(t, path)), &settings); err != nil {
		t.Fatal(err)
	}
	if settings["defaultBranch"] != "master" || settings["editor"] != os.Getenv("EDITOR") {
		t.Fatal(settings)
	}
	writeTestFile(t, path, `{"defaultBranch":"main"}`)
	if err := ensureConfig(path); err != nil {
		t.Fatal(err)
	}
	if contents(t, path) != `{"defaultBranch":"main"}` {
		t.Fatal("existing configuration changed")
	}
	t.Setenv("EDITOR", "")
	if !strings.Contains(defaultSettingsJSON(), `"editor": "vi"`) {
		t.Fatal("missing editor fallback")
	}
}

func TestInitCustomBranchAndReinitialization(t *testing.T) {
	inRepository(t, "feature/main")
	if contents(t, ".ssc/branch") != "feature/main" {
		t.Fatal("wrong active branch")
	}
	writeTestFile(t, "hello.txt", "hello")
	snapshot(t, "initial")
	before := contents(t, ".ssc/branches/feature/main/commitlog")
	if err := initRepository("master"); err == nil {
		t.Fatal("reinitialization accepted")
	}
	if contents(t, ".ssc/branch") != "feature/main" || contents(t, ".ssc/branches/feature/main/commitlog") != before {
		t.Fatal("reinitialization changed history")
	}
	if err := initRepository("../outside"); err == nil {
		t.Fatal("invalid initial branch accepted")
	}
}

func TestCreateBranchInheritsEntireHistory(t *testing.T) {
	inRepository(t, "main")
	if err := createBranch("empty"); err == nil {
		t.Fatal("branched before first commit")
	}
	writeTestFile(t, "file", "one")
	snapshot(t, "one")
	writeTestFile(t, "file", "two")
	snapshot(t, "two")
	if err := createBranch("feature/login"); err != nil {
		t.Fatal(err)
	}
	want, _ := readBranchLog("main")
	got, _ := readBranchLog("feature/login")
	if !reflect.DeepEqual(got, want) || len(got) != 2 {
		t.Fatalf("history = %v; want %v", got, want)
	}
	if err := createBranch("feature/login"); err == nil {
		t.Fatal("duplicate branch accepted")
	}
	got, _ = readBranchLog("feature/login")
	if !reflect.DeepEqual(got, want) {
		t.Fatal("duplicate creation erased history")
	}
	for _, name := range []string{"main/child", "feature", "../outside"} {
		if err := createBranch(name); err == nil {
			t.Errorf("accepted conflicting/invalid name %q", name)
		}
	}
}

func TestSwitchRestoresLatestSnapshotInBothDirections(t *testing.T) {
	inRepository(t, "main")
	writeTestFile(t, "folder/file with spaces.txt", "one")
	snapshot(t, "one")
	writeTestFile(t, "folder/file with spaces.txt", "two")
	snapshot(t, "two")
	if err := createBranch("feature"); err != nil {
		t.Fatal(err)
	}
	if err := switchBranch("feature"); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, "folder/file with spaces.txt", "feature")
	writeTestFile(t, "feature-only", "new")
	snapshot(t, "feature")
	if err := switchBranch("main"); err != nil {
		t.Fatal(err)
	}
	if contents(t, "folder/file with spaces.txt") != "two" {
		t.Fatal("did not restore latest main snapshot")
	}
	if _, err := os.Stat("feature-only"); !os.IsNotExist(err) {
		t.Fatal("extra file survived switch")
	}
	if err := switchBranch("feature"); err != nil {
		t.Fatal(err)
	}
	if contents(t, "folder/file with spaces.txt") != "feature" || contents(t, "feature-only") != "new" {
		t.Fatal("did not restore latest feature snapshot")
	}
	writeTestFile(t, "folder/file with spaces.txt", "uncommitted")
	if err := switchBranch("feature"); err != nil {
		t.Fatal(err)
	}
	if contents(t, "folder/file with spaces.txt") != "uncommitted" {
		t.Fatal("same-branch switch changed working files")
	}
}

func TestFailedSwitchPreservesActiveBranchAndFiles(t *testing.T) {
	inRepository(t, "main")
	writeTestFile(t, "file", "original")
	snapshot(t, "original")
	if err := createBranch("broken"); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, ".ssc/branches/broken/commitlog", strings.Repeat("a", 40)+"\n")
	for _, name := range []string{"missing", "broken", "../outside"} {
		if err := switchBranch(name); err == nil {
			t.Fatalf("switch to %q succeeded", name)
		}
		if contents(t, ".ssc/branch") != "main" || contents(t, "file") != "original" {
			t.Fatal("failed switch changed state")
		}
	}
}

func TestDeleteBranchProtectsCurrentAndOtherBranches(t *testing.T) {
	inRepository(t, "main")
	writeTestFile(t, "file", "data")
	snapshot(t, "initial")
	for _, name := range []string{"feature/one", "feature/two"} {
		if err := createBranch(name); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"main", "feature", "../outside", "missing"} {
		if err := deleteBranch(name); err == nil {
			t.Errorf("deleted protected branch %q", name)
		}
	}
	if err := deleteBranch("feature/one"); err != nil {
		t.Fatal(err)
	}
	if _, err := readBranchLog("feature/two"); err != nil {
		t.Fatal("sibling removed:", err)
	}
	if _, err := os.Stat(".ssc/branches/feature/one"); !os.IsNotExist(err) {
		t.Fatal("branch still exists")
	}
}
