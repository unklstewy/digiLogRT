package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/unklstewy/digiLog/internal/api"
	"github.com/unklstewy/digiLog/internal/database"
)

func main() {
	dbFile := flag.String("db", "fast_sync.db", "Database file to sync to")
	flag.Bool("verbose", false, "Enable verbose output (for compatibility)")
	flag.Parse()

	fmt.Printf("🚀 FAST SYNC: Reading from pre-warmed caches\n")
	start := time.Now()

	// Initialize database only
	fmt.Printf("Initializing database: %s\n", *dbFile)
	dbStart := time.Now()
	db, err := database.NewDatabase(*dbFile)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	dbInitTime := time.Since(dbStart)
	fmt.Printf("✓ Database initialized: %s (took %v)\n", *dbFile, dbInitTime)

	// Read directly from cache files instead of initializing APIs
	cacheDir := filepath.Join(os.TempDir(), "digiLogRT", "cache")

	// Load Brandmeister data from cache
	brandmeisterFile := filepath.Join(cacheDir, "brandmeister_repeaters.json")
	bmStart := time.Now()
	bmData, err := readBrandmeisterCache(brandmeisterFile)
	if err != nil {
		log.Fatalf("Failed to read Brandmeister cache: %v (run warm_cache first)", err)
	}
	bmReadTime := time.Since(bmStart)

	// Load TGIF data from cache
	tgifFile := filepath.Join(cacheDir, "tgif_talkgroups.json")
	tgStart := time.Now()
	tgData, err := readTGIFCache(tgifFile)
	if err != nil {
		log.Fatalf("Failed to read TGIF cache: %v (run warm_cache first)", err)
	}
	tgReadTime := time.Since(tgStart)

	// Load hearham data from cache
	hearhamFile := filepath.Join(cacheDir, "hearham_repeaters.json")
	hhStart := time.Now()
	hhData, err := readHearhamCache(hearhamFile)
	if err != nil {
		log.Fatalf("Failed to read hearham cache: %v (run warm_cache first)", err)
	}
	hhReadTime := time.Since(hhStart)

	totalReadTime := bmReadTime + tgReadTime + hhReadTime
	fmt.Printf("✓ All cache files loaded in %v\n", totalReadTime)

	// Sync to database
	fmt.Printf("\n🚀 SYNCING FROM CACHED DATA\n")

	// Sync Brandmeister - use the same method names as the original sync
	fmt.Printf("Syncing %d Brandmeister repeaters...\n", len(bmData))
	bmSyncStart := time.Now()
	if err := db.SyncBrandmeisterData(bmData); err != nil {
		log.Fatalf("Failed to sync Brandmeister data: %v", err)
	}
	bmSyncTime := time.Since(bmSyncStart)

	// Sync TGIF
	fmt.Printf("Syncing %d TGIF talkgroups...\n", len(tgData))
	tgSyncStart := time.Now()
	if err := db.SyncTGIFData(tgData); err != nil {
		log.Fatalf("Failed to sync TGIF data: %v", err)
	}
	tgSyncTime := time.Since(tgSyncStart)

	// Sync hearham
	fmt.Printf("Syncing %d hearham repeaters...\n", len(hhData))
	hhSyncStart := time.Now()
	if err := db.SyncHearhamData(hhData); err != nil {
		log.Fatalf("Failed to sync hearham data: %v", err)
	}
	hhSyncTime := time.Since(hhSyncStart)

	totalSyncTime := bmSyncTime + tgSyncTime + hhSyncTime
	totalTime := time.Since(start)
	totalRecords := len(bmData) + len(tgData) + len(hhData)

	// Performance metrics
	fmt.Printf("\n======================================================================\n")
	fmt.Printf("FAST SYNC PERFORMANCE ANALYSIS\n")
	fmt.Printf("======================================================================\n")
	fmt.Printf("Database init:     %v (%.1f%%)\n", dbInitTime, float64(dbInitTime)/float64(totalTime)*100)
	fmt.Printf("Cache file reads:  %v (%.1f%%)\n", totalReadTime, float64(totalReadTime)/float64(totalTime)*100)
	fmt.Printf("Database sync:     %v (%.1f%%)\n", totalSyncTime, float64(totalSyncTime)/float64(totalTime)*100)
	fmt.Printf("Total time:        %v\n", totalTime)
	fmt.Printf("Total records:     %d\n", totalRecords)
	fmt.Printf("Overall rate:      %.0f records/second\n", float64(totalRecords)/totalTime.Seconds())
	fmt.Printf("Pure DB rate:      %.0f records/second\n", float64(totalRecords)/totalSyncTime.Seconds())

	// Performance comparison
	fmt.Printf("\n🚀 PERFORMANCE COMPARISON:\n")
	fmt.Printf("  Fast sync total:    %v\n", totalTime)
	fmt.Printf("  Regular sync est:   ~9.6s (from previous runs)\n")
	if totalTime.Seconds() < 9.6 {
		improvement := 9.6 / totalTime.Seconds()
		fmt.Printf("  Speed improvement:  %.1fx faster! 🚀\n", improvement)
	}

	// Show database statistics
	statsStart := time.Now()
	stats, err := db.GetRepeaterStats()
	statsTime := time.Since(statsStart)

	if err != nil {
		log.Printf("Failed to get final stats: %v", err)
	} else {
		fmt.Printf("\nFinal Database Statistics (query took %v):\n", statsTime)
		fmt.Printf("  Total repeaters: %d\n", stats["total_repeaters"])
		fmt.Printf("  Online repeaters: %d\n", stats["online_repeaters"])
		if bySource, ok := stats["by_source"].(map[string]int); ok {
			for source, count := range bySource {
				fmt.Printf("  %s: %d repeaters\n", source, count)
			}
		}
	}

	fmt.Printf("\n✅ BLAZING FAST SYNC COMPLETE! Database ready: %s\n", *dbFile)
}

func readBrandmeisterCache(filename string) ([]api.BrandmeisterRepeater, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var repeaters []api.BrandmeisterRepeater
	err = json.Unmarshal(data, &repeaters)
	return repeaters, err
}

func readTGIFCache(filename string) ([]api.TGIFTalkgroup, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var talkgroups []api.TGIFTalkgroup
	err = json.Unmarshal(data, &talkgroups)
	return talkgroups, err
}

func readHearhamCache(filename string) ([]api.HearhamRepeater, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var repeaters []api.HearhamRepeater
	err = json.Unmarshal(data, &repeaters)
	return repeaters, err
}
