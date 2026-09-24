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
	if path != filepath.Join(appdata, "dotenvless", "vault.dat") {
		t.Fatal("unexpected vault path")
	}
	files, _ := os.ReadDir(appdata)
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
