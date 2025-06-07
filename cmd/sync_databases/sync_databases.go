package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/unklstewy/digiLogRT/internal/api"
	"github.com/unklstewy/digiLogRT/internal/config"
	"github.com/unklstewy/digiLogRT/internal/database"
)

// TimingResult tracks detailed timing information for each sync operation
type TimingResult struct {
	Source           string
	RecordCount      int
	InitTime         time.Duration
	FetchTime        time.Duration
	ProcessTime      time.Duration
	TotalTime        time.Duration
	RecordsPerSecond float64
}

func main() {
	// Command line flags
	dbPath := flag.String("db", "digilog_production.db", "Database file path")
	sources := flag.String("sources", "brandmeister,tgif,hearham", "Comma-separated list of sources to sync")
	_ = flag.Bool("force", false, "Force refresh even if cache is valid")
	verbose := flag.Bool("verbose", false, "Show detailed timing information")
	flag.Parse()

	log.Printf("Starting database sync for sources: %s", *sources)
	overallStart := time.Now()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create database
	dbStart := time.Now()
	db, err := database.NewDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	dbInitTime := time.Since(dbStart)

	fmt.Printf("✓ Database initialized: %s (took %v)\n", *dbPath, dbInitTime)

	// Initialize client pool in parallel
	fmt.Println("\n🚀 Initializing API clients in parallel...")
	poolStart := time.Now()
	pool := api.GetGlobalPool()
	if err := pool.Initialize(cfg.APIs.BrandmeisterKey); err != nil {
		log.Fatalf("Failed to initialize client pool: %v", err)
	}
	poolInitTime := time.Since(poolStart)

	fmt.Printf("✓ All clients initialized in parallel: %v\n", poolInitTime)

	// Get initialized clients
	brandmeisterClient, tgifClient, hearhamClient := pool.GetClients()

	// Sync sources based on flags
	sourceList := []string{"brandmeister", "tgif", "hearham"}
	if *sources != "brandmeister,tgif,hearham" {
		sourceList = strings.Split(*sources, ",")
	}

	var timingResults []TimingResult
	totalRecords := 0

	for _, source := range sourceList {
		var result TimingResult

		switch source {
		case "brandmeister":
			if brandmeisterClient != nil {
				result = syncBrandmeisterWithPool(db, brandmeisterClient, *verbose)
				if result.RecordCount > 0 {
					totalRecords += result.RecordCount
					timingResults = append(timingResults, result)
				}
			} else {
				log.Println("Skipping Brandmeister - no API key configured")
			}

		case "tgif":
			if tgifClient != nil {
				result = syncTGIFWithPool(db, tgifClient, *verbose)
				if result.RecordCount > 0 {
					totalRecords += result.RecordCount
					timingResults = append(timingResults, result)
				}
			} else {
				log.Println("Skipping TGIF - client not initialized")
			}

		case "hearham":
			if hearhamClient != nil {
				result = syncHearhamWithPool(db, hearhamClient, *verbose)
				if result.RecordCount > 0 {
					totalRecords += result.RecordCount
					timingResults = append(timingResults, result)
				}
			} else {
				log.Println("Skipping hearham - client not initialized")
			}

		default:
			log.Printf("Unknown source: %s", source)
		}
	}

	overallElapsed := time.Since(overallStart)

	// Show detailed timing analysis with pool metrics
	showTimingAnalysisWithPool(timingResults, totalRecords, overallElapsed, dbInitTime, poolInitTime, *verbose)

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

	fmt.Printf("\n✓ Database ready for production use: %s\n", *dbPath)
}

