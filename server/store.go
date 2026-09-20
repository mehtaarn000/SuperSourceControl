// Package server implements SSC's repository HTTP service.
package server

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"ssc/core"
	"strings"
	"sync"
)

const MaxObjectBytes = 8 << 20

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid request")
	ErrConflict = errors.New("branch tip conflict")
)

type repository struct {
	dir string
	mu  sync.Mutex
}
type Store struct{ repos map[string]*repository }
type Object struct {
	Type string
	Data []byte
}

func validRepoName(name string) bool {
	if len(name) == 0 || len(name) > 100 || name == "." || name == ".." {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// NewStore provisions only repositories explicitly listed by the administrator.
// One process owns a store; the serve command holds a process lock for its lifetime.
func NewStore(root string, names []string) (*Store, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("at least one repository is required")
	}
	s := &Store{repos: map[string]*repository{}}
	for _, name := range names {
		if !validRepoName(name) {
			return nil, fmt.Errorf("invalid repository name %q", name)
		}
		if _, ok := s.repos[name]; ok {
			return nil, fmt.Errorf("duplicate repository %q", name)
		}
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(dir, "objects"), 0700); err != nil {
			return nil, err
		}
		s.repos[name] = &repository{dir: dir}
	}
	return s, nil
}

func (s *Store) repo(name string) (*repository, error) {
	r, ok := s.repos[name]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *Store) PutObject(repo, hash, kind string, data []byte) error {
	r, err := s.repo(repo)
	if err != nil {
		return err
	}
	if !core.ValidObjectID(hash) || len(data) > MaxObjectBytes {
		return ErrInvalid
	}
	actual, _, err := core.InspectObject(kind, data)
	if err != nil || actual != hash {
		return fmt.Errorf("%w: object type, content, or hash mismatch", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	old, err := s.GetObject(repo, hash)
	if err == nil {
		if old.Type != kind || !bytes.Equal(old.Data, data) {
			return fmt.Errorf("%w: object is immutable", ErrConflict)
		}
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	var b bytes.Buffer
	z := zlib.NewWriter(&b)
	if _, err := z.Write(append([]byte(kind+"\n"), data...)); err != nil {
		return err
	}
	if err := z.Close(); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(r.dir, "objects", hash), b.Bytes())
}

func (s *Store) GetObject(repo, hash string) (Object, error) {
	r, err := s.repo(repo)
	if err != nil {
		return Object{}, err
	}
	if !core.ValidObjectID(hash) {
		return Object{}, ErrInvalid
	}
	f, err := os.Open(filepath.Join(r.dir, "objects", hash))
	if os.IsNotExist(err) {
		return Object{}, ErrNotFound
	}
	if err != nil {
		return Object{}, err
	}
	defer f.Close()
	z, err := zlib.NewReader(f)
	if err != nil {
		return Object{}, err
	}
	defer z.Close()
	data, err := ioutil.ReadAll(io.LimitReader(z, MaxObjectBytes+32))
	if err != nil {
		return Object{}, err
	}
	sep := bytes.IndexByte(data, '\n')
	if sep < 0 || sep > 10 || len(data)-sep-1 > MaxObjectBytes {
		return Object{}, fmt.Errorf("invalid stored object")
	}
	obj := Object{Type: string(data[:sep]), Data: data[sep+1:]}
	actual, _, err := core.InspectObject(obj.Type, obj.Data)
	if err != nil || actual != hash {
		return Object{}, fmt.Errorf("stored object failed verification")
	}
	return obj, nil
}

func writeAtomic(path string, data []byte) error {
	f, err := ioutil.TempFile(filepath.Dir(path), ".pending-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func validBranch(name string) bool {
	return len(name) <= 255 && core.ValidBranchName(name) && !strings.Contains(name, "\x00")
}
