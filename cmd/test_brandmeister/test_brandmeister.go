package main

import (
	"fmt"
	"log"

	"github.com/unklstewy/digiLog/internal/api"
	"github.com/unklstewy/digiLog/internal/config"
)

func main() {
	log.Println("Testing Brandmeister.network API with intelligent caching...")

	// Load configuration to get API key
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Check if API key is configured
	if cfg.APIs.BrandmeisterKey == "" {
		log.Fatalf("Brandmeister API key not configured in config.yaml")
	}

	// Create Brandmeister client with API key from configuration
	client := api.NewBrandmeisterClient(cfg.APIs.BrandmeisterKey)

	// Test initialization (will check cache age and refresh if needed)
	fmt.Println("Initializing Brandmeister client...")
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

	// Get all repeaters
	repeaters, err := client.GetAllRepeaters()
	if err != nil {
		log.Fatalf("Failed to get repeaters: %v", err)
	}

	fmt.Printf("Total repeaters: %d\n", repeaters.Count)

	// Show first 3 repeaters as examples
	maxShow := 3
	if len(repeaters.Repeaters) < maxShow {
		maxShow = len(repeaters.Repeaters)
	}

	fmt.Printf("\nFirst %d repeaters:\n", maxShow)
	for i := 0; i < maxShow; i++ {
		r := repeaters.Repeaters[i]
		fmt.Printf("\nRepeater %d:\n", i+1)
		fmt.Printf("  ID: %d\n", r.ID)
		fmt.Printf("  Callsign: %s\n", r.Callsign)
		fmt.Printf("  Frequency: %s\n", r.GetFrequencyString())
		fmt.Printf("  Color Code: %d\n", r.ColorCode)
		fmt.Printf("  Location: %s\n", r.GetLocationString())
		lat, lon := r.GetCoordinates()
		fmt.Printf("  Coordinates: %s, %s\n", lat, lon)
		fmt.Printf("  Online: %t\n", r.IsOnline())
		if r.Hardware != "" {
			fmt.Printf("  Hardware: %s\n", r.Hardware)
		}
	}

	// Test repeater search
	fmt.Println("\n\nTesting repeater search (searching for 'Pennsylvania')...")
	searchResults, err := client.SearchRepeaters("Pennsylvania")
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d repeaters matching 'Pennsylvania'\n", searchResults.Count)

		// Show first 3 search results
		maxSearchShow := 3
		if len(searchResults.Repeaters) < maxSearchShow {
			maxSearchShow = len(searchResults.Repeaters)
		}

		for i := 0; i < maxSearchShow; i++ {
			r := searchResults.Repeaters[i]
			fmt.Printf("  %d: %s (%s) - %s\n", i+1, r.Callsign, r.GetFrequencyString(), r.GetLocationString())
		}
	}

	// Test specific repeater lookup
	fmt.Println("\nTesting specific repeater lookup...")

	// Show sample of available IDs
	fmt.Println("Sample of available repeater IDs:")
	for i := 0; i < 5 && i < len(repeaters.Repeaters); i++ {
		fmt.Printf("  Repeater ID: %d (%s)\n", repeaters.Repeaters[i].ID, repeaters.Repeaters[i].Callsign)
	}

	// Try looking up the first repeater
	if len(repeaters.Repeaters) > 0 {
		firstID := repeaters.Repeaters[0].ID
		fmt.Printf("\nTesting lookup of Repeater ID %d...\n", firstID)
		rep, err := client.GetRepeater(firstID)
		if err != nil {
			log.Printf("Repeater %d lookup failed: %v", firstID, err)
		} else {
			fmt.Printf("Repeater %d: %s\n", firstID, rep.Callsign)
			fmt.Printf("Location: %s\n", rep.GetLocationString())
			fmt.Printf("Frequency: %s\n", rep.GetFrequencyString())
		}
	}

	// Test edge case - non-existent repeater
	fmt.Println("\nTesting lookup of non-existent Repeater ID 999999...")
	if _, err := client.GetRepeater(999999); err != nil {
		fmt.Printf("✓ Correctly handled non-existent repeater: %v\n", err)
	} else {
		fmt.Println("⚠ Warning: Found unexpected repeater 999999")
	}

	// Add some dataset statistics
	fmt.Println("\nDataset Statistics:")

	// Count online vs offline repeaters
	onlineCount := 0
	for _, r := range repeaters.Repeaters {
		if r.IsOnline() {
			onlineCount++
		}
	}
	fmt.Printf("  Online repeaters: %d/%d (%.1f%%)\n",
		onlineCount, len(repeaters.Repeaters),
		float64(onlineCount)/float64(len(repeaters.Repeaters))*100)

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

	fmt.Println("\n✓ Brandmeister.network API test completed successfully!")
}
