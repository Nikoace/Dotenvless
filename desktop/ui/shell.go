package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func Sections() []string { return []string{"工程", "共享库", "状态与接入"} }
func Shell() fyne.CanvasObject {
	tabs := container.NewAppTabs()
	for _, label := range Sections() {
		tabs.Append(container.NewTabItem(label, widget.NewLabel("Dotenvless · 本地 ENV 管理中心")))
	}
	return tabs
}
