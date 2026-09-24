package vault

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type fixtureRecord struct{ value, context []byte }
type fixtureProtector struct {
	mu                    sync.Mutex
	records               map[string]fixtureRecord
	failProtect, failOpen bool
	failAt                int
}

func (f *fixtureProtector) Protect(value, context []byte) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failProtect || (f.failAt > 0 && len(f.records)+1 == f.failAt) {
		return nil, errors.New("FAKE-SENSITIVE-ERROR")
	}
	token := fmt.Sprintf("fixture-token-%d", len(f.records)+1)
	f.records[token] = fixtureRecord{bytes.Clone(value), bytes.Clone(context)}
	return []byte(token), nil
}
func (f *fixtureProtector) Unprotect(value, context []byte) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOpen {
		return nil, errors.New("FAKE-SENSITIVE-ERROR")
	}
	r, ok := f.records[string(value)]
	if !ok || !bytes.Equal(r.context, context) {
		return nil, errors.New("invalid fixture")
	}
	return bytes.Clone(r.value), nil
}
func fixture(t *testing.T) (*Store, *fixtureProtector) {
	t.Helper()
	p := &fixtureProtector{records: map[string]fixtureRecord{}}
	return New(filepath.Join(t.TempDir(), "vault.dat"), p), p
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func TestCRUDAndProjectIsolation(t *testing.T) {
	s, p := fixture(t)
	must(t, s.Init("project-one"))
	must(t, s.Init("project-two"))
	must(t, s.Set("project-one", "token", []byte("FAKE-SECRET-ONE")))
	must(t, s.Set("project-two", "TOKEN", []byte("FAKE-SECRET-TWO")))
	must(t, s.Init("project-one"))
	reopened := New(s.path, p)
	values, err := reopened.Values("project-one")
	must(t, err)
	if values["TOKEN"] != "FAKE-SECRET-ONE" || len(values) != 1 {
		t.Fatal("wrong project values")
	}
	must(t, s.Set("project-one", "TOKEN", []byte("FAKE-OVERWRITE")))
	must(t, s.Set("project-one", "EMPTY", nil))
	keys, err := s.Keys("project-one")
	must(t, err)
	if !reflect.DeepEqual(keys, []string{"EMPTY", "TOKEN"}) {
		t.Fatalf("keys: %v", keys)
	}
	data, err := os.ReadFile(s.path)
	must(t, err)
	for _, secret := range []string{"FAKE-SECRET-ONE", "FAKE-SECRET-TWO", "FAKE-OVERWRITE"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatal("plaintext persisted")
		}
	}
	must(t, s.Unset("project-one", "ToKeN"))
	if err := s.Unset("project-one", "TOKEN"); err == nil {
		t.Fatal("missing key deletion accepted")
	}
	values, err = s.Values("project-two")
	must(t, err)
	if values["TOKEN"] != "FAKE-SECRET-TWO" {
		t.Fatal("other project changed")
	}
}
func TestUninitializedAndInvalidInput(t *testing.T) {
	s, _ := fixture(t)
	if err := s.Set("missing", "TOKEN", []byte("x")); err == nil {
		t.Fatal("set without init")
	}
	if _, err := s.Values("missing"); err == nil {
		t.Fatal("values without init")
	}
	must(t, s.Init("p"))
	for _, key := range []string{"", "A=B", "A B", "1TOKEN", "UNICODE_日本"} {
		if err := s.Set("p", key, []byte("x")); err == nil {
			t.Fatal("invalid key accepted")
		}
	}
	for _, value := range [][]byte{[]byte("a\x00b"), bytes.Repeat([]byte{'x'}, 32769)} {
		if err := s.Set("p", "TOKEN", value); err == nil {
			t.Fatal("invalid value accepted")
		}
	}
}
func TestCorruptVaultNeverOverwritten(t *testing.T) {
	for _, data := range []string{
		"{", "null", `{"version":2,"projects":{}}`, `{"version":1,"projects":null}`,
		`{"version":1,"projects":{"p":null}}`, `{"version":1,"projects":{"p":{"TOKEN":""}}}`,
		`{"version":1,"projects":{},"unexpected":"FAKE-SECRET"}`,
		`{"version":1,"version":1,"projects":{}}`,
		`{"version":1,"projects":{"p":{"TOKEN":"YQ==","TOKEN":"Yg=="}}}`,
	} {
		t.Run(fmt.Sprintf("case-%d", len(data)), func(t *testing.T) {
			s, _ := fixture(t)
			must(t, os.WriteFile(s.path, []byte(data), 0600))
			if err := s.Init("p"); err == nil {
				t.Fatal("corrupt vault accepted")
			} else if strings.Contains(err.Error(), "FAKE-SECRET") {
				t.Fatal("data leaked")
			}
			after, err := os.ReadFile(s.path)
			must(t, err)
			if string(after) != data {
				t.Fatal("corrupt vault overwritten")
			}
		})
	}
}
func TestCryptoAndCommitFailuresPreserveVault(t *testing.T) {
	s, p := fixture(t)
	must(t, s.Init("p"))
	must(t, s.Set("p", "TOKEN", []byte("FAKE-ORIGINAL")))
	before, err := os.ReadFile(s.path)
	must(t, err)
	p.failProtect = true
	if err := s.Set("p", "TOKEN", []byte("x")); err == nil || strings.Contains(err.Error(), "FAKE-SENSITIVE") {
		t.Fatal("unsafe encryption failure")
	}
	p.failProtect = false
	p.failOpen = true
	if _, err := s.Values("p"); err == nil || strings.Contains(err.Error(), "FAKE-SENSITIVE") {
		t.Fatal("unsafe decryption failure")
	}
	p.failOpen = false
	s.replace = func(string, string) error { return errors.New("injected replacement failure") }
	if err := s.Set("p", "TOKEN", []byte("FAKE-NEW")); err == nil {
		t.Fatal("commit failure ignored")
	}
	after, err := os.ReadFile(s.path)
	must(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("failed operation changed vault")
	}
	files, err := os.ReadDir(filepath.Dir(s.path))
	must(t, err)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".vault-") {
			t.Fatal("temporary file remained")
		}
	}
}
func TestConcurrentUpdates(t *testing.T) {
	s, _ := fixture(t)
	must(t, s.Init("p"))
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); errs <- s.Set("p", fmt.Sprintf("KEY_%d", i), []byte("FAKE")) }(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		must(t, err)
	}
	keys, err := s.Keys("p")
	must(t, err)
	if len(keys) != 12 {
		t.Fatalf("lost writes: %d", len(keys))
	}
}

