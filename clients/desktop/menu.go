package main

import (
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

func buildMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()
	appMenu.Append(menu.AppMenu())

	viewMenu := appMenu.AddSubmenu("Ansicht")
	viewMenu.AddText("Neu laden", keys.CmdOrCtrl("r"), func(*menu.CallbackData) { app.reload() })

	appMenu.Append(menu.EditMenu())
	appMenu.Append(menu.WindowMenu())
	return appMenu
}
