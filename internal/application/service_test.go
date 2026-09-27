package application

import (
	"context"
	"dotenvless/internal/crypto"
	"dotenvless/internal/project"
	"dotenvless/internal/vault"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func serviceFixture(t *testing.T) *Service {
	t.Helper()
	return New(filepath.Join(t.TempDir(), "vault.dat"), crypto.DPAPI{})
}
func check(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func mkdir(t *testing.T, path string) { t.Helper(); check(t, os.MkdirAll(path, 0700)) }
func TestRegisterPlainGitAndNearestRoots(t *testing.T) {
	s := serviceFixture(t)
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	nested := filepath.Join(sub, "repo")
	mkdir(t, filepath.Join(nested, ".git"))
	p, e := s.Register(root, 0)
	check(t, e)
	q, e := s.FindProject(sub)
	check(t, e)
	if q.ID != p.ID {
		t.Fatal("ordinary descendant not associated")
	}
	state, e := s.State()
	check(t, e)
	if state.Projects[p.ID].Kind != "directory" {
		t.Fatal("wrong kind")
	}
	child, e := s.Register(sub, state.Revision)
	check(t, e)
	q, e = s.FindProject(sub)
	check(t, e)
	if q.ID != child.ID {
		t.Fatal("nearest ordinary root lost")
	}
	q, e = s.FindProject(nested)
	check(t, e)
	gitID, e := project.Discover(nested)
	check(t, e)
	if q.ID != gitID.ID || q.ID == child.ID {
		t.Fatal("nested git isolation lost")
	}
	_, e = s.Register(nested, vault.AnyRevision)
	check(t, e)
	state, e = s.State()
	check(t, e)
	if len(state.Projects) != 3 || len(state.Projects[p.ID].Environments) != 1 {
		t.Fatal("wrong project registry")
	}
	if _, e = s.Register(root, 0); !errors.Is(e, vault.ErrConflict) {
		t.Fatal("stale registration accepted")
	}
	check(t, s.RemoveProject(p.ID, state.Revision))
	if _, e = os.Stat(root); e != nil {
		t.Fatal("project folder deleted")
	}
}
func TestRegistrationAssociatesMigratedHashAndRejectsInProjectVault(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, ".git"))
	id, e := project.Discover(root)
	check(t, e)
	s := serviceFixture(t)
	legacy := vault.New(s.Path, crypto.DPAPI{})
	check(t, legacy.Init(id.ID))
	check(t, legacy.Set(id.ID, "KEY", []byte("GENERATED_TEST_"+filepath.Base(root))))
	_, e = s.Store.Migrate()
	check(t, e)
	state, e := s.State()
	check(t, e)
	if state.Projects[id.ID].Root != "" {
		t.Fatal("migration guessed root")
	}
	got, e := s.Register(root, state.Revision)
	check(t, e)
	if got.ID != id.ID {
		t.Fatal("hash changed")
	}
	state, e = s.State()
	check(t, e)
	if len(state.Projects) != 1 || len(state.Projects[id.ID].Environments["local"].Bindings) != 1 {
		t.Fatal("association lost values")
	}
	bad := New(filepath.Join(root, "secrets", "vault.dat"), crypto.DPAPI{})
	if _, e = bad.Register(root, 0); e == nil {
		t.Fatal("vault inside project accepted")
	}
}
func TestDiscoveryIsExplicitCancellableAndSkipsDependencies(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "one", ".git"))
	mkdir(t, filepath.Join(root, "two", ".git"))
	mkdir(t, filepath.Join(root, "node_modules", "hidden", ".git"))
	check(t, os.WriteFile(filepath.Join(root, "one", ".env"), []byte("INVALID_FAKE_DOTENV_NOT_TO_BE_PARSED"), 0600))
	candidates, e := Discover(context.Background(), root)
	check(t, e)
	if len(candidates) != 2 {
		t.Fatal("candidate scan missing roots or entered dependencies")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = Discover(ctx, root); !errors.Is(e, context.Canceled) {
		t.Fatal("scan ignored cancellation")
	}
}
