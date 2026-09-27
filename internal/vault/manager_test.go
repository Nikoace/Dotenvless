package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func emptyRecord() ProjectRecord {
	return ProjectRecord{Name: "test", Kind: "directory", Base: map[string]Binding{}, Environments: map[string]Environment{"local": {ID: "scope-local", Bindings: map[string]Binding{}}}}
}
func TestManagerEmptyRevisionAndConcurrentWrites(t *testing.T) {
	legacy, p := fixture(t)
	m := NewManager(legacy.path, p)
	s, e := m.Read()
	must(t, e)
	if s.Version != 2 || s.Revision != 0 || len(s.Projects) != 0 {
		t.Fatal("new snapshot is not empty v2")
	}
	if _, e = os.Stat(legacy.path); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("read created vault")
	}
	must(t, m.Update(0, func(s *Snapshot) error { s.Projects["one"] = emptyRecord(); return nil }))
	if e = m.Update(0, func(s *Snapshot) error { delete(s.Projects, "one"); return nil }); !errors.Is(e, ErrConflict) {
		t.Fatal("stale revision accepted")
	}
	var wg sync.WaitGroup
	for _, id := range []string{"two", "three", "four"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if e := m.Update(AnyRevision, func(s *Snapshot) error { s.Projects[id] = emptyRecord(); return nil }); e != nil {
				t.Error(e)
			}
		}(id)
	}
	wg.Wait()
	s, e = m.Read()
	must(t, e)
	if len(s.Projects) != 4 || s.Revision != 4 {
		t.Fatal("concurrent update lost")
	}
	if _, e = legacy.Keys("one"); e == nil {
		t.Fatal("old CLI accepted v2")
	}
}
func TestManagerMigrationAndContext(t *testing.T) {
	legacy, p := fixture(t)
	must(t, legacy.Init("project"))
	value := []byte("GENERATED_FAKE_" + filepath.Base(t.TempDir()))
	must(t, legacy.Set("project", "TOKEN", value))
	original, e := os.ReadFile(legacy.path)
	must(t, e)
	m := NewManager(legacy.path, p)
	s, e := m.Read()
	must(t, e)
	if s.Version != 1 || len(s.Projects["project"].Environments["local"].Bindings) != 1 {
		t.Fatal("legacy view missing")
	}
	if e = m.Update(AnyRevision, func(*Snapshot) error { return nil }); !errors.Is(e, ErrMigration) {
		t.Fatal("legacy write allowed")
	}
	backup, e := m.Migrate()
	must(t, e)
	b, e := os.ReadFile(backup)
	must(t, e)
	if !bytes.Equal(b, original) {
		t.Fatal("backup is not original")
	}
	s, e = m.Read()
	must(t, e)
	env := s.Projects["project"].Environments["local"]
	got, e := m.Open(env.Bindings["TOKEN"].Cipher, "project", "project", env.ID, "TOKEN")
	must(t, e)
	if !bytes.Equal(got, value) {
		t.Fatal("migration value changed")
	}
	if _, e = m.Open(env.Bindings["TOKEN"].Cipher, "project", "other", env.ID, "TOKEN"); e == nil {
		t.Fatal("cross-project swap accepted")
	}
	if _, e = m.Open(env.Bindings["TOKEN"].Cipher, "project", "project", "other", "TOKEN"); e == nil {
		t.Fatal("cross-scope swap accepted")
	}
	after, e := os.ReadFile(legacy.path)
	must(t, e)
	if bytes.Contains(after, value) {
		t.Fatal("plaintext persisted")
	}
	if _, e = m.Migrate(); e != nil {
		t.Fatal("repeat migration should be no-op")
	}
}
func TestManagerFailedMigrationPreservesBytes(t *testing.T) {
	for _, failure := range []string{"decrypt", "protect", "replace", "backup"} {
		t.Run(failure, func(t *testing.T) {
			legacy, p := fixture(t)
			must(t, legacy.Init("p"))
			must(t, legacy.Set("p", "KEY", []byte("GENERATED_FAKE")))
			original, e := os.ReadFile(legacy.path)
			must(t, e)
			m := NewManager(legacy.path, p)
			switch failure {
			case "decrypt":
				p.failOpen = true
			case "protect":
				p.failProtect = true
			case "replace":
				m.store.replace = func(string, string) error { return errors.New("fake") }
			case "backup":
				must(t, os.WriteFile(legacy.path+".v1.bak", []byte("different"), 0600))
			}
			if _, e = m.Migrate(); e == nil {
				t.Fatal("failed migration succeeded")
			}
			after, e := os.ReadFile(legacy.path)
			must(t, e)
			if !bytes.Equal(original, after) {
				t.Fatal("failed migration changed original")
			}
		})
	}
}
func TestManagerStrictFormatAndFailedCommit(t *testing.T) {
	legacy, p := fixture(t)
	m := NewManager(legacy.path, p)
	for _, raw := range []string{`{"version":3,"projects":{}}`, `{"version":2,"revision":1,"projects":{},"shared":{},"unexpected":true}`, `{"version":2,"version":2,"revision":1,"projects":{},"shared":{}}`, `{"version":2,"revision":0,"projects":{},"shared":{}}`} {
		must(t, os.WriteFile(legacy.path, []byte(raw), 0600))
		if _, e := m.Read(); e == nil {
			t.Fatal("corrupt document accepted")
		}
		if e := m.Update(AnyRevision, func(*Snapshot) error { return nil }); e == nil {
			t.Fatal("corrupt document overwritten")
		}
		got, e := os.ReadFile(legacy.path)
		must(t, e)
		if string(got) != raw {
			t.Fatal("corrupt bytes changed")
		}
	}
}
