package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/unklstewy/digiLog/internal/api"
	"github.com/unklstewy/digiLog/internal/config"
)

type APRSTab struct {
	client       *api.APRSClient
	searchEntry  *widget.Entry
	searchButton *widget.Button
	resultsText  *widget.RichText
	statusLabel  *widget.Label
}

func NewAPRSTab(cfg *config.Config) *APRSTab {
	// Create APRS client
	client := api.NewAPRSClient(cfg.APIs.AprsKey)

	// Create UI elements
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Enter callsign (e.g., W3MSG, VK9C/VK4AAA, OH7RDA)")

	// Set minimum size to accommodate longest possible callsigns
	// ITU regions can have callsigns up to 9-10 characters plus portable indicators
	searchEntry.Resize(fyne.NewSize(250, searchEntry.MinSize().Height))

	resultsText := widget.NewRichText()
	resultsText.Wrapping = fyne.TextWrapWord

	statusLabel := widget.NewLabel("Ready")

	aprsTab := &APRSTab{
		client:      client,
		searchEntry: searchEntry,
		resultsText: resultsText,
		statusLabel: statusLabel,
	}

	// Create search button
	aprsTab.searchButton = widget.NewButton("Search Station", aprsTab.searchStation)

	return aprsTab
}

func (a *APRSTab) searchStation() {
	callsign := a.searchEntry.Text
	if callsign == "" {
		a.statusLabel.SetText("Please enter a callsign")
		return
	}

	a.statusLabel.SetText("Searching...")
	a.searchButton.Disable()

	// Perform search in background
	go func() {
		response, err := a.client.GetStation(callsign)

		// Update UI in main thread
		if err != nil {
			log.Printf("APRS search error: %v", err)
			a.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			a.searchButton.Enable()
			return
		}

		// Format results
		var resultText string
		if response.Found == 0 {
			resultText = fmt.Sprintf("No stations found for '%s'", callsign)
		} else {
			resultText = fmt.Sprintf("Found %d station(s) for '%s':\n\n", response.Found, callsign)

			for i, station := range response.Entries {
				resultText += fmt.Sprintf("Station %d:\n", i+1)
				resultText += fmt.Sprintf("  Callsign: %s\n", station.Name)
				resultText += fmt.Sprintf("  Location: %.6f, %.6f\n", station.GetLatitude(), station.GetLongitude())
				resultText += fmt.Sprintf("  Last Heard: %s\n", station.GetLastTimeString())
				resultText += fmt.Sprintf("  Timestamp: %s\n", station.GetTimeString())
				if station.Comment != "" {
					resultText += fmt.Sprintf("  Comment: %s\n", station.Comment)
				}
				if station.Speed.Value > 0 {
					resultText += fmt.Sprintf("  Speed: %d km/h\n", station.Speed.Value)
				}
				if station.Course.Value > 0 {
					resultText += fmt.Sprintf("  Course: %d°\n", station.Course.Value)
				}
				resultText += "\n"
			}
		}

		a.resultsText.ParseMarkdown(resultText)
		a.statusLabel.SetText("Search completed")
		a.searchButton.Enable()
	}()
}

func (a *APRSTab) GetContainer() *fyne.Container {
	// Search section with better layout for callsign entry
	callsignLabel := widget.NewLabel("Callsign:")
	callsignLabel.Resize(fyne.NewSize(70, callsignLabel.MinSize().Height))

	// Create a container that gives the entry field more space
	searchForm := container.NewBorder(
		nil, nil, // top, bottom
		callsignLabel, a.searchButton, // left, right
		a.searchEntry, // center - this will expand to fill available space
	)

	// Results section
	resultsScroll := container.NewScroll(a.resultsText)
	resultsScroll.SetMinSize(fyne.NewSize(600, 300))

	// Status section
	statusSection := container.NewHBox(
		widget.NewLabel("Status:"),
		a.statusLabel,
	)

	return container.NewVBox(
		widget.NewLabel("APRS Station Tracking"),
		widget.NewSeparator(),
		searchForm,
		widget.NewSeparator(),
		widget.NewLabel("Results:"),
		resultsScroll,
		widget.NewSeparator(),
		statusSection,
	)
}
