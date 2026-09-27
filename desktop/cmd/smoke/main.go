//go:build smoke

// Native fixture harness: no real Vault, clipboard writes, network calls or launch commands.
package main

import (
	"dotenvless/desktop/ui"
	"dotenvless/internal/application"
	"dotenvless/internal/crypto"
	"dotenvless/internal/vault"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(2)
	}
	root, e := os.MkdirTemp("", "dotenvless-native-")
	if e != nil {
		panic("fixture setup failed")
	}
	defer os.RemoveAll(root)
	profile := filepath.Join(root, "profile")
	os.MkdirAll(profile, 0700)
	os.Setenv("APPDATA", profile)
	projectRoot := filepath.Join(root, "示例工程")
	os.MkdirAll(projectRoot, 0700)
	s := application.New(filepath.Join(profile, "vault.dat"), crypto.DPAPI{})
	p, e := s.Register(projectRoot, 0)
	check(e)
	check(s.CreateEnvironment(p.ID, "dev", "", vault.AnyRevision))
	check(s.Set(p.ID, "base", "LOG_LEVEL", []byte("GENERATED_MASKED"), vault.AnyRevision))
	check(s.Set(p.ID, "base", "EMPTY_VALUE", []byte(""), vault.AnyRevision))
	shared, e := s.SaveShared("", "开发数据库", []byte("GENERATED_SHARED"), vault.AnyRevision)
	check(e)
	check(s.BindShared(p.ID, "dev", "DATABASE_URL", shared, vault.AnyRevision))
	check(s.Disable(p.ID, "dev", "OLD_TOKEN", vault.AnyRevision))
	a := app.NewWithID("io.dotenvless.fixture")
	w := a.NewWindow("Dotenvless · 界面验收（假数据）")
	w.Resize(fyne.NewSize(1280, 800))
	c := ui.NewClient(s, w, nil)
	c.SelectProject(p.ID)
	c.SelectScope("dev")
	a.Lifecycle().SetOnExitedForeground(c.HideSecrets)
	w.SetOnClosed(c.Close)
	time.AfterFunc(2*time.Second, func() {
		fyne.Do(func() {
			f, e := os.Create(os.Args[1])
			check(e)
			check(png.Encode(f, w.Canvas().Capture()))
			check(f.Close())
			fmt.Println("native canvas captured")
			w.Close()
		})
	})
	w.ShowAndRun()
}
func check(e error) {
	if e != nil {
		panic("fixture operation failed")
	}
}
