package server

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

type Config struct {
	Repositories []string      `json:"repositories"`
	Tokens       []TokenConfig `json:"tokens"`
}

type TokenConfig struct {
	Name         string            `json:"name"`
	SHA256       string            `json:"sha256"`
	Repositories map[string]string `json:"repositories"`
}

type credential struct {
	digest      [32]byte
	permissions map[string]string
}
type Authorizer struct{ credentials []credential }

// LoadConfig reads digests, never plaintext bearer tokens. Restart to reload ACLs.
func LoadConfig(path string) (Config, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if len(f) > 1<<20 {
		return Config{}, fmt.Errorf("configuration exceeds 1 MiB")
	}
	var c Config
	d := json.NewDecoder(bytes.NewReader(f))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	if err := d.Decode(new(interface{})); err != io.EOF {
		return c, fmt.Errorf("unexpected trailing configuration data")
	}
	if _, err := NewAuthorizer(c); err != nil {
		return c, err
	}
	return c, nil
}

func NewAuthorizer(c Config) (*Authorizer, error) {
	repos := map[string]bool{}
	for _, name := range c.Repositories {
		if !validRepoName(name) || repos[name] {
			return nil, fmt.Errorf("invalid or duplicate repository name")
		}
		repos[name] = true
	}
	if len(repos) == 0 || len(c.Tokens) == 0 {
		return nil, fmt.Errorf("repositories and authentication tokens are required")
	}
	a := &Authorizer{}
	digests := map[string]bool{}
	names := map[string]bool{}
	for _, token := range c.Tokens {
		digest, err := hex.DecodeString(token.SHA256)
		if err != nil || len(digest) != sha256.Size || token.SHA256 != strings.ToLower(token.SHA256) || digests[token.SHA256] {
			return nil, fmt.Errorf("token digests must be unique SHA-256 hex strings")
		}
		if strings.TrimSpace(token.Name) == "" || names[token.Name] {
			return nil, fmt.Errorf("token names must be nonempty and unique")
		}
		if len(token.Repositories) == 0 {
			return nil, fmt.Errorf("each token needs an explicit repository grant")
		}
		cred := credential{permissions: map[string]string{}}
		copy(cred.digest[:], digest)
		for repo, permission := range token.Repositories {
			if !repos[repo] || (permission != "read" && permission != "write") {
				return nil, fmt.Errorf("invalid repository grant for %q", token.Name)
			}
			cred.permissions[repo] = permission
		}
		names[token.Name], digests[token.SHA256] = true, true
		a.credentials = append(a.credentials, cred)
	}
	return a, nil
}

// authenticate compares fixed-size hashes, including when the supplied token is wrong.
func (a *Authorizer) authenticate(header string) (map[string]string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) > 4096 {
		return nil, false
	}
	digest := sha256.Sum256([]byte(parts[1]))
	var permissions map[string]string
	for _, cred := range a.credentials {
		if subtle.ConstantTimeCompare(digest[:], cred.digest[:]) == 1 {
			permissions = cred.permissions
		}
	}
	return permissions, permissions != nil
}
