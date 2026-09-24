package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture git: %v: %s", err, out)
	}
}
func write(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestWorkspaceScopeAndGitStates(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "--quiet")
	write(t, root, ".gitignore", ".env\n.env.*\n!.env.example\n")
	for _, name := range []string{".env", "config/.env.production", ".env.example", ".env.local.example", "node_modules/dep/.env", "dist/.env", "application.yml"} {
		write(t, root, name, "FAKE_WORKSPACE_VALUE")
	}
	report, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report.Files, []string{".env", "config/.env.production"}) {
		t.Fatalf("scope: %v", report.Files)
	}
	if report.Git[".env"] != "ignored" || report.Git["config/.env.production"] != "ignored" {
		t.Fatalf("ignore state: %v", report.Git)
	}
	git(t, root, "add", "--force", "--", ".env")
	report, err = Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Git[".env"] != "tracked" {
		t.Fatal("tracked file incorrectly marked ignored")
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	report, err = Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Git["config/.env.production"] != "not ignored" {
		t.Fatal("not ignored state missing")
	}
}
func TestDirectoryLinksAreNotFollowed(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	git(t, root, "init", "--quiet")
	write(t, outside, ".env", "FAKE_OUTSIDE")
	alias := filepath.Join(root, "external")
	if output, err := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", alias, outside).CombinedOutput(); err != nil {
		t.Fatalf("junction fixture: %v %s", err, output)
	}
	t.Cleanup(func() { _ = os.Remove(alias) })
	report, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) != 0 {
		t.Fatal("followed outside directory link")
	}
}
