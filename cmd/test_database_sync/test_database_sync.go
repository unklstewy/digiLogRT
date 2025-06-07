package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/unklstewy/digiLogRT/internal/api"
	"github.com/unklstewy/digiLogRT/internal/config"
	"github.com/unklstewy/digiLogRT/internal/database"
)

func main() {
	log.Println("Testing full database sync with all APIs...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create database
	dbPath := "digilog_full.db"
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath) // Remove test database

	fmt.Println("✓ Database initialized successfully!")

	// Sync Brandmeister data
	if cfg.APIs.BrandmeisterKey != "" {
		fmt.Println("\nSyncing Brandmeister data...")
		client := api.NewBrandmeisterClient(cfg.APIs.BrandmeisterKey)
		if err := client.Initialize(); err != nil {
			log.Printf("Failed to initialize Brandmeister: %v", err)
		} else {
			response, err := client.GetAllRepeaters()
			if err != nil {
				log.Printf("Failed to get Brandmeister repeaters: %v", err)
			} else {
				if err := db.SyncBrandmeisterData(response); err != nil {
					log.Printf("Failed to sync Brandmeister data: %v", err)
				}
			}
		}
	}

	// Sync TGIF data
	fmt.Println("\nSyncing TGIF data...")
	tgifClient := api.NewTGIFClient()
	if err := tgifClient.Initialize(); err != nil {
		log.Printf("Failed to initialize TGIF: %v", err)
	} else {
		talkgroups, err := tgifClient.GetAllTalkgroups()
		if err != nil {
			log.Printf("Failed to get TGIF talkgroups: %v", err)
		} else {
			if err := db.SyncTGIFData(talkgroups); err != nil {
				log.Printf("Failed to sync TGIF data: %v", err)
			}
		}
	}

	// Sync hearham data
	fmt.Println("\nSyncing hearham data...")
	hearhamClient := api.NewHearhamClient()
	if err := hearhamClient.Initialize(); err != nil {
		log.Printf("Failed to initialize hearham: %v", err)
	} else {
		repeaters, err := hearhamClient.GetAllRepeaters()
		if err != nil {
			log.Printf("Failed to get hearham repeaters: %v", err)
		} else {
			if err := db.SyncHearhamData(repeaters); err != nil {
				log.Printf("Failed to sync hearham data: %v", err)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("FINAL DATABASE STATISTICS")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println(strings.Repeat("=", 50))

	stats, err := db.GetRepeaterStats()
	if err != nil {
		log.Printf("Failed to get final stats: %v", err)
	} else {
		fmt.Printf("Total repeaters in database: %d\n", stats["total_repeaters"])
		fmt.Printf("Online repeaters: %d\n", stats["online_repeaters"])
		fmt.Println("\nBy source:")
		if bySource, ok := stats["by_source"].(map[string]int); ok {
			for source, count := range bySource {
				fmt.Printf("  %s: %d repeaters\n", source, count)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("TESTING COMPLEX QUERIES")
	fmt.Println(strings.Repeat("=", 50))

	// Search test
	fmt.Println("\nSearching for 'California' repeaters...")
	results, err := db.SearchRepeaters("California", 5)
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d results:\n", len(results))
		for i, r := range results {
			fmt.Printf("  %d. %s (%s) - %s - %s\n",
				i+1, r.Callsign, r.GetFrequencyString(), r.Mode, r.GetLocationString())
		}
	}

	// Frequency search test
	fmt.Println("\nSearching for repeaters near 146.52 MHz (±1 MHz)...")
	freqResults, err := db.GetRepeatersByFrequency(146.52, 1.0, 5)
	if err != nil {
		log.Printf("Frequency search failed: %v", err)
	} else {
		fmt.Printf("Found %d repeaters near 146.52 MHz:\n", len(freqResults))
		for i, r := range freqResults {
			fmt.Printf("  %d. %s (%s) - %s - %s\n",
				i+1, r.Callsign, r.GetFrequencyString(), r.Mode, r.GetLocationString())
		}
	}

	fmt.Println("\n✓ Full database sync and testing completed successfully!")
}
