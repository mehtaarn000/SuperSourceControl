package server

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

func (h *Handler) refs(w http.ResponseWriter, r *http.Request, repo, branch string) {
	switch r.Method {
	case http.MethodGet:
		refs, err := h.store.ListRefs(repo)
		if err != nil {
			storeError(w, err)
			return
		}
		if branch == "" {
			writeJSON(w, 200, map[string]interface{}{"refs": refs})
			return
		}
		if !validBranch(branch) {
			fail(w, 400, "invalid branch name")
			return
		}
		hash, ok := refs[branch]
		if !ok {
			fail(w, 404, "not found")
			return
		}
		writeJSON(w, 200, map[string]string{"name": branch, "hash": hash})
	case http.MethodPut:
		if branch == "" {
			methodNotAllowed(w, "GET")
			return
		}
		media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "application/json" || r.Header.Get("Content-Encoding") != "" {
			fail(w, 415, "send application/json")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		defer r.Body.Close()
		// Pointer fields require both old and new explicitly, including old:"".
		var update struct {
			Old *string `json:"old"`
			New *string `json:"new"`
		}
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		err = d.Decode(&update)
		if err == nil {
			if trailing := d.Decode(new(interface{})); trailing != io.EOF {
				err = ErrInvalid
			}
		}
		if err != nil || update.Old == nil || update.New == nil {
			var large *http.MaxBytesError
			if errors.As(err, &large) {
				fail(w, 413, "request exceeds size limit")
			} else {
				fail(w, 400, "expected explicit old and new hashes")
			}
			return
		}
		if err := h.store.UpdateRef(r.Context(), repo, branch, *update.Old, *update.New); err != nil {
			storeError(w, err)
			return
		}
		writeJSON(w, 200, map[string]string{"name": branch, "hash": *update.New})
	default:
		methodNotAllowed(w, "GET, PUT")
	}
}