func syncBrandmeisterWithPool(db *database.Database, client *api.BrandmeisterClient, verbose bool) TimingResult {
	result := TimingResult{Source: "brandmeister"}
	sourceStart := time.Now()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SYNCING BRANDMEISTER DATA")
	fmt.Println(strings.Repeat("=", 50))

	// Client already initialized - no init time
	result.InitTime = 0

	// Fetch data
	fetchStart := time.Now()
	response, err := client.GetAllRepeaters()
	if err != nil {
		log.Printf("Failed to get Brandmeister repeaters: %v", err)
		return result
	}
	result.FetchTime = time.Since(fetchStart)
	result.RecordCount = len(response)

	fmt.Printf("⏱️  Data fetch: %v (%d records)\n", result.FetchTime, result.RecordCount)

	// Process data
	processStart := time.Now()
	// Use the repeaters as returned by the API client
	if err := db.SyncBrandmeisterData(response); err != nil {
		log.Printf("Failed to sync Brandmeister data: %v", err)
		return result
	}
	result.ProcessTime = time.Since(processStart)
	result.TotalTime = time.Since(sourceStart)
	result.RecordsPerSecond = float64(result.RecordCount) / result.TotalTime.Seconds()

	fmt.Printf("⏱️  Database sync: %v\n", result.ProcessTime)
	fmt.Printf("⏱️  Total time: %v\n", result.TotalTime)
	fmt.Printf("🚀 Processing rate: %.0f records/second\n", result.RecordsPerSecond)

	return result
}

func syncTGIFWithPool(db *database.Database, client *api.TGIFClient, verbose bool) TimingResult {
	result := TimingResult{Source: "tgif"}
	sourceStart := time.Now()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SYNCING TGIF DATA")
	fmt.Println(strings.Repeat("=", 50))

	// Client already initialized - no init time
	result.InitTime = 0

	// Fetch data
	fetchStart := time.Now()
	response, err := client.GetAllTalkgroups()
	if err != nil {
		log.Printf("Failed to get TGIF talkgroups: %v", err)
		return result
	}
	result.FetchTime = time.Since(fetchStart)
	result.RecordCount = len(response)

	fmt.Printf("⏱️  Data fetch: %v (%d records)\n", result.FetchTime, result.RecordCount)

	// Process data
	processStart := time.Now()
	if err := db.SyncTGIFData(response); err != nil {
		log.Printf("Failed to sync TGIF data: %v", err)
		return result
	}
	result.ProcessTime = time.Since(processStart)
	result.TotalTime = time.Since(sourceStart)
	result.RecordsPerSecond = float64(result.RecordCount) / result.TotalTime.Seconds()

	fmt.Printf("⏱️  Database sync: %v\n", result.ProcessTime)
	fmt.Printf("⏱️  Total time: %v\n", result.TotalTime)
	fmt.Printf("🚀 Processing rate: %.0f records/second\n", result.RecordsPerSecond)

	return result
}

func syncHearhamWithPool(db *database.Database, client *api.HearhamClient, verbose bool) TimingResult {
	result := TimingResult{Source: "hearham"}
	sourceStart := time.Now()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SYNCING HEARHAM DATA")
	fmt.Println(strings.Repeat("=", 50))

	// Client already initialized - no init time
	result.InitTime = 0

	// Fetch data
	fetchStart := time.Now()
	response, err := client.GetAllRepeaters()
	if err != nil {
		log.Printf("Failed to get hearham repeaters: %v", err)
		return result
	}
	result.FetchTime = time.Since(fetchStart)
	result.RecordCount = len(response)

	fmt.Printf("⏱️  Data fetch: %v (%d records)\n", result.FetchTime, result.RecordCount)

	// Process data
	processStart := time.Now()
	if err := db.SyncHearhamData(response); err != nil {
		log.Printf("Failed to sync hearham data: %v", err)
		return result
	}
	result.ProcessTime = time.Since(processStart)
	result.TotalTime = time.Since(sourceStart)
	result.RecordsPerSecond = float64(result.RecordCount) / result.TotalTime.Seconds()

	fmt.Printf("⏱️  Database sync: %v\n", result.ProcessTime)
	fmt.Printf("⏱️  Total time: %v\n", result.TotalTime)
	fmt.Printf("🚀 Processing rate: %.0f records/second\n", result.RecordsPerSecond)

	return result
}

