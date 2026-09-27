package main

import (
	"dotenvless/desktop/privacy"
	"dotenvless/desktop/ui"
	"dotenvless/internal/application"
	"dotenvless/internal/config"
	"dotenvless/internal/crypto"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.NewWithID("io.dotenvless.desktop")
	w := a.NewWindow("Dotenvless · 本地 ENV 管理中心")
	w.Resize(fyne.NewSize(1280, 800))
	path, err := config.DefaultVaultPath()
	if err != nil {
		w.SetContent(widget.NewLabel("无法确定安全的 Vault 存储位置。请检查 APPDATA 配置。"))
	} else {
		clipboard := privacy.NewNativeClipboard(func() uintptr {
			var handle uintptr
			if native, ok := w.(driver.NativeWindow); ok {
				native.RunNative(func(context any) {
					if ctx, ok := context.(driver.WindowsWindowContext); ok {
						handle = ctx.HWND
					}
				})
			}
			return handle
		})
		client := ui.NewClient(application.New(path, crypto.DPAPI{}), w, clipboard)
		a.Lifecycle().SetOnExitedForeground(client.HideSecrets)
		w.SetOnClosed(client.Close)
	}
	w.ShowAndRun()
}
