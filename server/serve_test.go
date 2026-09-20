package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"--generate-token"}, &out); err != nil {
		t.Fatal(err)
	}
	var result map[string]string
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	token, err := hex.DecodeString(result["token"])
	if err != nil || len(token) != 32 {
		t.Fatal("invalid token")
	}
	digest := sha256.Sum256([]byte(result["token"]))
	if result["sha256"] != hex.EncodeToString(digest[:]) {
		t.Fatal("digest mismatch")
	}
}
func TestServeRejectsUnsafeOrIncompleteArguments(t *testing.T) {
	for _, args := range [][]string{
		{}, {"--config", "missing"},
		{"--config", "missing", "--data", t.TempDir(), "--listen", "0.0.0.0:8080"},
		{"--config", "missing", "--data", t.TempDir(), "--listen", "localhost:8080"},
		{"--config", "missing", "--data", t.TempDir(), "--tls-cert", "cert"},
		{"--generate-token", "--data", t.TempDir()},
	} {
		var out bytes.Buffer
		if err := Run(context.Background(), args, &out); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"--help"}, &out); err != nil || !strings.Contains(out.String(), "config") {
		t.Fatalf("%v %s", err, out.String())
	}
}
