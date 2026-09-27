package ui

import (
	"dotenvless/internal/application"
	"dotenvless/internal/crypto"
	"dotenvless/internal/vault"
	"fmt"
	"fyne.io/fyne/v2/test"
	"path/filepath"
	"strings"
	"testing"
)

type fakeClip struct{}

func (fakeClip) SetSensitive([]byte) (uint32, error) { return 1, nil }
func (fakeClip) ClearOwned(uint32) error             { return nil }
func TestClientEditScopeAndStaleView(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("test")
	defer w.Close()
	s := application.New(filepath.Join(t.TempDir(), "vault.dat"), crypto.DPAPI{})
	p, e := s.Register(t.TempDir(), 0)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CreateEnvironment(p.ID, "dev", "", vault.AnyRevision); e != nil {
		t.Fatal(e)
	}
	c := NewClient(s, w, fakeClip{})
	defer c.Close()
	c.SelectProject(p.ID)
	c.SelectScope("dev")
	c.Key.SetText("TOKEN")
	c.Value.SetText("GENERATED_UI_SECRET")
	test.Tap(c.SaveButton)
	got, e := s.Resolve(p.ID, "dev")
	if e != nil {
		t.Fatal(e)
	}
	if got.Values["TOKEN"] != "GENERATED_UI_SECRET" {
		t.Fatal("GUI save did not target selected environment")
	}
	if strings.Contains(fmt.Sprint(c.view), "GENERATED_UI_SECRET") || c.Value.Text != "" {
		t.Fatal("saved value retained in view/editor")
	}
	c.Key.SetText("TOKEN")
	c.Value.SetText("GENERATED_STALE")
	if e = s.Set(p.ID, "dev", "TOKEN", []byte("GENERATED_NEWER"), vault.AnyRevision); e != nil {
		t.Fatal(e)
	}
	test.Tap(c.SaveButton)
	got, e = s.Resolve(p.ID, "dev")
	if e != nil {
		t.Fatal(e)
	}
	if got.Values["TOKEN"] != "GENERATED_NEWER" || !strings.Contains(c.Message.Text, "刷新") {
		t.Fatal("stale GUI overwrote external update")
	}
}
func TestClientNavigationCancelsImport(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("test")
	defer w.Close()
	s := application.New(filepath.Join(t.TempDir(), "vault.dat"), crypto.DPAPI{})
	p, e := s.Register(t.TempDir(), 0)
	if e != nil {
		t.Fatal(e)
	}
	q, e := s.Register(t.TempDir(), vault.AnyRevision)
	if e != nil {
		t.Fatal(e)
	}
	c := NewClient(s, w, fakeClip{})
	defer c.Close()
	c.SelectProject(p.ID)
	preview, e := c.PrepareImport(strings.NewReader("KEY=GENERATED_IMPORT"))
	if e != nil {
		t.Fatal(e)
	}
	if len(preview.Keys) != 1 {
		t.Fatal("missing preview")
	}
	c.SelectProject(q.ID)
	if c.ApplyImport(false) == nil {
		t.Fatal("navigation did not cancel pending import")
	}
	got, e := s.Resolve(p.ID, "local")
	if e != nil || len(got.Values) != 0 {
		t.Fatal("cancelled import changed original project")
	}
}

func TestSharedRefreshAndSecretControls(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("test")
	defer w.Close()
	s := application.New(filepath.Join(t.TempDir(), "vault.dat"), crypto.DPAPI{})
	p, e := s.Register(t.TempDir(), 0)
	if e != nil {
		t.Fatal(e)
	}
	shared, e := s.SaveShared("", "original", []byte("GENERATED_SHARED"), vault.AnyRevision)
	if e != nil {
		t.Fatal(e)
	}
	c := NewClient(s, w, fakeClip{})
	defer c.Close()
	c.SelectProject(p.ID)
	c.sharedList.Select(0)
	if e = s.RenameShared(shared, "updated", vault.AnyRevision); e != nil {
		t.Fatal(e)
	}
	if e = c.Refresh(); e != nil {
		t.Fatal(e)
	}
	if c.sharedName.Text != "updated" {
		t.Fatal("refresh retained stale shared editor")
	}
	c.Key.SetText("ALIAS")
	c.reference.SetSelected(c.reference.Options[0])
	c.bindShared()
	got, e := s.Resolve(p.ID, "local")
	if e != nil || got.Values["ALIAS"] != "GENERATED_SHARED" {
		t.Fatal("GUI reference did not resolve")
	}
	c.Key.SetText("ALIAS")
	c.revealValue()
	if c.revealed.Text != "GENERATED_SHARED" {
		t.Fatal("single reveal failed")
	}
	c.HideSecrets()
	if strings.Contains(c.revealed.Text, "GENERATED") {
		t.Fatal("focus hide retained reveal")
	}
	c.Value.SetText("GENERATED_DRAFT")
	if c.Value.ActionItem == nil {
		t.Fatal("editor exposes unrestricted password eye")
	}
}

func TestFilteredProjectSelectionUsesVisibleIdentity(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("test")
	defer w.Close()
	s := application.New(filepath.Join(t.TempDir(), "vault.dat"), crypto.DPAPI{})
	p, e := s.Register(t.TempDir(), 0)
	if e != nil {
		t.Fatal(e)
	}
	q, e := s.Register(t.TempDir(), vault.AnyRevision)
	if e != nil {
		t.Fatal(e)
	}
	c := NewClient(s, w, fakeClip{})
	defer c.Close()
	for i, item := range c.visible {
		if item.ID == p.ID {
			c.projects.Select(i)
			break
		}
	}
	c.search.SetText(q.Root)
	if len(c.visible) != 1 {
		t.Fatal("search did not filter directory")
	}
	c.projects.Select(0)
	if c.selected != q.ID {
		t.Fatal("filtered row retained previous project selection")
	}
}
