package project

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
}
func write(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}
func discover(t *testing.T, path string) Identity {
	t.Helper()
	p, err := Discover(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRootAndSubdirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ProjectOne")
	mkdir(t, filepath.Join(root, ".git"))
	sub := filepath.Join(root, "src", "app")
	mkdir(t, sub)
	p := discover(t, root)
	q := discover(t, sub)
	if p != q {
		t.Fatalf("different identities: %+v / %+v", p, q)
	}
	normalized, _ := filepath.EvalSymlinks(root)
	digest := sha256.Sum256([]byte(normalized))
	if p.Root != normalized || p.Name != "ProjectOne" || p.ID != hex.EncodeToString(digest[:]) {
		t.Fatalf("identity: %+v", p)
	}
	if discover(t, root+string(os.PathSeparator)+".") != p {
		t.Fatal("path alias changed identity")
	}
	if runtime.GOOS == "windows" && discover(t, strings.ToLower(root)) != p {
		t.Fatal("case alias changed identity")
	}
}

func TestSameNamesAndNestedRepository(t *testing.T) {
	base := t.TempDir()
	first, second := filepath.Join(base, "a", "same"), filepath.Join(base, "b", "same")
	mkdir(t, filepath.Join(first, ".git"))
	mkdir(t, filepath.Join(second, ".git"))
	if discover(t, first).ID == discover(t, second).ID {
		t.Fatal("different repositories share ID")
	}
	nested := filepath.Join(first, "vendor", "nested")
	mkdir(t, filepath.Join(nested, ".git"))
	if discover(t, nested).ID == discover(t, first).ID {
		t.Fatal("nested repository not isolated")
	}
}

func TestGitFilesAndWorktrees(t *testing.T) {
	base := t.TempDir()
	mkdir(t, filepath.Join(base, "admin"))
	first, second := filepath.Join(base, "work1"), filepath.Join(base, "work2")
	for _, root := range []string{first, second} {
		mkdir(t, root)
		write(t, filepath.Join(root, ".git"), "gitdir: ../admin\n")
	}
	p, q := discover(t, first), discover(t, second)
	if p.ID == q.ID || p.Root == q.Root {
		t.Fatal("worktrees share identity")
	}
}

func TestInvalidProjectDoesNotWriteOrLeak(t *testing.T) {
	for _, value := range []string{"", "FAKE-SECRET-IN-BAD-MARKER", "gitdir: missing"} {
		root := t.TempDir()
		if value != "" {
			write(t, filepath.Join(root, ".git"), value)
		}
		before, _ := os.ReadDir(root)
		_, err := Discover(root)
		if err == nil {
			t.Fatal("invalid project accepted")
		}
		if strings.Contains(err.Error(), "FAKE-SECRET") {
			t.Fatal("marker content leaked")
		}
		after, _ := os.ReadDir(root)
		if len(before) != len(after) {
			t.Fatal("discovery wrote files")
		}
	}
}

func TestDirectoryJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows junction test")
	}
	base := t.TempDir()
	root, alias := filepath.Join(base, "physical"), filepath.Join(base, "alias")
	mkdir(t, filepath.Join(root, ".git"))
	if output, err := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", alias, root).CombinedOutput(); err != nil {
		t.Fatalf("create test junction: %v: %s", err, output)
	}
	t.Cleanup(func() { _ = os.Remove(alias) })
	if p, q := discover(t, root), discover(t, alias); p != q {
		t.Fatalf("junction changed identity: %+v / %+v", p, q)
	}
}
