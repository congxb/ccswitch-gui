package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

func main() {
	a := app.NewWithID("com.ccswitch.gui")
	a.Settings().SetTheme(theme.DarkTheme())

	w := a.NewWindow("CCSwitch - Claude Code API 配置管理器")
	w.Resize(fyne.NewSize(700, 650))
	w.SetMinSize(fyne.NewSize(700, 650))

	configPath := defaultConfigPath()
	ui := createUI(w, configPath)
	if ui == nil {
		w.ShowAndRun()
		return
	}

	w.ShowAndRun()
}

var _ = theme.ColorNamePrimary
var _ fyne.Theme = nil
