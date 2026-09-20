package remote

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"ssc/core"
	"strings"
)

type Repository struct{ Root string }

func Open(root string) (*Repository, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	r := &Repository{Root: abs}
	for _, path := range []string{".ssc", ".ssc/objects", ".ssc/branches", ".ssc/tmp"} {
		info, err := os.Lstat(filepath.Join(abs, path))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("run this command from an SSC repository root")
		}
	}
	return r, nil
}
func (r *Repository) metadata(parts ...string) string {
	return filepath.Join(append([]string{r.Root, ".ssc"}, parts...)...)
}
func (r *Repository) Branch() (string, error) {
	data, err := os.ReadFile(r.metadata("branch"))
	if err != nil {
		return "", err
	}
	name := string(data)
	if !core.ValidBranchName(name) {
		return "", fmt.Errorf("invalid active branch")
	}
	return name, nil
}
func (r *Repository) historyPath(branch string) (string, error) {
	if !core.ValidBranchName(branch) {
		return "", fmt.Errorf("invalid branch")
	}
	current := r.metadata("branches")
	for _, part := range strings.Split(branch, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("unsafe branch directory")
		}
	}
	target := filepath.Join(current, "commitlog")
	info, err := os.Lstat(target)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("unsafe branch log")
	}
	return target, nil
}
func (r *Repository) History(branch string) ([]string, error) {
	path, err := r.historyPath(branch)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	history := strings.Fields(string(data))
	for _, hash := range history {
		if !core.ValidObjectID(hash) {
			return nil, fmt.Errorf("invalid local history")
		}
	}
	return history, nil
}
func (r *Repository) readObject(hash, kind string) ([]byte, error) {
	if !core.ValidObjectID(hash) {
		return nil, fmt.Errorf("invalid object ID")
	}
	path := r.metadata("objects", hash)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("unsafe local object")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	z, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	data, err := io.ReadAll(io.LimitReader(z, MaxObjectBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxObjectBytes {
		return nil, fmt.Errorf("local object exceeds transfer limit")
	}
	actual, _, err := core.InspectObject(kind, data)
	if err != nil || actual != hash {
		return nil, fmt.Errorf("local object failed verification: %s", hash)
	}
	return data, nil
}
func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".sync-")
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
func (r *Repository) saveObject(hash, kind string, data []byte) error {
	actual, _, err := core.InspectObject(kind, data)
	if err != nil || actual != hash || len(data) > MaxObjectBytes {
		return fmt.Errorf("invalid imported object")
	}
	var b bytes.Buffer
	z := zlib.NewWriter(&b)
	if _, err := z.Write(data); err != nil {
		return err
	}
	if err := z.Close(); err != nil {
		return err
	}
	return atomicWrite(r.metadata("objects", hash), b.Bytes())
}
func (r *Repository) SetRemote(raw string) error {
	normalized, err := ValidateURL(raw)
	if err != nil {
		return err
	}
	unlock, err := core.LockRepository(r.Root)
	if err != nil {
		return err
	}
	defer unlock()
	data, _ := json.Marshal(map[string]string{"url": normalized})
	return atomicWrite(r.metadata("remote.json"), data)
}
func (r *Repository) Remote() (string, error) {
	data, err := os.ReadFile(r.metadata("remote.json"))
	if err != nil {
		return "", fmt.Errorf("configure a remote with ssc remote <repository-url>")
	}
	var config struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}
	return ValidateURL(config.URL)
}
func tip(history []string) string {
	if len(history) == 0 {
		return ""
	}
	return history[len(history)-1]
}
