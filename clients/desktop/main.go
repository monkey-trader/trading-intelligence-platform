package main

import (
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

func main() {
	app := &App{}

	err := wails.Run(&options.App{
		Title:                    "Trading Intelligence Platform",
		Width:                    1440,
		Height:                   900,
		MinWidth:                 960,
		MinHeight:                600,
		BackgroundColour:         &options.RGBA{R: 40, G: 40, B: 40, A: 1}, // Gruvbox dunkel, verhindert weißes Aufblitzen
		AssetServer:              &assetserver.Options{Handler: newServerProxy(serverURL())},
		Menu:                     buildMenu(app),
		OnStartup:                app.startup,
		OnBeforeClose:            app.beforeClose,
		EnableDefaultContextMenu: false,
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Trading Intelligence Platform",
				Message: "Persönliches Trading-Journal mit BingX-Import",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
