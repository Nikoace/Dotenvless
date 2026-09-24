package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAndVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"help", []string{"--help"}, "Usage: dvl"},
		{"short help", []string{"-h"}, "Usage: dvl"},
		{"version", []string{"--version"}, "dvl " + Version},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, err bytes.Buffer
			if code := Run(tc.args, nil, &out, &err); code != 0 {
				t.Fatalf("exit = %d", code)
			}
			if !strings.Contains(out.String(), tc.want) || err.Len() != 0 {
				t.Fatalf("unexpected output: %q / %q", out.String(), err.String())
			}
		})
	}
}

func TestInvalidArgumentsDoNotEchoInput(t *testing.T) {
	secret := "FAKE-CLI-SECRET-NEVER-ECHO"
	for _, args := range [][]string{nil, {secret}, {"--help", secret}, {"--version", secret}, {"set", "TOKEN", secret}} {
		var out, err bytes.Buffer
		if code := Run(args, nil, &out, &err); code != 2 {
			t.Errorf("invalid arguments exit = %d, want 2", code)
		}
		if strings.Contains(out.String()+err.String(), secret) {
			t.Fatal("argument leaked")
		}
	}
}

func TestStatusReportsIdentityWithoutVault(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	appdata := t.TempDir()
	t.Setenv("APPDATA", appdata)
	t.Chdir(root)
	var out, err bytes.Buffer
	if code := Run([]string{"status"}, nil, &out, &err); code != 0 {
		t.Fatalf("exit=%d, error=%s", code, err.String())
	}
	for _, label := range []string{"Project:", "Root:", "ID:"} {
		if !strings.Contains(out.String(), label) {
			t.Errorf("missing %s", label)
		}
	}
	files, readErr := os.ReadDir(appdata)
	if readErr != nil || len(files) != 0 {
		t.Fatal("status created vault state")
	}
}
