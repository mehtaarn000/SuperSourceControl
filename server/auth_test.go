package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func testToken(name, token string, grants map[string]string) TokenConfig {
	h := sha256.Sum256([]byte(token))
	return TokenConfig{Name: name, SHA256: hex.EncodeToString(h[:]), Repositories: grants}
}
func testConfig() Config {
	return Config{Repositories: []string{"demo", "private"}, Tokens: []TokenConfig{
		testToken("writer", "write-secret", map[string]string{"demo": "write"}),
		testToken("reader", "read-secret", map[string]string{"demo": "read"}),
		testToken("private", "private-secret", map[string]string{"private": "write"}),
	}}
}
func TestAuthenticationAndGrants(t *testing.T) {
	a, err := NewAuthorizer(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, header := range []string{"", "Bearer wrong", "Basic write-secret", "Bearer", "Bearer write-secret extra"} {
		if _, ok := a.authenticate(header); ok {
			t.Fatalf("accepted %q", header)
		}
	}
	grants, ok := a.authenticate("Bearer read-secret")
	if !ok || grants["demo"] != "read" || grants["private"] != "" {
		t.Fatal(grants)
	}
	// Never retain mutable references to administrator configuration maps.
	c := testConfig()
	a, err = NewAuthorizer(c)
	if err != nil {
		t.Fatal(err)
	}
	c.Tokens[0].Repositories["private"] = "write"
	grants, _ = a.authenticate("Bearer write-secret")
	if grants["private"] != "" {
		t.Fatal("ACL mutated")
	}
}
func TestRejectInvalidAuthConfiguration(t *testing.T) {
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Tokens = nil },
		func(c *Config) { c.Tokens[0].Repositories["demo"] = "admin" },
		func(c *Config) { c.Tokens[0].Repositories["missing"] = "read" },
		func(c *Config) { c.Tokens[1].SHA256 = c.Tokens[0].SHA256 },
		func(c *Config) { c.Tokens[0].SHA256 = "plaintext-token" },
	} {
		c := testConfig()
		mutate(&c)
		if _, err := NewAuthorizer(c); err == nil {
			t.Fatal("invalid config accepted")
		}
	}
	path := filepath.Join(t.TempDir(), "config.json")
	data, _ := json.Marshal(testConfig())
	if err := ioutil.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path, append(data, []byte(" {}")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("trailing configuration accepted")
	}
}
