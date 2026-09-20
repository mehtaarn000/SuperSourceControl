package core

import (
	"path/filepath"
	"testing"
)

func TestConfiguredAuthorAndLegacySettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SSC_CONFIG_FILE", path)
	writeTestFile(t, path, `{"defaultBranch":"main","aliases":{}}`)
	if _, _, err := ConfiguredAuthor(); err == nil {
		t.Fatal("missing author accepted")
	}
	ChangeSetting("authorName", "Example Developer")
	ChangeSetting("authorEmail", "dev@example.com")
	name, email, err := ConfiguredAuthor()
	if err != nil || name != "Example Developer" || email != "dev@example.com" {
		t.Fatalf("%q %q %v", name, email, err)
	}
	if GetSetting("defaultBranch") != "main" {
		t.Fatal("existing preference lost")
	}
	for _, data := range []string{`null`, `{}`, `{"authorName":42,"authorEmail":"dev@example.com"}`, `{"authorName":"name\nparent injected","authorEmail":"dev@example.com"}`} {
		writeTestFile(t, path, data)
		if _, _, err := ConfiguredAuthor(); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}
