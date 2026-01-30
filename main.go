package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	rt "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

var AppVersion = "dev"

func main() {
	// Create an instance of the app structure
	app := NewApp()
	appMenu := menu.NewMenu()
	about := func() {
		_, _ = rt.MessageDialog(app.ctx, rt.MessageDialogOptions{
			Type:    rt.InfoDialog,
			Title:   "About eLabFTW Desktop",
			Message: "Version: " + AppVersion + "\n\nLocal-first eLabFTW desktop client." + "\nDevelopment sponsored by CNRS.",
			Buttons: []string{"OK"},
		})
	}
	if runtime.GOOS == "darwin" {
		appSubmenu := appMenu.AddSubmenu("eLabFTW Desktop")
		appSubmenu.AddText("About...", nil, func(_ *menu.CallbackData) {
			about()
		})
		appSubmenu.AddSeparator()
		appSubmenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			rt.Quit(app.ctx)
		})

		appMenu.Append(menu.EditMenu())
	} else {
		fileMenu := appMenu.AddSubmenu("File")
		fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			rt.Quit(app.ctx)
		})

		helpMenu := appMenu.AddSubmenu("Help")
		helpMenu.AddText("About...", nil, func(_ *menu.CallbackData) {
			about()
		})
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "elabftw-desktop",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Menu:             appMenu,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
