package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/unklstewy/digiLogRT/internal/api"
)

func main() {
	fmt.Println("🧪 Testing API Cache Functionality")
	fmt.Println("==================================")

	// Test cache directory
	cacheDir := filepath.Join(os.TempDir(), "digiLogRT", "cache")
	fmt.Printf("Expected cache directory: %s\n", cacheDir)

	// Ensure directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Printf("Failed to create cache directory: %v\n", err)
		return
	}
	fmt.Printf("✓ Cache directory created/exists\n")

	// Test Brandmeister client
	fmt.Println("\n🔹 Testing Brandmeister Client")
	bmClient := api.NewBrandmeisterClient("")
	if data, err := bmClient.GetAllRepeaters(); err != nil {
		fmt.Printf("❌ Brandmeister test failed: %v\n", err)
	} else {
		fmt.Printf("✅ Brandmeister: loaded %d repeaters\n", len(data))
	}

	// Test TGIF client
	fmt.Println("\n🔹 Testing TGIF Client")
	tgifClient := api.NewTGIFClient()
	if data, err := tgifClient.GetAllTalkgroups(); err != nil {
		fmt.Printf("❌ TGIF test failed: %v\n", err)
	} else {
		fmt.Printf("✅ TGIF: loaded %d talkgroups\n", len(data))
	}

	// Test hearham client
	fmt.Println("\n🔹 Testing hearham Client")
	hearhamClient := api.NewHearhamClient()
	if data, err := hearhamClient.GetAllRepeaters(); err != nil {
		fmt.Printf("❌ hearham test failed: %v\n", err)
	} else {
		fmt.Printf("✅ hearham: loaded %d repeaters\n", len(data))
	}

	// Check what cache files were created
	fmt.Println("\n📁 Cache Files Created:")
	if files, err := os.ReadDir(cacheDir); err != nil {
		fmt.Printf("Failed to read cache directory: %v\n", err)
	} else {
		for _, file := range files {
			if !file.IsDir() {
				fullPath := filepath.Join(cacheDir, file.Name())
				if info, err := os.Stat(fullPath); err == nil {
					fmt.Printf("  %s (%d bytes)\n", file.Name(), info.Size())
				}
			}
		}
		if len(files) == 0 {
			fmt.Printf("  (no cache files found)\n")
		}
	}
}
