package cli

import (
	"bytes"
	"errors"
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

func TestSecretCRUDCommandsAndFailures(t *testing.T) {
	root, data := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("APPDATA", data)
	secret := "FAKE-CLI-SECRET_日本語"
	a := app{readSecret: func(*os.File) ([]byte, error) { return []byte(secret), nil }}
	invoke := func(args ...string) (int, string) {
		var out, err bytes.Buffer
		code := a.run(args, nil, &out, &err)
		text := out.String() + err.String()
		if strings.Contains(text, secret) {
			t.Fatal("CLI leaked secret")
		}
		return code, text
	}
	if code, _ := invoke("set", "TOKEN"); code != 1 {
		t.Fatal("set without init accepted")
	}
	if code, _ := invoke("init"); code != 0 {
		t.Fatal("init failed")
	}
	if code, _ := invoke("set", "token"); code != 0 {
		t.Fatal("set failed")
	}
	if code, text := invoke("list"); code != 0 || text != "TOKEN\n" {
		t.Fatalf("list: %d %q", code, text)
	}
	if code, _ := invoke("init"); code != 0 {
		t.Fatal("repeat init failed")
	}
	if code, text := invoke("list"); code != 0 || text != "TOKEN\n" {
		t.Fatal("init lost secret")
	}
	before, err := os.ReadFile(filepath.Join(data, "dotenvless", "vault.dat"))
	if err != nil {
		t.Fatal(err)
	}
	a.readSecret = func(*os.File) ([]byte, error) { return nil, errors.New(secret) }
	if code, text := invoke("set", "TOKEN"); code != 1 || strings.Contains(text, secret) {
		t.Fatal("unsafe input error")
	}
	after, err := os.ReadFile(filepath.Join(data, "dotenvless", "vault.dat"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("read failure changed vault")
	}
	if code, _ := invoke("unset", "token"); code != 0 {
		t.Fatal("unset failed")
	}
	if code, text := invoke("list"); code != 0 || text != "" {
		t.Fatal("deleted key remains")
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 1 {
		t.Fatal("CLI created project files")
	}
}

func TestSetRejectsNonTerminal(t *testing.T) {
	root, data := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("APPDATA", data)
	var out, err bytes.Buffer
	if code := Run([]string{"init"}, nil, &out, &err); code != 0 {
		t.Fatal("init failed")
	}
	if code := Run([]string{"set", "TOKEN"}, nil, &out, &err); code != 1 {
		t.Fatal("nonterminal accepted")
	}
}

func TestSecretLineEditingCancellationAndLimits(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		fail        bool
	}{
		{"token\r", "token", false}, {"\r", "", false}, {"日本\b語\r", "日語", false},
		{"fake\x03", "", true}, {"unterminated", "", true}, {"bad\x00value\r", "", true},
		{"\xff\r", "", true}, {strings.Repeat("x", 32769) + "\r", "", true},
	} {
		value, err := readSecretLine(strings.NewReader(tc.input))
		if (err != nil) != tc.fail || (!tc.fail && string(value) != tc.want) {
			t.Fatalf("unexpected input result; fail=%v", tc.fail)
		}
	}
}

func TestRunInjectsSecretAndPreservesExit(t *testing.T) {
	root, data := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("APPDATA", data)
	t.Setenv("DVL_CLI_CHILD", "1")
	a := app{readSecret: func(*os.File) ([]byte, error) { return []byte("FAKE-CLI-RUN-SECRET"), nil }}
	var out, stderr bytes.Buffer
	for _, args := range [][]string{{"init"}, {"set", "DVL_CHILD_SECRET"}} {
		if code := a.run(args, nil, &out, &stderr); code != 0 {
			t.Fatal("setup failed")
		}
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	code := a.run([]string{"run", "--", binary, "-test.run=^TestCLIRunHelper$"}, nil, &out, &stderr)
	if code != 23 {
		t.Fatalf("child exit not preserved: %d %s", code, stderr.String())
	}
	if out.Len() != 0 || stderr.Len() != 0 {
		t.Fatal("wrapper printed child secret")
	}
	if os.Getenv("DVL_CHILD_SECRET") != "" {
		t.Fatal("secret changed parent environment")
	}
	for _, args := range [][]string{{"run"}, {"run", "--"}, {"run", binary}} {
		if code := a.run(args, nil, &out, &stderr); code != 2 {
			t.Fatal("invalid run grammar accepted")
		}
	}
}
func TestCLIRunHelper(t *testing.T) {
	if os.Getenv("DVL_CLI_CHILD") != "1" {
		return
	}
	if os.Getenv("DVL_CHILD_SECRET") != "FAKE-CLI-RUN-SECRET" {
		os.Exit(91)
	}
	os.Exit(23)
}

func TestImportCommandPreservesSourceAndRejectsConflicts(t *testing.T) {
	root, data := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("APPDATA", data)
	source := []byte("TOKEN=FAKE_IMPORT_VALUE\n")
	if err := os.WriteFile("input.env", source, 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := Run([]string{"init"}, nil, &out, &stderr); code != 0 {
		t.Fatal("init failed")
	}
	out.Reset()
	stderr.Reset()
	if code := Run([]string{"import", "input.env"}, nil, &out, &stderr); code != 0 {
		t.Fatalf("import exit=%d", code)
	}
	if strings.Contains(out.String()+stderr.String(), "FAKE_IMPORT_VALUE") {
		t.Fatal("import value leaked")
	}
	if code := Run([]string{"import", "input.env"}, nil, &out, &stderr); code != 1 {
		t.Fatal("conflicting import accepted")
	}
	if code := Run([]string{"import", "--overwrite", "input.env"}, nil, &out, &stderr); code != 0 {
		t.Fatal("explicit overwrite failed")
	}
	after, err := os.ReadFile("input.env")
	if err != nil || !bytes.Equal(after, source) {
		t.Fatal("source changed")
	}
}
