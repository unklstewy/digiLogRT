package main

import (
	"fmt"
	"log"

	"github.com/unklstewy/digiLog/internal/api"
)

func main() {
	log.Println("Testing hearham.com API with intelligent caching...")

	// Create hearham client
	client := api.NewHearhamClient()

	// Test initialization (will check cache age and refresh if needed)
	fmt.Println("Initializing hearham client...")
	if err := client.Initialize(); err != nil {
		log.Fatalf("Client initialization failed: %v", err)
	}
	fmt.Println("✓ Client initialized successfully!")

	// Show cache status
	status := client.GetCacheStatus()
	fmt.Printf("\nCache Status:\n")
	fmt.Printf("  Repeaters: %d\n", status["count"])
	fmt.Printf("  Last Update: %v\n", status["last_update"])
	fmt.Printf("  Cache Age: %v\n", status["age"])
	fmt.Printf("  Cache Valid: %t\n", status["cache_valid"])
	fmt.Printf("  Needs Refresh: %t\n", status["needs_refresh"])

	// Get all data to show examples
	allData, err := client.GetAllData()
	if err != nil {
		log.Fatalf("Failed to get all data: %v", err)
	}

	fmt.Printf("Total repeaters in database: %d\n", allData.Count)

	// Show first 3 repeaters as examples
	maxShow := 3
	if len(allData.Repeaters) < maxShow {
		maxShow = len(allData.Repeaters)
	}

	fmt.Printf("\nFirst %d repeaters in database:\n", maxShow)
	for i := 0; i < maxShow; i++ {
		repeater := allData.Repeaters[i]
		fmt.Printf("\nRepeater %d:\n", i+1)
		fmt.Printf("  ID: %d\n", repeater.ID)
		fmt.Printf("  Callsign: %s\n", repeater.Callsign)
		fmt.Printf("  Frequency: %.5f MHz\n", repeater.GetFrequencyMHz())
		if repeater.Offset != 0 {
			fmt.Printf("  Input: %.5f MHz (offset: %+.3f MHz)\n", repeater.GetInputFrequencyMHz(), repeater.GetOffsetMHz())
		}
		tone := repeater.GetTone()
		if tone != "" {
			fmt.Printf("  Tone: %s\n", tone)
		}
		if repeater.Mode != "" {
			fmt.Printf("  Mode: %s\n", repeater.Mode)
		}
		fmt.Printf("  Location: %s\n", repeater.GetLocation())
		fmt.Printf("  Coordinates: %.6f, %.6f\n", repeater.Latitude, repeater.Longitude)
		fmt.Printf("  Operational: %t\n", repeater.IsOperational())
		if repeater.Description != "" {
			fmt.Printf("  Description: %s\n", repeater.Description)
		}
	}

	// Test state search with local filtering
	fmt.Println("\n\nTesting state search (Pennsylvania) with local filtering...")
	response, err := client.SearchByState("PA")
	if err != nil {
		log.Printf("State search failed: %v", err)
	} else {
		fmt.Printf("Found %d repeaters in Pennsylvania\n", response.Count)
	}

	// Test frequency search (2m band) with local filtering
	fmt.Println("\nTesting frequency search (2m band: 144-148 MHz) with local filtering...")
	freqResponse, err := client.SearchByFrequency(144.0, 148.0)
	if err != nil {
		log.Printf("Frequency search failed: %v", err)
	} else {
		fmt.Printf("Found %d repeaters in 2m band\n", freqResponse.Count)
	}

	// Test manual refresh capability
	fmt.Println("\nTesting manual refresh capability...")
	fmt.Println("Note: This will re-download the data to show change detection")
	if err := client.ForceRefresh(); err != nil {
		log.Printf("Force refresh failed: %v", err)
	} else {
		fmt.Println("✓ Manual refresh completed")

		// Show updated cache status
		newStatus := client.GetCacheStatus()
		fmt.Printf("Updated cache age: %v\n", newStatus["age"])
	}

	fmt.Println("\n✓ hearham.com API test completed successfully!")
}
