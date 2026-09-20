package remote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestURLValidation(t *testing.T) {
	for _, raw := range []string{"https://example.com/v1/repos/demo", "http://127.0.0.1:8080/v1/repos/demo", "http://[::1]:8080/v1/repos/demo/"} {
		if _, err := ValidateURL(raw); err != nil {
			t.Fatal(raw, err)
		}
	}
	for _, raw := range []string{"http://example.com/v1/repos/demo", "http://localhost/v1/repos/demo", "https://user:secret@example.com/v1/repos/demo", "https://example.com/v1/repos/demo?token=secret", "https://example.com/v1/repos/../private", "https://example.com/v1/repos/%64emo"} {
		if _, err := ValidateURL(raw); err == nil {
			t.Fatal("accepted", raw)
		}
	}
}
func TestRedirectsDoNotForwardTokens(t *testing.T) {
	leaked := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer source.Close()
	c, err := NewClient(source.URL+"/v1/repos/demo", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Refs(context.Background()); err == nil {
		t.Fatal("redirect accepted")
	}
	if leaked {
		t.Fatal("token forwarded")
	}
}
func TestRejectTamperedDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-SSC-Object-Type", "blob")
		w.Write([]byte("wrong"))
	}))
	defer srv.Close()
	c, _ := NewClient(srv.URL+"/v1/repos/demo", "secret")
	if _, err := c.Get(context.Background(), strings.Repeat("a", 40), "blob"); err == nil {
		t.Fatal("tampered object accepted")
	}
}
