package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	app := NewApp()

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--game" && i+1 < len(args) {
			app.spoofName = args[i+1]
			break
		}
	}

	err := wails.Run(&options.App{
		Title:  "XXMI Mod Manager",
		Width:  1600,
		Height: 900,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
		Logger:             nil,
		LogLevel:           logger.ERROR,
	})

	if err != nil {
		log.Fatal("Error:", err)
	}
}