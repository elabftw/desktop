/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas <Deltablot>
 * @author Moustapha <Deltablot>
 * @copyright 2026 Deltablot
 * @see https://www.elabftw.net
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

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

	// full screen
	toggleFullscreen := func() {
		if rt.WindowIsFullscreen(app.ctx) {
			rt.WindowUnfullscreen(app.ctx)
			return
		}
		rt.WindowFullscreen(app.ctx)
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

		// toggle full screen
		viewMenu := appMenu.AddSubmenu("View")
		viewMenu.AddText("Toggle Full Screen", keys.Combo("f", keys.CmdOrCtrlKey, keys.ControlKey), func(_ *menu.CallbackData) {
			toggleFullscreen()
		})
	} else {
		fileMenu := appMenu.AddSubmenu("File")
		fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			rt.Quit(app.ctx)
		})

		viewMenu := appMenu.AddSubmenu("View")
		viewMenu.AddText("Toggle Full Screen", keys.Key("F11"), func(_ *menu.CallbackData) {
			toggleFullscreen()
		})

		helpMenu := appMenu.AddSubmenu("Help")
		helpMenu.AddText("About...", nil, func(_ *menu.CallbackData) {
			about()
		})
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "elabftw-desktop",
		Width:  1200,
		Height: 900,
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
