package server

import (
	"errors"
	"ssc/core"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir(), []string{"demo", "private"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func putTestObject(t *testing.T, s *Store, kind string, data []byte) string {
	t.Helper()
	hash, _, err := core.InspectObject(kind, data)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PutObject("demo", hash, kind, data); err != nil {
		t.Fatal(err)
	}
	return hash
}
func TestObjectStorageVerificationAndIsolation(t *testing.T) {
	s := testStore(t)
	hash := putTestObject(t, s, "blob", []byte("hello"))
	obj, err := s.GetObject("demo", hash)
	if err != nil || string(obj.Data) != "hello" || obj.Type != "blob" {
		t.Fatalf("%+v %v", obj, err)
	}
	if err := s.PutObject("demo", hash, "blob", []byte("hello")); err != nil {
		t.Fatal("retry failed:", err)
	}
	if err := s.PutObject("demo", hash, "blob", []byte("changed")); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.GetObject("private", hash); !errors.Is(err, ErrNotFound) {
		t.Fatal("object leaked across repositories")
	}
	if _, err := s.GetObject("demo", "../outside"); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := NewStore(t.TempDir(), []string{"../escape"}); err == nil {
		t.Fatal("invalid repo accepted")
	}
}
