/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

var AppVersion = "dev"

func main() {
	backend := NewApp()
	app := application.New(application.Options{
		Name:        "elabftw-desktop",
		Description: "Local-first eLabFTW desktop client.",
		Services: []application.Service{
			application.NewService(backend),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	appMenu := app.NewMenu()
	about := func(_ *application.Context) {
		app.Dialog.Info().
			SetTitle("About eLabFTW Desktop").
			SetMessage("Version: " + AppVersion + "\n\nLocal-first eLabFTW desktop client.\nDevelopment sponsored by CNRS.").
			Show()
	}

	if runtime.GOOS == "darwin" {
		appSubmenu := appMenu.AddSubmenu("eLabFTW Desktop")
		appSubmenu.Add("About...").OnClick(about)
		appSubmenu.AddSeparator()
		appSubmenu.AddRole(application.Quit)

		appMenu.AddRole(application.EditMenu)

		viewMenu := appMenu.AddSubmenu("View")
		viewMenu.AddRole(application.ToggleFullscreen)
	} else {
		fileMenu := appMenu.AddSubmenu("File")
		fileMenu.AddRole(application.Quit)

		viewMenu := appMenu.AddSubmenu("View")
		viewMenu.AddRole(application.ToggleFullscreen)

		helpMenu := appMenu.AddSubmenu("Help")
		helpMenu.Add("About...").OnClick(about)
	}
	app.Menu.Set(appMenu)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:              "elabftw-desktop",
		Width:              1200,
		Height:             900,
		BackgroundColour:   application.NewRGB(27, 38, 54),
		URL:                "/",
		UseApplicationMenu: true,
	})

	if err := app.Run(); err != nil {
		println("Error:", err.Error())
	}
}
