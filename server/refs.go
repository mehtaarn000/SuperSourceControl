package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"ssc/core"
)

const MaxGraphObjects = 10000
const MaxGraphBytes = 256 << 20

var ErrIncomplete = errors.New("incomplete or invalid object graph")
var ErrGraphLimit = errors.New("object graph exceeds validation limits")

func readRefs(r *repository) (map[string]string, error) {
	data, err := ioutil.ReadFile(filepath.Join(r.dir, "refs.json"))
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var refs map[string]string
	if err := json.Unmarshal(data, &refs); err != nil {
		return nil, err
	}
	if refs == nil {
		return nil, fmt.Errorf("invalid stored refs")
	}
	for name, hash := range refs {
		if !validBranch(name) || !core.ValidObjectID(hash) {
			return nil, fmt.Errorf("invalid stored ref")
		}
	}
	return refs, nil
}

func (s *Store) ListRefs(repo string) (map[string]string, error) {
	r, err := s.repo(repo)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return readRefs(r)
}

// UpdateRef is a compare-and-swap. Empty old creates a branch; no force or delete.
func (s *Store) UpdateRef(ctx context.Context, repo, branch, old, next string) error {
	if !validBranch(branch) || !core.ValidObjectID(next) || (old != "" && !core.ValidObjectID(old)) {
		return ErrInvalid
	}
	r, err := s.repo(repo)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	refs, err := readRefs(r)
	if err != nil {
		return err
	}
	if refs[branch] != old {
		return ErrConflict
	}
	commits, err := s.validateGraph(ctx, repo, next)
	if err != nil {
		return err
	}
	if old != "" && !commits[old] {
		return fmt.Errorf("%w: update is not a fast-forward", ErrConflict)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	refs[branch] = next
	data, err := json.Marshal(refs)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(r.dir, "refs.json"), data)
}

func (s *Store) validateGraph(ctx context.Context, repo, tip string) (map[string]bool, error) {
	pending := []core.ObjectReference{{Hash: tip, Type: "commit"}}
	seen := map[string]string{}
	commits := map[string]bool{}
	total := 0
	for len(pending) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ref := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if kind, ok := seen[ref.Hash]; ok {
			if kind != ref.Type {
				return nil, ErrIncomplete
			}
			continue
		}
		if len(seen) >= MaxGraphObjects {
			return nil, ErrGraphLimit
		}
		obj, err := s.GetObject(repo, ref.Hash)
		if err != nil || obj.Type != ref.Type {
			return nil, ErrIncomplete
		}
		total += len(obj.Data)
		if total > MaxGraphBytes {
			return nil, ErrGraphLimit
		}
		_, children, err := core.InspectObject(obj.Type, obj.Data)
		if err != nil {
			return nil, ErrIncomplete
		}
		seen[ref.Hash] = obj.Type
		if obj.Type == "commit" {
			commits[ref.Hash] = true
		}
		pending = append(pending, children...)
	}
	return commits, nil
}