func TestImportIsAtomicAndRequiresExplicitOverwrite(t *testing.T) {
	s, p := fixture(t)
	must(t, s.Init("p"))
	must(t, s.Set("p", "TOKEN", []byte("FAKE-OLD")))
	before, err := os.ReadFile(s.path)
	must(t, err)
	values := map[string][]byte{"token": []byte("FAKE-NEW"), "ADDED": []byte("FAKE-ADDED")}
	if _, err := s.Import("p", values, false); err == nil {
		t.Fatal("conflict accepted")
	}
	after, err := os.ReadFile(s.path)
	must(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("conflicting import changed vault")
	}
	p.failProtect = true
	if _, err := s.Import("p", values, true); err == nil {
		t.Fatal("failed encryption accepted")
	}
	after, err = os.ReadFile(s.path)
	must(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("failed import changed vault")
	}
	p.failProtect = false
	keys, err := s.Import("p", values, true)
	must(t, err)
	if !reflect.DeepEqual(keys, []string{"ADDED", "TOKEN"}) {
		t.Fatal("imported keys mismatch")
	}
	actual, err := s.Values("p")
	must(t, err)
	if actual["TOKEN"] != "FAKE-NEW" || actual["ADDED"] != "FAKE-ADDED" {
		t.Fatal("import values mismatch")
	}
}

func TestImportRollsBackWhenLaterEncryptionFails(t *testing.T) {
	s, p := fixture(t)
	must(t, s.Init("p"))
	must(t, s.Set("p", "OLD", []byte("FAKE-OLD")))
	before, err := os.ReadFile(s.path)
	must(t, err)
	p.failAt = 3
	if _, err := s.Import("p", map[string][]byte{"A": []byte("FAKE-A"), "B": []byte("FAKE-B")}, false); err == nil {
		t.Fatal("later encryption failure ignored")
	}
	after, err := os.ReadFile(s.path)
	must(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("partial import persisted")
	}
	keys, err := s.Keys("p")
	must(t, err)
	if !reflect.DeepEqual(keys, []string{"OLD"}) {
		t.Fatal("partial keys persisted")
	}
}
