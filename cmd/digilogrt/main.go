package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	"github.com/unklstewy/digiLog/internal/config"
	"github.com/unklstewy/digiLog/internal/ui"
)

func main() {
	log.Println("Starting DigiLogRT...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Warning: Could not load config, using defaults: %v", err)
		cfg = config.GetDefaultConfig()
	}
	log.Printf("Configuration loaded: %s v%s", cfg.App.Name, cfg.App.Version)

	// Create the Fyne application
	myApp := app.New()
	myApp.Settings().SetTheme(theme.DefaultTheme())

	// Create main window
	myWindow := myApp.NewWindow("DigiLogRT - Digital Log Road Trip")
	myWindow.Resize(fyne.NewSize(float32(cfg.Window.Width), float32(cfg.Window.Height)))

	// Create the main tabbed interface with config
	mainTabs := ui.NewMainTabs(cfg)
	myWindow.SetContent(mainTabs.GetContainer())

	// Set window properties
	myWindow.SetFixedSize(false)
	myWindow.CenterOnScreen()

	log.Println("Window initialized with functional APRS tab...")
	myWindow.ShowAndRun()
}
