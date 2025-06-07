package main

import (
	"fmt"
	"log"

	"github.com/unklstewy/digiLogRT/internal/api"
)

func main() {
	log.Println("Testing APRS.fi API...")

	// Create APRS client with your API key
	client := api.NewAPRSClient("126515.6ryMvtanTmJDG")

	// Test the connection
	fmt.Println("Testing API connection...")
	if err := client.TestConnection(); err != nil {
		log.Fatalf("API connection test failed: %v", err)
	}
	fmt.Println("✓ API connection successful!")

	// Get a specific station
	fmt.Println("\nTesting station lookup...")
	response, err := client.GetStation("OH7RDA")
	if err != nil {
		log.Fatalf("Station lookup failed: %v", err)
	}

	fmt.Printf("Command: %s\n", response.Command)
	fmt.Printf("Result: %s\n", response.Result)
	fmt.Printf("Found %d station(s)\n", response.Found)

	if len(response.Entries) > 0 {
		station := response.Entries[0]
		fmt.Printf("Station: %s\n", station.Name)
		fmt.Printf("Location: %.6f, %.6f\n", station.Lat, station.Lng)
		fmt.Printf("Last seen: %s\n", station.GetLastTimeString()) // Using helper method
		fmt.Printf("Time: %s\n", station.GetTimeString())          // Using helper method
		fmt.Printf("Comment: %s\n", station.Comment)
	}

	fmt.Println("\n✓ APRS API test completed successfully!")
}
