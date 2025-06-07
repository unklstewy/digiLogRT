package ui

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/unklstewy/digiLogRT/internal/config"
)

type MainTabs struct {
	Container *container.AppTabs
	config    *config.Config
}

func NewMainTabs(cfg *config.Config) *MainTabs {
	tabs := container.NewAppTabs()

	mainTabs := &MainTabs{
		Container: tabs,
		config:    cfg,
	}

	// Dashboard tab
	dashboardContent := container.NewVBox(
		widget.NewLabel("DigiLogRT Dashboard"),
		widget.NewSeparator(),
		widget.NewLabel("System Status: Ready"),
		widget.NewLabel("API Connections: APRS Connected"),
		widget.NewLabel("Google Earth: Not Configured"),
	)
	tabs.Append(container.NewTabItem("Dashboard", dashboardContent))

	// Repeaters tab
	repeatersContent := container.NewVBox(
		widget.NewLabel("Repeater Information"),
		widget.NewSeparator(),
		widget.NewLabel("Search and display repeater data from:"),
		widget.NewLabel("• hearham.com"),
		widget.NewLabel("• RepeaterBook.com"),
		widget.NewLabel("Coming soon..."),
	)
	tabs.Append(container.NewTabItem("Repeaters", repeatersContent))

	// APRS tab - now functional!
	aprsTab := NewAPRSTab(cfg)
	tabs.Append(container.NewTabItem("APRS", aprsTab.GetContainer()))

	// DMR tab
	dmrContent := container.NewVBox(
		widget.NewLabel("DMR Networks"),
		widget.NewSeparator(),
		widget.NewLabel("Monitor DMR networks:"),
		widget.NewLabel("• Brandmeister.network"),
		widget.NewLabel("• TGIF.network"),
		widget.NewLabel("Coming soon..."),
	)
	tabs.Append(container.NewTabItem("DMR", dmrContent))

	// Google Earth tab
	earthContent := container.NewVBox(
		widget.NewLabel("Google Earth Integration"),
		widget.NewSeparator(),
		widget.NewLabel("Export data to Google Earth Pro"),
		widget.NewLabel("• Static KML files"),
		widget.NewLabel("• Live KML feeds"),
		widget.NewLabel("• KMZ bundles"),
		widget.NewLabel("Coming soon..."),
	)
	tabs.Append(container.NewTabItem("Google Earth", earthContent))

	// Settings tab
	settingsContent := container.NewVBox(
		widget.NewLabel("Settings & Configuration"),
		widget.NewSeparator(),
		widget.NewLabel("Configure API keys and preferences"),
		widget.NewLabel("Coming soon..."),
	)
	tabs.Append(container.NewTabItem("Settings", settingsContent))

	return mainTabs
}

func (mt *MainTabs) GetContainer() *container.AppTabs {
	return mt.Container
}
