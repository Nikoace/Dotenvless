package application

import (
	"bytes"
	"dotenvless/internal/vault"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLayeredResolutionEmptyDisabledAndEnvironmentLifecycle(t *testing.T) {
	s := serviceFixture(t)
	p, e := s.Register(t.TempDir(), 0)
	check(t, e)
	rev := vault.AnyRevision
	check(t, s.Set(p.ID, "base", "TOKEN", []byte("GENERATED_BASE"), rev))
	check(t, s.Set(p.ID, "base", "BASE_ONLY", []byte("GENERATED_BASE_ONLY"), rev))
	check(t, s.Set(p.ID, "local", "TOKEN", []byte("GENERATED_OVERRIDE"), rev))
	check(t, s.Set(p.ID, "local", "EMPTY", nil, rev))
	check(t, s.Disable(p.ID, "local", "BASE_ONLY", rev))
	got, e := s.Resolve(p.ID, "local")
	check(t, e)
	if got.Values["TOKEN"] != "GENERATED_OVERRIDE" || len(got.Values) != 2 || len(got.Removed) != 1 || got.Removed[0] != "BASE_ONLY" {
		t.Fatal("wrong layered result")
	}
	if v, ok := got.Values["EMPTY"]; !ok || v != "" {
		t.Fatal("empty value was removed")
	}
	check(t, s.Unset(p.ID, "local", "TOKEN", rev))
	got, e = s.Resolve(p.ID, "local")
	check(t, e)
	if got.Values["TOKEN"] != "GENERATED_BASE" {
		t.Fatal("unset did not restore base")
	}
	check(t, s.CreateEnvironment(p.ID, "test", "local", rev))
	check(t, s.RenameEnvironment(p.ID, "test", "qa", rev))
	got, e = s.Resolve(p.ID, "qa")
	check(t, e)
	if len(got.Removed) != 1 {
		t.Fatal("copy lost mask")
	}
	for _, name := range []string{"local", "base"} {
		if s.DeleteEnvironment(p.ID, name, rev) == nil {
			t.Fatal("protected layer deleted")
		}
	}
	check(t, s.DeleteEnvironment(p.ID, "qa", rev))
	if _, e = s.Resolve(p.ID, "qa"); e == nil {
		t.Fatal("missing environment accepted")
	}
	if s.Disable(p.ID, "base", "TOKEN", rev) == nil {
		t.Fatal("base mask accepted")
	}
	data, e := os.ReadFile(s.Path)
	check(t, e)
	if bytes.Contains(data, []byte("GENERATED_")) {
		t.Fatal("plaintext persisted")
	}
}
func TestSharedExplicitAliasingImpactAndSingleSnapshot(t *testing.T) {
	s := serviceFixture(t)
	p, e := s.Register(t.TempDir(), 0)
	check(t, e)
	q, e := s.Register(t.TempDir(), vault.AnyRevision)
	check(t, e)
	shared, e := s.SaveShared("", "provider/key", []byte("GENERATED_SHARED"), vault.AnyRevision)
	check(t, e)
	check(t, s.BindShared(p.ID, "base", "API_KEY", shared, vault.AnyRevision))
	check(t, s.BindShared(q.ID, "local", "OTHER_NAME", shared, vault.AnyRevision))
	uses, e := s.SharedUsage(shared)
	check(t, e)
	if len(uses) != 2 {
		t.Fatal("missing impact metadata")
	}
	if s.DeleteShared(shared, vault.AnyRevision) == nil {
		t.Fatal("referenced shared deleted")
	}
	snapshot, e := s.State()
	check(t, e)
	_, e = s.SaveShared(shared, "provider/key", []byte("GENERATED_ROTATED"), snapshot.Revision)
	check(t, e)
	if e = s.Set(p.ID, "local", "OTHER", []byte("stale"), snapshot.Revision); !errors.Is(e, vault.ErrConflict) {
		t.Fatal("stale editor overwrote vault")
	}
	for _, id := range []string{p.ID, q.ID} {
		got, e := s.Resolve(id, "local")
		check(t, e)
		if len(got.Values) != 1 {
			t.Fatal("global library implicitly injected")
		}
		for _, v := range got.Values {
			if v != "GENERATED_ROTATED" {
				t.Fatal("rotation missing")
			}
		}
	}
	check(t, s.Set(p.ID, "local", "API_KEY", []byte("GENERATED_PRIVATE"), vault.AnyRevision))
	got, e := s.Resolve(q.ID, "local")
	check(t, e)
	if got.Values["OTHER_NAME"] != "GENERATED_ROTATED" {
		t.Fatal("project edit changed shared source")
	}
	entries, _, e := s.Entries(p.ID, "local")
	check(t, e)
	if len(entries) != 1 || entries[0].Source != "local" {
		t.Fatal("wrong entry provenance")
	}
	check(t, s.Unset(p.ID, "base", "API_KEY", vault.AnyRevision))
	check(t, s.Unset(q.ID, "local", "OTHER_NAME", vault.AnyRevision))
	check(t, s.DeleteShared(shared, vault.AnyRevision))
}
func TestCopyReprotectsValuesAndRevealOnlyOne(t *testing.T) {
	s := serviceFixture(t)
	p, e := s.Register(t.TempDir(), 0)
	check(t, e)
	check(t, s.Set(p.ID, "local", "VALUE", []byte("GENERATED_"+filepath.Base(p.Root)), vault.AnyRevision))
	check(t, s.CreateEnvironment(p.ID, "dev", "local", vault.AnyRevision))
	check(t, s.Store.Update(vault.AnyRevision, func(state *vault.Snapshot) error {
		p := state.Projects[p.ID]
		env := p.Environments["local"]
		env.Bindings["BROKEN"] = vault.Binding{Kind: "shared", SharedID: "missing"}
		return nil
	}))
	one, e := s.Reveal(p.ID, "local", "VALUE")
	check(t, e)
	if len(one) == 0 {
		t.Fatal("single value missing")
	}
	clear(one)
	if _, e = s.Resolve(p.ID, "local"); e == nil {
		t.Fatal("broken reference did not stop resolution")
	}
	if _, e = s.Resolve(p.ID, "dev"); e != nil {
		t.Fatal("copy used old encryption context")
	}
}
