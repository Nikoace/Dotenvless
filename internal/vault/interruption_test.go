package vault

import (
	"bytes"
	"dotenvless/internal/crypto"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type exitDuringProtect struct{ crypto.DPAPI }

func (exitDuringProtect) Protect([]byte, []byte) ([]byte, error) { os.Exit(29); return nil, nil }
func TestInterruptedMigrationReleasesLockAndPreservesOriginal(t *testing.T) {
	if os.Getenv("DVL_MIGRATION_WORKER") == "1" {
		_, _ = NewManager(os.Getenv("DVL_MIGRATION_FIXTURE"), exitDuringProtect{}).Migrate()
		os.Exit(90)
	}
	path := filepath.Join(t.TempDir(), "vault.dat")
	legacy := New(path, crypto.DPAPI{})
	must(t, legacy.Init("fixture"))
	must(t, legacy.Set("fixture", "TOKEN", []byte("GENERATED_MIGRATION")))
	original, e := os.ReadFile(path)
	must(t, e)
	self, e := os.Executable()
	must(t, e)
	cmd := exec.Command(self, "-test.run=^TestInterruptedMigrationReleasesLockAndPreservesOriginal$")
	cmd.Env = append(os.Environ(), "DVL_MIGRATION_WORKER=1", "DVL_MIGRATION_FIXTURE="+path)
	out, e := cmd.CombinedOutput()
	status, ok := e.(*exec.ExitError)
	if !ok || status.ExitCode() != 29 || len(out) != 0 {
		t.Fatal("migration worker did not interrupt silently")
	}
	after, e := os.ReadFile(path)
	must(t, e)
	backup, e := os.ReadFile(path + ".v1.bak")
	must(t, e)
	if !bytes.Equal(after, original) || !bytes.Equal(backup, original) {
		t.Fatal("interruption changed original ciphertext")
	}
	m := NewManager(path, crypto.DPAPI{})
	_, e = m.Migrate()
	must(t, e)
	state, e := m.Read()
	must(t, e)
	if state.Version != 2 {
		t.Fatal("resume did not commit migration")
	}
	env := state.Projects["fixture"].Environments["local"]
	value, e := m.Open(env.Bindings["TOKEN"].Cipher, "project", "fixture", env.ID, "TOKEN")
	must(t, e)
	if string(value) != "GENERATED_MIGRATION" {
		t.Fatal("resumed migration changed value")
	}
}
