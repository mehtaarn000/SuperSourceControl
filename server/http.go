package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
)

type Handler struct {
	store *Store
	auth  *Authorizer
	slots chan struct{}
}

func NewHandler(s *Store, c Config) (*Handler, error) {
	auth, err := NewAuthorizer(c)
	if err != nil {
		return nil, err
	}
	for _, name := range c.Repositories {
		if _, err := s.repo(name); err != nil {
			return nil, err
		}
	}
	return &Handler{store: s, auth: auth, slots: make(chan struct{}, 32)}, nil
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		fail(w, 404, "not found")
	case errors.Is(err, ErrInvalid):
		fail(w, 400, "invalid object or request")
	case errors.Is(err, ErrIncomplete):
		fail(w, 422, "upload the complete valid object graph before updating a branch")
	case errors.Is(err, ErrGraphLimit):
		fail(w, 422, "object graph exceeds server validation limits")
	case errors.Is(err, ErrConflict):
		fail(w, 409, "branch tip conflict")
	default:
		fail(w, 500, "repository operation failed")
	}
}
func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	fail(w, 405, "method not allowed")
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path == "/healthz" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, "GET")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
		return
	}
	// Reject ambiguous credentials; never accept tokens in query parameters.
	if len(r.Header.Values("Authorization")) != 1 {
		w.Header().Set("WWW-Authenticate", "Bearer")
		fail(w, 401, "authentication required")
		return
	}
	grants, ok := h.auth.authenticate(r.Header.Get("Authorization"))
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		fail(w, 401, "authentication required")
		return
	}
	if r.URL.Path != path.Clean(r.URL.Path) || strings.ContainsAny(r.URL.Path, "\\\x00") {
		fail(w, 400, "invalid path")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[0] != "v1" || parts[1] != "repos" || grants[parts[2]] == "" {
		fail(w, 404, "not found")
		return
	}
	repo := parts[2]
	if r.Method != http.MethodGet && r.Method != http.MethodHead && grants[repo] != "write" {
		fail(w, 403, "write access required")
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		fail(w, 503, "server busy; retry later")
		return
	}
	if parts[3] == "objects" && len(parts) == 5 {
		h.object(w, r, repo, parts[4])
		return
	}
	if parts[3] == "refs" {
		h.refs(w, r, repo, strings.Join(parts[4:], "/"))
		return
	}
	fail(w, 404, "not found")
}

func (h *Handler) object(w http.ResponseWriter, r *http.Request, repo, hash string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		obj, err := h.store.GetObject(repo, hash)
		if err != nil {
			storeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-SSC-Object-Type", obj.Type)
		w.Header().Set("Content-Length", strconv.Itoa(len(obj.Data)))
		w.WriteHeader(200)
		if r.Method == http.MethodGet {
			w.Write(obj.Data)
		}
	case http.MethodPut:
		media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "application/octet-stream" || r.Header.Get("Content-Encoding") != "" {
			fail(w, 415, "send an uncompressed application/octet-stream body")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxObjectBytes)
		defer r.Body.Close()
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			var large *http.MaxBytesError
			if errors.As(err, &large) {
				fail(w, 413, "object exceeds size limit")
			} else {
				fail(w, 400, "incomplete object body")
			}
			return
		}
		if err := h.store.PutObject(repo, hash, r.Header.Get("X-SSC-Object-Type"), data); err != nil {
			storeError(w, err)
			return
		}
		w.WriteHeader(204)
	default:
		methodNotAllowed(w, "GET, HEAD, PUT")
	}
}
