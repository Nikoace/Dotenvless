package cli

import (
	"bytes"
	"dotenvless/internal/application"
	"dotenvless/internal/crypto"
	"dotenvless/internal/project"
	"dotenvless/internal/vault"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCentreCLIEnvironmentSelectionAndParentMask(t *testing.T) {
	root, data := t.TempDir(), t.TempDir()
	t.Chdir(root)
	t.Setenv("APPDATA", data)
	t.Setenv("DVL_V2_CHILD", "1")
	t.Setenv("REMOVE_ME", "GENERATED_PARENT")
	s := application.New(filepath.Join(data, "dotenvless", "vault.dat"), crypto.DPAPI{})
	p, e := s.Register(root, 0)
	if e != nil {
		t.Fatal(e)
	}
	for _, e := range []error{s.CreateEnvironment(p.ID, "test", "", vault.AnyRevision), s.Set(p.ID, "base", "TOKEN", []byte("GENERATED_RUN"), vault.AnyRevision), s.Disable(p.ID, "test", "REMOVE_ME", vault.AnyRevision)} {
		if e != nil {
			t.Fatal(e)
		}
	}
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"--env", "test", "run", "--", binary, "-test.run=^TestCentreCLIHelper$"}, nil, &out, &stderr)
	if code != 47 {
		t.Fatalf("scoped run exit=%d; %s", code, stderr.String())
	}
	if out.Len() != 0 || stderr.Len() != 0 {
		t.Fatal("run emitted unexpected output")
	}
	if os.Getenv("REMOVE_ME") != "GENERATED_PARENT" {
		t.Fatal("parent changed")
	}
	for _, args := range [][]string{{"--base", "run", "--", "x"}, {"--env", "BAD VALUE", "list"}, {"--base", "--env", "test", "list"}} {
		if c := Run(args, nil, &out, &stderr); c != 2 {
			t.Fatal("invalid selector accepted")
		}
	}
}
func TestCentreCLIHelper(t *testing.T) {
	if os.Getenv("DVL_V2_CHILD") != "1" {
		return
	}
	if os.Getenv("TOKEN") != "GENERATED_RUN" {
		os.Exit(91)
	}
	if _, ok := os.LookupEnv("REMOVE_ME"); ok {
		os.Exit(92)
	}
	os.Exit(47)
}
func TestCentreCLILegacyReadsAndExplicitMigration(t *testing.T) {
	root, data := t.TempDir(), t.TempDir()
	if e := os.Mkdir(filepath.Join(root, ".git"), 0700); e != nil {
		t.Fatal(e)
	}
	t.Chdir(root)
	t.Setenv("APPDATA", data)
	p, e := project.Discover(root)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(data, "dotenvless", "vault.dat")
	v := vault.New(path, crypto.DPAPI{})
	if e = v.Init(p.ID); e != nil {
		t.Fatal(e)
	}
	if e = v.Set(p.ID, "OLD", []byte("GENERATED_LEGACY")); e != nil {
		t.Fatal(e)
	}
	original, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var out, stderr bytes.Buffer
	if c := Run([]string{"list"}, nil, &out, &stderr); c != 0 || out.String() != "OLD\n" {
		t.Fatal("v1 read compatibility lost")
	}
	if c := Run([]string{"unset", "OLD"}, nil, &out, &stderr); c != 1 || !strings.Contains(stderr.String(), "migrate") {
		t.Fatal("legacy mutation did not request migration")
	}
	after, e := os.ReadFile(path)
	if e != nil || !bytes.Equal(after, original) {
		t.Fatal("legacy mutation changed file")
	}
	t.Chdir(t.TempDir())
	out.Reset()
	stderr.Reset()
	if c := Run([]string{"migrate"}, nil, &out, &stderr); c != 0 {
		t.Fatalf("migration from outside project failed: %s", stderr.String())
	}
	backup, e := os.ReadFile(path + ".v1.bak")
	if e != nil || !bytes.Equal(backup, original) {
		t.Fatal("backup lost")
	}
}
