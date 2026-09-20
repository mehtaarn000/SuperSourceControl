package server

import (
	"bytes"
	"errors"
	"net/http/httptest"
	"ssc/core"
	"strings"
	"testing"
)

func testHandler(t *testing.T) (*Handler, *Store) {
	t.Helper()
	s := testStore(t)
	h, err := NewHandler(s, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	return h, s
}
func request(h *Handler, method, url, token, kind string, body []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, url, bytes.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if kind != "" {
		r.Header.Set("X-SSC-Object-Type", kind)
		r.Header.Set("Content-Type", "application/octet-stream")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestObjectHTTPPermissionsAndRoundTrip(t *testing.T) {
	h, _ := testHandler(t)
	hash, _, _ := core.InspectObject("blob", []byte("hello"))
	url := "/v1/repos/demo/objects/" + hash
	for _, tc := range []struct {
		method, token string
		want          int
	}{{"GET", "", 401}, {"GET", "wrong", 401}, {"GET", "private-secret", 404}, {"PUT", "read-secret", 403}, {"PUT", "write-secret", 204}, {"PUT", "write-secret", 204}, {"GET", "read-secret", 200}, {"HEAD", "read-secret", 200}, {"DELETE", "write-secret", 405}} {
		w := request(h, tc.method, url, tc.token, "blob", []byte("hello"))
		if w.Code != tc.want {
			t.Fatalf("%+v: %d %s", tc, w.Code, w.Body.String())
		}
		if tc.method == "GET" && tc.want == 200 && (w.Body.String() != "hello" || w.Header().Get("X-SSC-Object-Type") != "blob") {
			t.Fatal(w)
		}
		if tc.method == "HEAD" && w.Body.Len() != 0 {
			t.Fatal("HEAD returned a body")
		}
	}
	w := request(h, "PUT", url, "write-secret", "blob", []byte("bad"))
	if w.Code != 400 {
		t.Fatal(w)
	}
	w = request(h, "GET", url, "read-secret", "", nil)
	if w.Body.String() != "hello" {
		t.Fatal("invalid upload changed object")
	}
}
func TestHTTPRejectsOversizeAndTraversal(t *testing.T) {
	h, _ := testHandler(t)
	for _, url := range []string{"/v1/repos/demo/../private/objects/x", "/v1/repos/demo/objects/../../secret"} {
		if w := request(h, "GET", url, "write-secret", "", nil); w.Code != 400 {
			t.Fatal(w)
		}
	}
	hash := strings.Repeat("a", 40)
	if w := request(h, "PUT", "/v1/repos/demo/objects/"+hash, "write-secret", "blob", make([]byte, MaxObjectBytes+1)); w.Code != 413 {
		t.Fatal(w.Code)
	}
	if w := request(h, "GET", "/healthz", "", "", nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
}

func TestInterruptedUploadAndDuplicateAuthorization(t *testing.T) {
	h, s := testHandler(t)
	hash, _, _ := core.InspectObject("blob", []byte("partial"))
	route := "/v1/repos/demo/objects/" + hash
	r := httptest.NewRequest("PUT", route, &brokenReader{})
	r.Header.Set("Authorization", "Bearer write-secret")
	r.Header.Set("Content-Type", "application/octet-stream")
	r.Header.Set("X-SSC-Object-Type", "blob")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	if _, err := s.GetObject("demo", hash); !errors.Is(err, ErrNotFound) {
		t.Fatal("partial upload persisted")
	}
	if w := request(h, "PUT", route, "write-secret", "blob", []byte("partial")); w.Code != 204 {
		t.Fatal("retry failed", w.Code)
	}
	r = httptest.NewRequest("GET", route, nil)
	r.Header.Add("Authorization", "Bearer read-secret")
	r.Header.Add("Authorization", "Bearer write-secret")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
