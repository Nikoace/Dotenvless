package main

import (
	"dotenvless/desktop/ui"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.NewWithID("io.dotenvless.desktop")
	w := a.NewWindow("Dotenvless · 本地 ENV 管理中心")
	w.Resize(fyne.NewSize(1120, 720))
	w.SetContent(ui.Shell())
	w.ShowAndRun()
}
