package ui

import (
	"dotenvless/internal/application"
	"dotenvless/internal/crypto"
	"dotenvless/internal/vault"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestMaskedCanvas(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("Dotenvless")
	defer w.Close()
	w.Resize(fyne.NewSize(1280, 800))
	s := application.New(filepath.Join(t.TempDir(), "vault.dat"), crypto.DPAPI{})
	p, e := s.Register(t.TempDir(), 0)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CreateEnvironment(p.ID, "dev", "", vault.AnyRevision); e != nil {
		t.Fatal(e)
	}
	for _, key := range []string{"DATABASE_URL", "LOG_LEVEL", "EMPTY_VALUE"} {
		if e = s.Set(p.ID, "base", key, []byte("GENERATED_MASKED"), vault.AnyRevision); e != nil {
			t.Fatal(e)
		}
	}
	shared, e := s.SaveShared("", "开发数据库", []byte("GENERATED_SHARED"), vault.AnyRevision)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.BindShared(p.ID, "dev", "DATABASE_URL", shared, vault.AnyRevision); e != nil {
		t.Fatal(e)
	}
	if e = s.Disable(p.ID, "dev", "OLD_TOKEN", vault.AnyRevision); e != nil {
		t.Fatal(e)
	}
	c := NewClient(s, w, fakeClip{})
	defer c.Close()
	c.SelectProject(p.ID)
	c.SelectScope("dev")
	w.Show()
	if len(c.view.Entries) != 4 {
		t.Fatal("masked view missing entries")
	}
	if output := os.Getenv("DVL_GUI_CAPTURE"); output != "" {
		f, e := os.Create(output)
		if e != nil {
			t.Fatal(e)
		}
		e = png.Encode(f, w.Canvas().Capture())
		closeErr := f.Close()
		if e != nil || closeErr != nil {
			t.Fatal("cannot capture fixture canvas")
		}
	}
}
