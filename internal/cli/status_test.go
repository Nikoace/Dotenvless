package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusDirectorySelectsProjectWithoutChangingCWDOrVault(t *testing.T) {
	base, data := t.TempDir(), t.TempDir()
	current, target := filepath.Join(base, "Current"), filepath.Join(base, "Target 日本 项目")
	t.Setenv("APPDATA", data)
	a := app{readSecret: func(*os.File) ([]byte, error) { return []byte("FAKE_STATUS_DIRECTORY_VALUE"), nil }}
	for _, fixture := range []struct{ root, key, file string }{
		{current, "CURRENT_ONLY", ".env.current"}, {target, "TARGET_ONLY", ".env.target"},
	} {
		if err := os.MkdirAll(filepath.Join(fixture.root, ".git"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fixture.root, fixture.file), []byte("FAKE_SOURCE_CONTENT"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Chdir(fixture.root)
		var out, stderr bytes.Buffer
		for _, args := range [][]string{{"init"}, {"set", fixture.key}} {
			if code := a.run(args, nil, &out, &stderr); code != 0 {
				t.Fatal("fixture initialization failed")
			}
		}
	}
	sub := filepath.Join(target, "src")
	if err := os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	vaultPath := filepath.Join(data, "dotenvless", "vault.dat")
	before, err := os.ReadFile(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, cwd, wantKey, otherKey, wantFile, otherFile string
		args                                              []string
	}{
		{"default", current, "CURRENT_ONLY", "TARGET_ONLY", ".env.current", ".env.target", []string{"status"}},
		{"dot", current, "CURRENT_ONLY", "TARGET_ONLY", ".env.current", ".env.target", []string{"status", "."}},
		{"absolute", current, "TARGET_ONLY", "CURRENT_ONLY", ".env.target", ".env.current", []string{"status", target}},
		{"relative", current, "TARGET_ONLY", "CURRENT_ONLY", ".env.target", ".env.current", []string{"status", filepath.Join("..", filepath.Base(target))}},
		{"subdirectory", current, "TARGET_ONLY", "CURRENT_ONLY", ".env.target", ".env.current", []string{"status", sub}},
		{"outside-git", base, "TARGET_ONLY", "CURRENT_ONLY", ".env.target", ".env.current", []string{"status", target}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(tc.cwd)
			cwdBefore, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			if code := Run(tc.args, nil, &out, &stderr); code != 0 {
				t.Fatalf("status exit=%d error=%s", code, stderr.String())
			}
			text := out.String() + stderr.String()
			if !strings.Contains(text, tc.wantKey) || strings.Contains(text, tc.otherKey) || !strings.Contains(text, tc.wantFile) || strings.Contains(text, tc.otherFile) {
				t.Fatal("status selected wrong project")
			}
			if strings.Contains(text, "FAKE_STATUS_DIRECTORY_VALUE") || strings.Contains(text, "FAKE_SOURCE_CONTENT") {
				t.Fatal("status exposed value")
			}
			cwdAfter, err := os.Getwd()
			if err != nil || cwdAfter != cwdBefore {
				t.Fatal("status changed working directory")
			}
			after, err := os.ReadFile(vaultPath)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("status changed vault")
			}
		})
	}
}

func TestStatusDirectoryErrorsDoNotFallBackOrEchoArguments(t *testing.T) {
	base, data := t.TempDir(), t.TempDir()
	current := filepath.Join(base, "Current")
	if err := os.MkdirAll(filepath.Join(current, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(base, "FAKE_ARGUMENT_NON_GIT")
	if err := os.Mkdir(plain, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(base, "FAKE_ARGUMENT_FILE")
	if err := os.WriteFile(file, []byte("FAKE_FILE_CONTENT"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPDATA", data)
	t.Chdir(current)
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"missing", []string{"status", filepath.Join(base, "FAKE_ARGUMENT_MISSING")}, 1},
		{"file", []string{"status", file}, 1},
		{"non-git", []string{"status", plain}, 1},
		{"empty", []string{"status", ""}, 2},
		{"unknown-option", []string{"status", "--FAKE_ARGUMENT_OPTION"}, 2},
		{"extra", []string{"status", current, "FAKE_ARGUMENT_EXTRA"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			if code := Run(tc.args, nil, &out, &stderr); code != tc.code {
				t.Errorf("exit=%d want=%d", code, tc.code)
			}
			if out.Len() != 0 || strings.Contains(stderr.String(), "FAKE_ARGUMENT") {
				t.Fatal("invalid target was displayed or fell back")
			}
		})
	}
	files, err := os.ReadDir(data)
	if err != nil || len(files) != 0 {
		t.Fatal("invalid status created vault state")
	}
}

func TestStatusHelpDescribesOptionalDirectory(t *testing.T) {
	var out, stderr bytes.Buffer
	if Run([]string{"--help"}, nil, &out, &stderr) != 0 || !strings.Contains(out.String(), "status [DIRECTORY]") {
		t.Fatal("help does not describe optional status directory")
	}
}
