package ui

import (
	"bytes"
	"dotenvless/internal/application"
	"dotenvless/internal/config"
	"dotenvless/internal/crypto"
	"dotenvless/internal/vault"
	"fmt"
	"fyne.io/fyne/v2/test"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopCLIEndToEnd(t *testing.T) {
	if os.Getenv("DVL_E2E_CHILD") == "1" {
		expected := os.Getenv("DVL_E2E_EXPECT")
		scope := os.Getenv("DVL_E2E_SCOPE")
		if os.Getenv("BASE_KEY") != expected+"_BASE" {
			os.Exit(81)
		}
		if scope == "dev" {
			if os.Getenv("ALIAS") != expected+"_SHARED" || os.Getenv("TOKEN") != expected+"_DEV" {
				os.Exit(82)
			}
			if _, ok := os.LookupEnv("DROP"); ok {
				os.Exit(83)
			}
			if v, ok := os.LookupEnv("EMPTY"); !ok || v != "" {
				os.Exit(84)
			}
		} else if os.Getenv("TOKEN") != expected+"_LOCAL" || os.Getenv("DROP") != "PARENT_SENTINEL" {
			os.Exit(85)
		}
		cwd, e := os.Getwd()
		if e != nil {
			os.Exit(86)
		}
		a, e := os.Stat(cwd)
		if e != nil {
			os.Exit(86)
		}
		b, e := os.Stat(os.Getenv("DVL_E2E_CWD"))
		if e != nil || !os.SameFile(a, b) {
			os.Exit(87)
		}
		os.Exit(47)
	}
	cli := os.Getenv("DVL_E2E_CLI")
	if cli == "" {
		t.Skip("set DVL_E2E_CLI to the built Windows CLI for process integration")
	}
	storage := t.TempDir()
	t.Setenv("APPDATA", storage)
	path, e := config.DefaultVaultPath()
	if e != nil {
		t.Fatal(e)
	}
	s := application.New(path, crypto.DPAPI{})
	dir := t.TempDir()
	sub := filepath.Join(dir, "子目录")
	if e = os.Mkdir(sub, 0700); e != nil {
		t.Fatal(e)
	}
	p, e := s.Register(dir, 0)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CreateEnvironment(p.ID, "dev", "", vault.AnyRevision); e != nil {
		t.Fatal(e)
	}
	nonce, e := vault.NewID()
	if e != nil {
		t.Fatal(e)
	}
	secret := "GENERATED_" + nonce
	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("integration")
	defer w.Close()
	c := NewClient(s, w, fakeClip{})
	defer c.Close()
	c.SelectProject(p.ID)
	save := func(scope, key, value string) {
		c.SelectScope(scope)
		c.Key.SetText(key)
		c.Value.SetText(value)
		test.Tap(c.SaveButton)
		if c.Value.Text != "" {
			t.Fatal("editor retained value")
		}
	}
	save("base", "BASE_KEY", secret+"_BASE")
	save("local", "TOKEN", secret+"_LOCAL")
	save("dev", "TOKEN", secret+"_DEV")
	save("dev", "EMPTY", "")
	shared, e := s.SaveShared("", "fixture", []byte(secret+"_SHARED"), vault.AnyRevision)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Disable(p.ID, "dev", "DROP", vault.AnyRevision); e != nil {
		t.Fatal(e)
	}
	if e = c.Refresh(); e != nil {
		t.Fatal(e)
	}
	c.Key.SetText("ALIAS")
	for label, id := range c.sharedLabels {
		if id == shared {
			c.reference.SetSelected(label)
		}
	}
	c.bindShared()
	if _, e = c.PrepareImport(strings.NewReader("IMPORTED=" + secret + "_IMPORT")); e != nil {
		t.Fatal(e)
	}
	if e = c.ApplyImport(false); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(fmt.Sprint(c.view), secret) || strings.Contains(c.Message.Text, secret) {
		t.Fatal("value leaked into view")
	}
	self, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	t.Setenv("DROP", "PARENT_SENTINEL")
	for _, scope := range []string{"dev", "local"} {
		args := []string{"run", "--", self, "-test.run=^TestDesktopCLIEndToEnd$"}
		if scope != "local" {
			args = append([]string{"--env", scope}, args...)
		}
		cmd := exec.Command(cli, args...)
		cmd.Dir = sub
		cmd.Env = append(os.Environ(), "DVL_E2E_CHILD=1", "DVL_E2E_EXPECT="+secret, "DVL_E2E_SCOPE="+scope, "DVL_E2E_CWD="+sub)
		output, e := cmd.CombinedOutput()
		exit, ok := e.(*exec.ExitError)
		if !ok || exit.ExitCode() != 47 || len(output) != 0 {
			t.Fatal("real CLI child failed environment, cwd, silence or exit contract")
		}
	}
	if os.Getenv("DROP") != "PARENT_SENTINEL" {
		t.Fatal("parent environment changed")
	}
	for _, command := range []string{"list", "status", "example"} {
		cmd := exec.Command(cli, "--env", "dev", command)
		cmd.Dir = dir
		output, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("CLI %s failed", command)
		}
		if bytes.Contains(output, []byte(secret)) {
			t.Fatal("CLI metadata leaked a value")
		}
	}

	// Real CLI processes and the UI compete for the same OS lock and revision.
	for _, key := range []string{"PEER_A", "PEER_B"} {
		if e = s.Set(p.ID, "dev", key, []byte(secret+"_PEER"), vault.AnyRevision); e != nil {
			t.Fatal(e)
		}
	}
	if e = c.Refresh(); e != nil {
		t.Fatal(e)
	}
	peers := []*exec.Cmd{}
	peerLogs := []*bytes.Buffer{}
	for _, key := range []string{"PEER_A", "PEER_B"} {
		peer := exec.Command(cli, "--env", "dev", "unset", key)
		peer.Dir = dir
		log := &bytes.Buffer{}
		peer.Stdout = log
		peer.Stderr = log
		if e = peer.Start(); e != nil {
			t.Fatal("cannot start CLI peer")
		}
		peers = append(peers, peer)
		peerLogs = append(peerLogs, log)
	}
	c.Key.SetText("GUI_PEER")
	c.Value.SetText(secret + "_GUI")
	test.Tap(c.SaveButton)
	for i, peer := range peers {
		if e = peer.Wait(); e != nil {
			t.Fatal("concurrent CLI update failed")
		}
		if bytes.Contains(peerLogs[i].Bytes(), []byte(secret)) {
			t.Fatal("peer output leaked value")
		}
	}
	current, e := s.Resolve(p.ID, "dev")
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := current.Values["PEER_A"]; ok {
		t.Fatal("peer update lost")
	}
	if _, ok := current.Values["PEER_B"]; ok {
		t.Fatal("peer update lost")
	}
	if _, ok := current.Values["GUI_PEER"]; !ok && !strings.Contains(c.Message.Text, "刷新") {
		t.Fatal("GUI conflict was not explained")
	}

	for _, root := range []string{storage, dir} {
		e = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			data, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			if bytes.Contains(data, []byte(secret)) {
				return fmt.Errorf("plaintext persisted in application artifact")
			}
			return nil
		})
		if e != nil {
			t.Fatal(e)
		}
	}
}
