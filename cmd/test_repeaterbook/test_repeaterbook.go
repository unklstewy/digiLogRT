package main

import (
	"fmt"
	"log"

	"github.com/unklstewy/digiLog/internal/api"
)

func main() {
	log.Println("Testing RepeaterBook.com API...")

	// Create RepeaterBook client (no API key needed for testing)
	client := api.NewRepeaterBookClient("")

	// Test the connection
	fmt.Println("Testing API connection...")
	if err := client.TestConnection(); err != nil {
		log.Fatalf("API connection test failed: %v", err)
	}
	fmt.Println("✓ API connection successful!")

	// Test state search
	fmt.Println("\nTesting state search (Pennsylvania)...")
	response, err := client.SearchByState("Pennsylvania")
	if err != nil {
		log.Fatalf("State search failed: %v", err)
	}

	fmt.Printf("Found %d repeaters in Pennsylvania\n", response.Count)

	// Show first 3 repeaters as examples
	maxShow := 3
	if len(response.Results) < maxShow {
		maxShow = len(response.Results)
	}

	for i := 0; i < maxShow; i++ {
		repeater := response.Results[i]
		fmt.Printf("\nRepeater %d:\n", i+1)
		fmt.Printf("  Callsign: %s\n", repeater.Callsign)
		fmt.Printf("  Frequency: %s MHz\n", repeater.Frequency)
		if repeater.InputFreq != "" {
			fmt.Printf("  Input: %s MHz\n", repeater.InputFreq)
		}
		if repeater.AccessTone != "" {
			fmt.Printf("  Tone: %s\n", repeater.AccessTone)
		}
		fmt.Printf("  Location: %s, %s\n", repeater.Nearest, repeater.County)

		lat, latErr := repeater.GetLatitude()
		lng, lngErr := repeater.GetLongitude()
		if latErr == nil && lngErr == nil {
			fmt.Printf("  Coordinates: %.6f, %.6f\n", lat, lng)
		}

		if repeater.IsDigital() {
			fmt.Printf("  Digital Modes: %v\n", repeater.GetDigitalModes())
		}

		if repeater.Notes != "" {
			fmt.Printf("  Notes: %s\n", repeater.Notes)
		}
	}

	fmt.Println("\n✓ RepeaterBook API test completed successfully!")
}
