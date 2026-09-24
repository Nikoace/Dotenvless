package vault

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLockReleasedAfterProcessExit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.dat")
	binary, err := os.Executable()
	must(t, err)
	cmd := exec.Command(binary, "-test.run=^TestAbandonedLockHelper$")
	cmd.Env = append(os.Environ(), "DVL_TEST_LOCK="+path+".lock")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("lock helper: %v: %s", err, output)
	}
	s := New(path, nil)
	must(t, s.Init("p"))
}
func TestAbandonedLockHelper(t *testing.T) {
	path := os.Getenv("DVL_TEST_LOCK")
	if path == "" {
		return
	}
	unlock, err := lockFile(path)
	if err != nil {
		os.Exit(15)
	}
	runtime.KeepAlive(unlock)
	// Intentionally skip unlock to model an interrupted writer.
	os.Exit(0)
}
