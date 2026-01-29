package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

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
		Bind: []interface{}{
			app,
		},
	})
	/*
		idx, err := loadProfileIndex()
		if err != nil {
			panic(err)
		}

			// Example: add a profile entry (you will generate a real UUIDv4 elsewhere)
			entry := ProfileEntry{
				UUID:        "11111111-2222-3333-4444-555555555555",
				CreatedAt:   time.Now(),
				DisplayName: "My Profile",
			}
			idx.Profiles = append(idx.Profiles, entry)

			if err := saveProfileIndex(idx); err != nil {
				panic(err)
			}

			metaPath, err := writeProfileMetaFile(entry.UUID, []byte(`{"version":1}`))
			if err != nil {
				panic(err)
			}
	*/

	//fmt.Println("Wrote meta:", metaPath)

	if err != nil {
		println("Error:", err.Error())
	}
}
