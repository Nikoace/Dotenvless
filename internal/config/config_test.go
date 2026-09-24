package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVaultPathOutsideProjectWithoutWriting(t *testing.T) {
	project := t.TempDir()
	appdata := t.TempDir()
	t.Setenv("APPDATA", appdata)
	path, err := VaultPath(project)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "vault.dat" || filepath.Base(filepath.Dir(path)) != "dotenvless" {
		t.Fatal("unexpected vault path suffix")
	}
	actualBase, err := os.Stat(filepath.Dir(filepath.Dir(path)))
	if err != nil {
		t.Fatal(err)
	}
	appdataInfo, err := os.Stat(appdata)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(actualBase, appdataInfo) {
		t.Fatal("vault path resolves outside APPDATA")
	}
	files, err := os.ReadDir(appdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatal("configuration wrote files")
	}
}
func TestRejectProjectStorageAndRelativePaths(t *testing.T) {
	project := t.TempDir()
	for _, appdata := range []string{project, filepath.Join(project, "subdir"), ".", ""} {
		t.Setenv("APPDATA", appdata)
		if _, err := VaultPath(project); err == nil {
			t.Fatal("unsafe storage path accepted")
		}
	}
}
