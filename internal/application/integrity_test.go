package application

import (
	"bytes"
	"dotenvless/internal/crypto"
	"dotenvless/internal/vault"
	"errors"
	"os"
	"testing"
)

func TestCorruptRegistryCannotBeModified(t *testing.T) {
	s := serviceFixture(t)
	p, e := s.Register(t.TempDir(), 0)
	check(t, e)
	other := t.TempDir()
	check(t, s.Store.Update(vault.AnyRevision, func(state *vault.Snapshot) error {
		v := state.Projects[p.ID]
		v.Root = other
		state.Projects[p.ID] = v
		return nil
	}))
	before, e := os.ReadFile(s.Path)
	check(t, e)
	if e = s.Set(p.ID, "local", "TOKEN", []byte("GENERATED"), vault.AnyRevision); !errors.Is(e, vault.ErrCorrupt) {
		t.Fatal("mutation accepted mismatched root identity")
	}
	after, e := os.ReadFile(s.Path)
	check(t, e)
	if !bytes.Equal(before, after) {
		t.Fatal("corrupt registry changed")
	}
}

type observeProtector struct {
	hook  func()
	calls int
}

func (p *observeProtector) Protect(v, c []byte) ([]byte, error) {
	return (crypto.DPAPI{}).Protect(v, c)
}
func (p *observeProtector) Unprotect(v, c []byte) ([]byte, error) {
	p.calls++
	if p.hook != nil {
		f := p.hook
		p.hook = nil
		f()
	}
	return (crypto.DPAPI{}).Unprotect(v, c)
}
func TestViewNeverDecryptsAndRunSnapshotSurvivesSharedUpdate(t *testing.T) {
	s := serviceFixture(t)
	p, e := s.Register(t.TempDir(), 0)
	check(t, e)
	id, e := s.SaveShared("", "fixture", []byte("GENERATED_OLD"), vault.AnyRevision)
	check(t, e)
	for _, key := range []string{"A", "B"} {
		check(t, s.BindShared(p.ID, "local", key, id, vault.AnyRevision))
	}
	protector := &observeProtector{}
	reading := New(s.Path, protector)
	v, e := reading.View(p.ID, "local")
	check(t, e)
	if protector.calls != 0 || len(v.Entries) != 2 {
		t.Fatal("metadata listing decrypted values")
	}
	protector.hook = func() { _, e := s.SaveShared(id, "fixture", []byte("GENERATED_NEW"), vault.AnyRevision); check(t, e) }
	got, e := reading.Resolve(p.ID, "local")
	check(t, e)
	if got.Values["A"] != "GENERATED_OLD" || got.Values["B"] != "GENERATED_OLD" {
		t.Fatal("resolution mixed revisions")
	}
	got, e = reading.Resolve(p.ID, "local")
	check(t, e)
	if got.Values["A"] != "GENERATED_NEW" || got.Values["B"] != "GENERATED_NEW" {
		t.Fatal("next resolution missed shared update")
	}
}