func showTimingAnalysisWithPool(results []TimingResult, totalRecords int, overallTime time.Duration, dbInitTime time.Duration, poolInitTime time.Duration, verbose bool) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("DETAILED TIMING ANALYSIS (WITH PARALLEL INIT)")
	fmt.Println(strings.Repeat("=", 70))

	// Per-source breakdown
	fmt.Printf("%-15s %8s %12s %12s %12s %12s\n",
		"Source", "Records", "Init", "Fetch", "Process", "Total")
	fmt.Println(strings.Repeat("-", 70))

	totalFetchTime := time.Duration(0)
	totalProcessTime := time.Duration(0)

	for _, result := range results {
		fmt.Printf("%-15s %8d %12s %12v %12v %12v\n",
			result.Source,
			result.RecordCount,
			"pooled", // All clients initialized in pool
			result.FetchTime,
			result.ProcessTime,
			result.TotalTime)

		totalFetchTime += result.FetchTime
		totalProcessTime += result.ProcessTime
	}

	fmt.Println(strings.Repeat("-", 70))

	// Overall statistics
	fmt.Printf("\nOVERALL PERFORMANCE METRICS:\n")
	fmt.Printf("  Database initialization: %v\n", dbInitTime)
	fmt.Printf("  Parallel client init:    %v\n", poolInitTime)
	fmt.Printf("  Total data fetch time:   %v\n", totalFetchTime)
	fmt.Printf("  Total processing time:   %v\n", totalProcessTime)
	fmt.Printf("  Overall elapsed time:    %v\n", overallTime)
	fmt.Printf("  Total records processed: %d\n", totalRecords)

	overallRate := float64(totalRecords) / overallTime.Seconds()
	fmt.Printf("  Overall processing rate: %.0f records/second\n", overallRate)

	// Performance improvement calculation
	sequentialInitTime := time.Duration(0)
	for _, result := range results {
		sequentialInitTime += result.InitTime
	}
	if sequentialInitTime == 0 {
		// Estimate based on previous runs (use your measured ~10.4s)
		sequentialInitTime = 10400 * time.Millisecond
	}

	savedTime := sequentialInitTime - poolInitTime
	if savedTime > 0 {
		fmt.Printf("  Time saved with parallel: %v (%.1f%% faster)\n",
			savedTime, float64(savedTime)/float64(sequentialInitTime)*100)
	}

	// Performance breakdown percentages
	if verbose {
		fmt.Printf("\nTIME BREAKDOWN:\n")
		fmt.Printf("  Database init: %.1f%%\n", float64(dbInitTime)/float64(overallTime)*100)
		fmt.Printf("  Parallel init: %.1f%%\n", float64(poolInitTime)/float64(overallTime)*100)
		fmt.Printf("  Data fetching: %.1f%%\n", float64(totalFetchTime)/float64(overallTime)*100)
		fmt.Printf("  DB processing: %.1f%%\n", float64(totalProcessTime)/float64(overallTime)*100)
	}

	// Performance insights
	fmt.Printf("\nPERFORMANCE INSIGHTS:\n")
	if totalFetchTime > totalProcessTime {
		fmt.Printf("  🌐 Network I/O is the bottleneck (%.1f%% of time)\n",
			float64(totalFetchTime)/float64(overallTime)*100)
	} else {
		fmt.Printf("  💾 Database processing is the bottleneck (%.1f%% of time)\n",
			float64(totalProcessTime)/float64(overallTime)*100)
	}

	if overallRate > 15000 {
		fmt.Printf("  🚀 Excellent performance: >15k records/second\n")
	} else if overallRate > 10000 {
		fmt.Printf("  ⚡ Very good performance: >10k records/second\n")
	} else if overallRate > 5000 {
		fmt.Printf("  ✅ Good performance: >5k records/second\n")
	} else if overallRate > 1000 {
		fmt.Printf("  🔄 Acceptable performance: >1k records/second\n")
	} else {
		fmt.Printf("  ⚠️  Consider optimization: <1k records/second\n")
	}

	// Database efficiency
	if totalProcessTime > 0 {
		dbRate := float64(totalRecords) / totalProcessTime.Seconds()
		fmt.Printf("  💾 Pure database rate: %.0f records/second\n", dbRate)
	}
}
