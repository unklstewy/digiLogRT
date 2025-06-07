package main

import (
	"fmt"
	"log"
	"os"

	"github.com/unklstewy/digiLogRT/internal/database"
)

func main() {
	log.Println("Testing SQLite3 database integration...")

	// Create temporary database for testing
	dbPath := "test_digilog.db"
	defer os.Remove(dbPath) // Clean up after test

	// Initialize database
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	fmt.Println("✓ Database created and schema initialized successfully!")

	// Test basic operations
	fmt.Println("\nTesting database operations...")

	// Test location insertion
	locationID, err := db.UpsertLocation("Los Angeles", "CA", "USA", 34.0522, -118.2437)
	if err != nil {
		log.Fatalf("Failed to insert location: %v", err)
	}
	fmt.Printf("✓ Location inserted with ID: %d\n", locationID)

	// Get source ID for Brandmeister
	sourceID, err := db.GetSourceID("brandmeister")
	if err != nil {
		log.Fatalf("Failed to get source ID: %v", err)
	}
	fmt.Printf("✓ Brandmeister source ID: %d\n", sourceID)

	// Test search functionality
	results, err := db.SearchRepeaters("Los Angeles", 10)
	if err != nil {
		log.Fatalf("Failed to search repeaters: %v", err)
	}
	fmt.Printf("✓ Search test completed, found %d results\n", len(results))

	// Get database statistics
	stats, err := db.GetRepeaterStats()
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}
	fmt.Printf("✓ Database stats: %+v\n", stats)

	fmt.Println("\n✓ Database test completed successfully!")
}
