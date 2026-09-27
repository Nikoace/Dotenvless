package application

import (
	"dotenvless/internal/vault"
	"errors"
	"strings"
	"testing"
)

func TestPreviewImportAtomicConflictCancelAndStale(t *testing.T) {
	s := serviceFixture(t)
	p, e := s.Register(t.TempDir(), 0)
	check(t, e)
	check(t, s.Set(p.ID, "base", "KEY", []byte("GENERATED_OLD"), vault.AnyRevision))
	preview, e := s.PreviewImport(p.ID, "local", strings.NewReader("KEY=GENERATED_NEW\nNEW=GENERATED_NEWER"))
	check(t, e)
	if len(preview.Keys) != 2 || len(preview.Conflicts) != 1 {
		t.Fatal("preview lacks conflicts")
	}
	if s.CommitImport(preview, false) == nil {
		t.Fatal("implicit overwrite allowed")
	}
	got, e := s.Resolve(p.ID, "local")
	check(t, e)
	if len(got.Values) != 1 {
		t.Fatal("partial batch committed")
	}
	check(t, s.CommitImport(preview, true))
	if s.CommitImport(preview, true) == nil {
		t.Fatal("used batch replayed")
	}
	preview, e = s.PreviewImport(p.ID, "local", strings.NewReader("CANCEL=GENERATED_CANCEL"))
	check(t, e)
	preview.Close()
	if s.CommitImport(preview, false) == nil {
		t.Fatal("cancelled batch committed")
	}
	preview, e = s.PreviewImport(p.ID, "local", strings.NewReader("STALE=GENERATED_STALE"))
	check(t, e)
	check(t, s.Set(p.ID, "local", "OTHER", []byte("new"), vault.AnyRevision))
	if e = s.CommitImport(preview, false); !errors.Is(e, vault.ErrConflict) {
		t.Fatal("stale batch accepted")
	}
	preview.Close()
}
