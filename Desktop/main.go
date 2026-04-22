package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	app := NewApp(Dependencies{})

	err := wails.Run(&options.App{
		Title:  "NetflixTorrent",
		Width:  1440,
		Height: 920,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			app,
		},
		OnStartup: func(ctx context.Context) {
			if err := app.StartRuntime(ctx); err != nil {
				log.Printf("desktop runtime startup failed: %v", err)
			}
		},
		OnShutdown: func(ctx context.Context) {
			if err := app.Shutdown(ctx); err != nil {
				log.Printf("desktop runtime shutdown failed: %v", err)
			}
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
