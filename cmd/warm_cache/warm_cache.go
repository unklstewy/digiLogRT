package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/unklstewy/digiLog/internal/api"
	"github.com/unklstewy/digiLog/internal/config"
)

func main() {
	maxAge := flag.String("max-age", "1h", "Maximum cache age before refresh (e.g., 1h, 30m, 24h)")
	sources := flag.String("sources", "brandmeister,tgif,hearham", "Comma-separated list of sources to warm")
	flag.Parse()

	fmt.Printf("🔥 Warming API caches (max age: %s)\n", *maxAge)
	start := time.Now()

	// Parse max age
	maxCacheAge, err := time.ParseDuration(*maxAge)
	if err != nil {
		log.Fatalf("Invalid max-age format: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Warm caches in parallel
	pool := api.GetGlobalPool()
	warmStart := time.Now()

	// Force cache refresh if older than maxAge
	if err := pool.WarmCaches(cfg.APIs.BrandmeisterKey, maxCacheAge); err != nil {
		log.Printf("Cache warming failed: %v", err)
	}

	warmTime := time.Since(warmStart)
	totalTime := time.Since(start)

	fmt.Printf("✓ Cache warming completed in %v\n", warmTime)
	fmt.Printf("✓ Total time: %v\n", totalTime)
	fmt.Println("🚀 Subsequent syncs should be near-instant!")
}
