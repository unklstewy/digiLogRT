package main

import (
	"fmt"
	"log"

	"github.com/unklstewy/digiLog/internal/api"
)

func main() {
	log.Println("Testing TGIF.network API with intelligent caching...")

	// Create TGIF client
	client := api.NewTGIFClient()

	// Test initialization (will check cache age and refresh if needed)
	fmt.Println("Initializing TGIF client...")
	if err := client.Initialize(); err != nil {
		log.Fatalf("Client initialization failed: %v", err)
	}
	fmt.Println("✓ Client initialized successfully!")

	// Show cache status
	status := client.GetCacheStatus()
	fmt.Printf("\nCache Status:\n")
	fmt.Printf("  Talkgroups: %d\n", status["count"])
	fmt.Printf("  Last Update: %v\n", status["last_update"])
	fmt.Printf("  Cache Age: %v\n", status["age"])
	fmt.Printf("  Cache Valid: %t\n", status["cache_valid"])
	fmt.Printf("  Needs Refresh: %t\n", status["needs_refresh"])

	// Get all talkgroups
	talkgroups, err := client.GetAllTalkgroups()
	if err != nil {
		log.Fatalf("Failed to get talkgroups: %v", err)
	}

	fmt.Printf("Total talkgroups: %d\n", talkgroups.Count)

	// Show first 5 talkgroups as examples
	maxShow := 5
	if len(talkgroups.Talkgroups) < maxShow {
		maxShow = len(talkgroups.Talkgroups)
	}

	fmt.Printf("\nFirst %d talkgroups:\n", maxShow)
	for i := 0; i < maxShow; i++ {
		tg := talkgroups.Talkgroups[i]
		fmt.Printf("\nTalkgroup %d:\n", i+1)
		fmt.Printf("  ID: %s\n", tg.ID)
		fmt.Printf("  Name: %s\n", tg.Name)
		if tg.Description != "" {
			if decoded, err := tg.GetDecodedDescription(); err == nil && decoded != "" {
				fmt.Printf("  Description: %s\n", decoded)
			} else {
				fmt.Printf("  Description: %s (encoded)\n", tg.Description)
			}
		}
	}

	// Test talkgroup search
	fmt.Println("\n\nTesting talkgroup search (searching for 'North America')...")
	searchResults, err := client.SearchTalkgroups("North America")
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d talkgroups matching 'North America'\n", searchResults.Count)

		// Show first 3 search results
		maxSearchShow := 3
		if len(searchResults.Talkgroups) < maxSearchShow {
			maxSearchShow = len(searchResults.Talkgroups)
		}

		for i := 0; i < maxSearchShow; i++ {
			tg := searchResults.Talkgroups[i]
			fmt.Printf("  %d: %s (TG %s)\n", i+1, tg.Name, tg.ID)
		}
	}

	// Test specific talkgroup lookup
	fmt.Println("\nTesting specific talkgroup lookup...")

	// First, let's see what IDs are in the first 10 talkgroups
	fmt.Println("Sample of available IDs:")
	for i := 0; i < 10 && i < len(talkgroups.Talkgroups); i++ {
		fmt.Printf("  TG ID: %s\n", talkgroups.Talkgroups[i].ID)
	}

	// Try looking up a known ID (like "101" from the first talkgroup)
	fmt.Println("\nTesting lookup of TG 101 (known to exist)...")
	tg101, err := client.GetTalkgroup(101)
	if err != nil {
		log.Printf("TG 101 lookup failed: %v", err)
	} else {
		fmt.Printf("TG 101: %s\n", tg101.Name)
		if decoded, err := tg101.GetDecodedDescription(); err == nil {
			fmt.Printf("Description: %s\n", decoded)
		}
	}

	// ...existing code...

	// Now try the original TG 91
	fmt.Println("\nTesting lookup of TG 91...")
	tg91, err := client.GetTalkgroup(91)
	if err != nil {
		fmt.Printf("TG 91 not found in database (expected) - %v\n", err)
	} else {
		fmt.Printf("TG 91: %s\n", tg91.Name)
		if decoded, err := tg91.GetDecodedDescription(); err == nil {
			fmt.Printf("Description: %s\n", decoded)
		}
	}

	// Test with a known good TG from our sample
	fmt.Println("\nTesting lookup of TG 110 (North America)...")
	tg110, err := client.GetTalkgroup(110)
	if err != nil {
		log.Printf("TG 110 lookup failed: %v", err)
	} else {
		fmt.Printf("TG 110: %s\n", tg110.Name)
		if decoded, err := tg110.GetDecodedDescription(); err == nil && decoded != "" {
			fmt.Printf("Description: %s\n", decoded)
		}
	}

	// Test edge case - non-existent high number
	fmt.Println("\nTesting lookup of non-existent TG 999999...")
	if _, err := client.GetTalkgroup(999999); err != nil {
		fmt.Printf("✓ Correctly handled non-existent TG: %v\n", err)
	} else {
		fmt.Println("⚠ Warning: Found unexpected talkgroup 999999")
	}
	// ...existing code...

	fmt.Printf("Total talkgroups: %d\n", talkgroups.Count)

	// Add some dataset statistics
	fmt.Println("\nDataset Statistics:")

	// Find ID range
	if len(talkgroups.Talkgroups) > 0 {
		firstID := talkgroups.Talkgroups[0].ID
		lastID := talkgroups.Talkgroups[len(talkgroups.Talkgroups)-1].ID
		fmt.Printf("  ID Range: %s to %s\n", firstID, lastID)
	}

	// Count talkgroups with descriptions
	describedCount := 0
	for _, tg := range talkgroups.Talkgroups {
		if tg.Description != "" {
			describedCount++
		}
	}
	fmt.Printf("  Talkgroups with descriptions: %d/%d (%.1f%%)\n",
		describedCount, len(talkgroups.Talkgroups),
		float64(describedCount)/float64(len(talkgroups.Talkgroups))*100)

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

	fmt.Println("\n✓ TGIF.network API test completed successfully!")
}
