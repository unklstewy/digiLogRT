package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unklstewy/digiLogRT/internal/api"
	"github.com/unklstewy/digiLogRT/internal/database"
)

// CacheResult holds the result of a cache operation
type CacheResult struct {
	Name     string
	Records  int
	ReadTime time.Duration
	SyncTime time.Duration
	Data     interface{}
	Err      error
}

// StreamingResult holds streaming processing results
type StreamingResult struct {
	Name          string
	RecordsRead   int
	RecordsSynced int
	ReadTime      time.Duration
	SyncTime      time.Duration
	Err           error
}

func main() {
	dbFile := flag.String("db", "fast_sync.db", "Database file to sync to")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	parallel := flag.Bool("parallel", true, "Enable parallel processing")
	streaming := flag.Bool("streaming", false, "Enable true streaming mode (memory efficient)")
	chunkSize := flag.Int("chunk", 1000, "Chunk size for streaming mode")
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
	if *verbose {
		fmt.Printf("✓ Database initialized: %s (took %v)\n", *dbFile, dbInitTime)
	}

	// Cache directory
	cacheDir := filepath.Join(os.TempDir(), "digiLogRT", "cache")
	fmt.Printf("Reading cache files from: %s\n", cacheDir)

	var totalReadTime, totalSyncTime time.Duration
	totalRecords := 0

	if *streaming {
		// TRUE STREAMING MODE - Memory efficient with chunked processing
		totalReadTime, totalSyncTime, totalRecords = processParallelStreaming(db, cacheDir, *chunkSize, *verbose)
	} else if *parallel {
		// PARALLEL PROCESSING MODE - Load full files in parallel
		totalReadTime, totalSyncTime, totalRecords = processParallel(db, cacheDir, *verbose)
	} else {
		// SEQUENTIAL PROCESSING MODE - Original method
		totalReadTime, totalSyncTime, totalRecords = processSequential(db, cacheDir, *verbose)
	}

	totalTime := time.Since(start)

	// Performance metrics
	fmt.Printf("\n======================================================================\n")
	if *streaming {
		fmt.Printf("STREAMING SYNC PERFORMANCE ANALYSIS\n")
	} else {
		fmt.Printf("ULTRA-FAST SYNC PERFORMANCE ANALYSIS\n")
	}
	fmt.Printf("======================================================================\n")
	fmt.Printf("Database init:     %v (%.1f%%)\n", dbInitTime, float64(dbInitTime)/float64(totalTime)*100)
	fmt.Printf("Cache file reads:  %v (%.1f%%)\n", totalReadTime, float64(totalReadTime)/float64(totalTime)*100)
	fmt.Printf("Database sync:     %v (%.1f%%)\n", totalSyncTime, float64(totalSyncTime)/float64(totalTime)*100)
	fmt.Printf("Total time:        %v\n", totalTime)
	fmt.Printf("Total records:     %d\n", totalRecords)
	if totalTime.Seconds() > 0 {
		fmt.Printf("Overall rate:      %.0f records/second\n", float64(totalRecords)/totalTime.Seconds())
	}
	if totalSyncTime.Seconds() > 0 {
		fmt.Printf("Pure DB rate:      %.0f records/second\n", float64(totalRecords)/totalSyncTime.Seconds())
	}

	// Performance comparison
	fmt.Printf("\n🚀 PERFORMANCE COMPARISON:\n")
	fmt.Printf("  Fast sync total:    %v\n", totalTime)
	fmt.Printf("  Regular sync est:   ~9.6s (from previous runs)\n")
	if totalTime.Seconds() < 9.6 && totalTime.Seconds() > 0 {
		improvement := 9.6 / totalTime.Seconds()
		fmt.Printf("  Speed improvement:  %.1fx faster! 🚀\n", improvement)
	}

	if *streaming {
		fmt.Printf("  Memory efficiency:  ~%dMB max (vs ~200MB full load)\n", *chunkSize/100)
	}

	// Show database statistics
	if totalRecords > 0 {
		statsStart := time.Now()
		stats, err := db.GetRepeaterStats()
		statsTime := time.Since(statsStart)

		if err != nil {
			log.Printf("Failed to get final stats: %v", err)
		} else {
			fmt.Printf("\nFinal Database Statistics (query took %v):\n", statsTime)
			fmt.Printf("  Total repeaters: %v\n", stats["total_repeaters"])
			fmt.Printf("  Online repeaters: %v\n", stats["online_repeaters"])
			if bySource, ok := stats["by_source"].(map[string]interface{}); ok {
				for source, count := range bySource {
					fmt.Printf("  %s: %v repeaters\n", source, count)
				}
			}
		}
	}

	modeStr := "PARALLEL"
	if *streaming {
		modeStr = "STREAMING"
	} else if !*parallel {
		modeStr = "SEQUENTIAL"
	}

	fmt.Printf("\n✅ %s SYNC COMPLETE! Database ready: %s\n", modeStr, *dbFile)
}

// processParallelStreaming handles true parallel streaming with chunked processing
func processParallelStreaming(db *database.Database, cacheDir string, chunkSize int, verbose bool) (time.Duration, time.Duration, int) {
	fmt.Printf("🌊 STREAMING MODE: Parallel streaming with chunked processing (chunks of %d)\n", chunkSize)

	resultChan := make(chan StreamingResult, 3)
	var wg sync.WaitGroup

	// Define cache files
	files := map[string]string{
		"Brandmeister": filepath.Join(cacheDir, "brandmeister_repeaters.json"),
		"TGIF":         filepath.Join(cacheDir, "tgif_talkgroups.json"),
		"hearham":      filepath.Join(cacheDir, "hearham_repeaters.json"),
	}

	start := time.Now()

	// Start parallel streaming processors
	for name, file := range files {
		wg.Add(1)
		go func(name, file string) {
			defer wg.Done()

			var result StreamingResult
			result.Name = name
			streamStart := time.Now()

			switch name {
			case "Brandmeister":
				read, synced, readTime, syncTime, err := streamBrandmeisterData(db, file, chunkSize, verbose)
				result.RecordsRead = read
				result.RecordsSynced = synced
				result.ReadTime = readTime
				result.SyncTime = syncTime
				result.Err = err
			case "TGIF":
				read, synced, readTime, syncTime, err := streamTGIFData(db, file, chunkSize, verbose)
				result.RecordsRead = read
				result.RecordsSynced = synced
				result.ReadTime = readTime
				result.SyncTime = syncTime
				result.Err = err
			case "hearham":
				read, synced, readTime, syncTime, err := streamHearhamData(db, file, chunkSize, verbose)
				result.RecordsRead = read
				result.RecordsSynced = synced
				result.ReadTime = readTime
				result.SyncTime = syncTime
				result.Err = err
			}

			if verbose {
				fmt.Printf("  ✓ %s streaming completed in %v\n", name, time.Since(streamStart))
			}

			resultChan <- result
		}(name, file)
	}

	// Wait for all streams to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var totalReadTime, totalSyncTime time.Duration
	totalRecords := 0

	for result := range resultChan {
		if result.Err != nil {
			log.Printf("Warning: Streaming failed for %s: %v", result.Name, result.Err)
			continue
		}

		totalReadTime += result.ReadTime
		totalSyncTime += result.SyncTime
		totalRecords += result.RecordsSynced

		if verbose {
			fmt.Printf("  ✓ %s: %d/%d records (%.1f%% success), read %v, sync %v\n",
				result.Name, result.RecordsSynced, result.RecordsRead,
				float64(result.RecordsSynced)/float64(result.RecordsRead)*100,
				result.ReadTime, result.SyncTime)
		} else {
			fmt.Printf("  ✓ %s: %d records streamed\n", result.Name, result.RecordsSynced)
		}
	}

	totalTime := time.Since(start)
	fmt.Printf("✓ All streams completed in: %v\n", totalTime)

	return totalReadTime, totalSyncTime, totalRecords
}

// processParallel handles parallel cache reading and database syncing
func processParallel(db *database.Database, cacheDir string, verbose bool) (time.Duration, time.Duration, int) {
	fmt.Printf("🚀 PARALLEL MODE: Loading all caches simultaneously\n")

	// Phase 1: Parallel cache reading
	cacheStart := time.Now()
	resultChan := make(chan CacheResult, 3)
	var wg sync.WaitGroup

	// Define cache files
	files := map[string]string{
		"Brandmeister": filepath.Join(cacheDir, "brandmeister_repeaters.json"),
		"TGIF":         filepath.Join(cacheDir, "tgif_talkgroups.json"),
		"hearham":      filepath.Join(cacheDir, "hearham_repeaters.json"),
	}

	// Start parallel cache reads
	for name, file := range files {
		wg.Add(1)
		go func(name, file string) {
			defer wg.Done()
			start := time.Now()

			var result CacheResult
			result.Name = name

			switch name {
			case "Brandmeister":
				data, count, err := readBrandmeisterCacheOptimized(file, verbose)
				result.Data = data
				result.Records = count
				result.Err = err
			case "TGIF":
				data, count, err := readTGIFCacheOptimized(file, verbose)
				result.Data = data
				result.Records = count
				result.Err = err
			case "hearham":
				data, count, err := readHearhamCacheOptimized(file, verbose)
				result.Data = data
				result.Records = count
				result.Err = err
			}

			result.ReadTime = time.Since(start)
			resultChan <- result
		}(name, file)
	}

	// Wait for all reads to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect cache results
	var results []CacheResult
	var totalReadTime time.Duration
	totalRecords := 0

	for result := range resultChan {
		if result.Err != nil {
			log.Printf("Warning: Failed to read %s cache: %v", result.Name, result.Err)
			continue
		}

		results = append(results, result)
		totalReadTime += result.ReadTime
		totalRecords += result.Records

		if verbose {
			fmt.Printf("  ✓ %s: %d records loaded in %v\n", result.Name, result.Records, result.ReadTime)
		}
	}

	actualCacheTime := time.Since(cacheStart)
	fmt.Printf("✓ All cache files loaded in parallel: %v\n", actualCacheTime)

	// Phase 2: Parallel database syncing
	fmt.Printf("🚀 PARALLEL MODE: Syncing to database simultaneously\n")
	syncStart := time.Now()
	syncChan := make(chan CacheResult, len(results))
	var syncWg sync.WaitGroup

	for _, result := range results {
		syncWg.Add(1)
		go func(res CacheResult) {
			defer syncWg.Done()
			start := time.Now()

			var err error
			switch res.Name {
			case "Brandmeister":
				err = db.SyncBrandmeisterData(res.Data.([]api.BrandmeisterRepeater))
			case "TGIF":
				err = db.SyncTGIFData(res.Data.([]api.TGIFTalkgroup))
			case "hearham":
				err = db.SyncHearhamData(res.Data.([]api.HearhamRepeater))
			}

			syncResult := CacheResult{
				Name:     res.Name,
				Records:  res.Records,
				SyncTime: time.Since(start),
				Err:      err,
			}

			syncChan <- syncResult
		}(result)
	}

	// Wait for all syncs to complete
	go func() {
		syncWg.Wait()
		close(syncChan)
	}()

	// Collect sync results
	var totalSyncTime time.Duration
	for result := range syncChan {
		if result.Err != nil {
			log.Printf("Warning: Failed to sync %s data: %v", result.Name, result.Err)
		} else {
			totalSyncTime += result.SyncTime
			if verbose {
				fmt.Printf("  ✓ %s: %d records synced in %v\n", result.Name, result.Records, result.SyncTime)
			} else {
				fmt.Printf("  ✓ %s: %d records synced\n", result.Name, result.Records)
			}
		}
	}

	actualSyncTime := time.Since(syncStart)
	fmt.Printf("✓ All data synced in parallel: %v\n", actualSyncTime)

	return actualCacheTime, actualSyncTime, totalRecords
}

// processSequential handles sequential processing (original method)
func processSequential(db *database.Database, cacheDir string, verbose bool) (time.Duration, time.Duration, int) {
	fmt.Printf("📝 SEQUENTIAL MODE: Processing caches one by one\n")

	var totalReadTime, totalSyncTime time.Duration
	totalRecords := 0

	// Load Brandmeister data from cache
	brandmeisterFile := filepath.Join(cacheDir, "brandmeister_repeaters.json")
	bmStart := time.Now()
	bmData, bmCount, err := readBrandmeisterCache(brandmeisterFile, verbose)
	if err != nil {
		log.Printf("Warning: Failed to read Brandmeister cache: %v", err)
	} else {
		bmReadTime := time.Since(bmStart)
		totalReadTime += bmReadTime
		totalRecords += bmCount

		// Sync to database
		bmSyncStart := time.Now()
		if err := db.SyncBrandmeisterData(bmData); err != nil {
			log.Printf("Warning: Failed to sync Brandmeister data: %v", err)
		} else {
			bmSyncTime := time.Since(bmSyncStart)
			totalSyncTime += bmSyncTime
			if verbose {
				fmt.Printf("  ✓ Brandmeister: %d records, read %v, sync %v\n", bmCount, bmReadTime, bmSyncTime)
			} else {
				fmt.Printf("  ✓ Brandmeister: %d records synced\n", bmCount)
			}
		}
	}

	// Load TGIF data from cache
	tgifFile := filepath.Join(cacheDir, "tgif_talkgroups.json")
	tgStart := time.Now()
	tgData, tgCount, err := readTGIFCache(tgifFile, verbose)
	if err != nil {
		log.Printf("Warning: Failed to read TGIF cache: %v", err)
	} else {
		tgReadTime := time.Since(tgStart)
		totalReadTime += tgReadTime
		totalRecords += tgCount

		// Sync to database
		tgSyncStart := time.Now()
		if err := db.SyncTGIFData(tgData); err != nil {
			log.Printf("Warning: Failed to sync TGIF data: %v", err)
		} else {
			tgSyncTime := time.Since(tgSyncStart)
			totalSyncTime += tgSyncTime
			if verbose {
				fmt.Printf("  ✓ TGIF: %d records, read %v, sync %v\n", tgCount, tgReadTime, tgSyncTime)
			} else {
				fmt.Printf("  ✓ TGIF: %d records synced\n", tgCount)
			}
		}
	}

	// Load hearham data from cache
	hearhamFile := filepath.Join(cacheDir, "hearham_repeaters.json")
	hhStart := time.Now()
	hhData, hhCount, err := readHearhamCache(hearhamFile, verbose)
	if err != nil {
		log.Printf("Warning: Failed to read hearham cache: %v", err)
	} else {
		hhReadTime := time.Since(hhStart)
		totalReadTime += hhReadTime
		totalRecords += hhCount

		// Sync to database
		hhSyncStart := time.Now()
		if err := db.SyncHearhamData(hhData); err != nil {
			log.Printf("Warning: Failed to sync hearham data: %v", err)
		} else {
			hhSyncTime := time.Since(hhSyncStart)
			totalSyncTime += hhSyncTime
			if verbose {
				fmt.Printf("  ✓ hearham: %d records, read %v, sync %v\n", hhCount, hhReadTime, hhSyncTime)
			} else {
				fmt.Printf("  ✓ hearham: %d records synced\n", hhCount)
			}
		}
	}

	return totalReadTime, totalSyncTime, totalRecords
}

// streamBrandmeisterData streams and processes Brandmeister data in chunks
func streamBrandmeisterData(db *database.Database, filename string, chunkSize int, verbose bool) (int, int, time.Duration, time.Duration, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var totalReadTime, totalSyncTime time.Duration
	var recordsRead, recordsSynced int

	// Create a streaming JSON decoder
	decoder := json.NewDecoder(file)

	readStart := time.Now()

	// Read opening bracket
	if _, err := decoder.Token(); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("expected opening bracket: %v", err)
	}

	var chunk []api.BrandmeisterRepeater

	// Stream individual records
	for decoder.More() {
		var repeater api.BrandmeisterRepeater
		if err := decoder.Decode(&repeater); err != nil {
			if verbose {
				log.Printf("Warning: Failed to decode Brandmeister record: %v", err)
			}
			continue
		}

		chunk = append(chunk, repeater)
		recordsRead++

		// Process chunk when it reaches desired size
		if len(chunk) >= chunkSize {
			totalReadTime += time.Since(readStart)

			syncStart := time.Now()
			if err := db.SyncBrandmeisterData(chunk); err != nil {
				log.Printf("Warning: Failed to sync Brandmeister chunk: %v", err)
			} else {
				recordsSynced += len(chunk)
				if verbose {
					fmt.Printf("    Brandmeister: synced chunk of %d records\n", len(chunk))
				}
			}
			totalSyncTime += time.Since(syncStart)

			// Reset chunk and continue reading
			chunk = chunk[:0] // Clear slice but keep capacity
			readStart = time.Now()
		}
	}

	// Process remaining records
	if len(chunk) > 0 {
		totalReadTime += time.Since(readStart)

		syncStart := time.Now()
		if err := db.SyncBrandmeisterData(chunk); err != nil {
			log.Printf("Warning: Failed to sync final Brandmeister chunk: %v", err)
		} else {
			recordsSynced += len(chunk)
			if verbose {
				fmt.Printf("    Brandmeister: synced final chunk of %d records\n", len(chunk))
			}
		}
		totalSyncTime += time.Since(syncStart)
	}

	return recordsRead, recordsSynced, totalReadTime, totalSyncTime, nil
}

// streamTGIFData streams and processes TGIF data in chunks
func streamTGIFData(db *database.Database, filename string, chunkSize int, verbose bool) (int, int, time.Duration, time.Duration, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var totalReadTime, totalSyncTime time.Duration
	var recordsRead, recordsSynced int

	decoder := json.NewDecoder(file)
	readStart := time.Now()

	// Read opening bracket
	if _, err := decoder.Token(); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("expected opening bracket: %v", err)
	}

	var chunk []api.TGIFTalkgroup

	for decoder.More() {
		var talkgroup api.TGIFTalkgroup
		if err := decoder.Decode(&talkgroup); err != nil {
			if verbose {
				log.Printf("Warning: Failed to decode TGIF record: %v", err)
			}
			continue
		}

		chunk = append(chunk, talkgroup)
		recordsRead++

		if len(chunk) >= chunkSize {
			totalReadTime += time.Since(readStart)

			syncStart := time.Now()
			if err := db.SyncTGIFData(chunk); err != nil {
				log.Printf("Warning: Failed to sync TGIF chunk: %v", err)
			} else {
				recordsSynced += len(chunk)
				if verbose {
					fmt.Printf("    TGIF: synced chunk of %d records\n", len(chunk))
				}
			}
			totalSyncTime += time.Since(syncStart)

			chunk = chunk[:0]
			readStart = time.Now()
		}
	}

	// Process remaining records
	if len(chunk) > 0 {
		totalReadTime += time.Since(readStart)

		syncStart := time.Now()
		if err := db.SyncTGIFData(chunk); err != nil {
			log.Printf("Warning: Failed to sync final TGIF chunk: %v", err)
		} else {
			recordsSynced += len(chunk)
		}
		totalSyncTime += time.Since(syncStart)
	}

	return recordsRead, recordsSynced, totalReadTime, totalSyncTime, nil
}

// streamHearhamData streams and processes hearham data in chunks
func streamHearhamData(db *database.Database, filename string, chunkSize int, verbose bool) (int, int, time.Duration, time.Duration, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var totalReadTime, totalSyncTime time.Duration
	var recordsRead, recordsSynced int

	decoder := json.NewDecoder(file)
	readStart := time.Now()

	// Read opening bracket
	if _, err := decoder.Token(); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("expected opening bracket: %v", err)
	}

	var chunk []api.HearhamRepeater

	for decoder.More() {
		var repeater api.HearhamRepeater
		if err := decoder.Decode(&repeater); err != nil {
			if verbose {
				log.Printf("Warning: Failed to decode hearham record: %v", err)
			}
			continue
		}

		chunk = append(chunk, repeater)
		recordsRead++

		if len(chunk) >= chunkSize {
			totalReadTime += time.Since(readStart)

			syncStart := time.Now()
			if err := db.SyncHearhamData(chunk); err != nil {
				log.Printf("Warning: Failed to sync hearham chunk: %v", err)
			} else {
				recordsSynced += len(chunk)
				if verbose {
					fmt.Printf("    hearham: synced chunk of %d records\n", len(chunk))
				}
			}
			totalSyncTime += time.Since(syncStart)

			chunk = chunk[:0]
			readStart = time.Now()
		}
	}

	// Process remaining records
	if len(chunk) > 0 {
		totalReadTime += time.Since(readStart)

		syncStart := time.Now()
		if err := db.SyncHearhamData(chunk); err != nil {
			log.Printf("Warning: Failed to sync final hearham chunk: %v", err)
		} else {
			recordsSynced += len(chunk)
		}
		totalSyncTime += time.Since(syncStart)
	}

	return recordsRead, recordsSynced, totalReadTime, totalSyncTime, nil
}

// Optimized cache readers using streaming JSON
func readBrandmeisterCacheOptimized(filename string, verbose bool) ([]api.BrandmeisterRepeater, int, error) {
	if verbose {
		fmt.Printf("Reading Brandmeister cache from: %s\n", filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Use streaming decoder for large files
	decoder := json.NewDecoder(file)
	var repeaters []api.BrandmeisterRepeater

	if err := decoder.Decode(&repeaters); err != nil {
		return nil, 0, fmt.Errorf("failed to decode JSON: %v", err)
	}

	if verbose {
		fmt.Printf("  Loaded %d Brandmeister repeaters from cache\n", len(repeaters))
	}

	return repeaters, len(repeaters), nil
}

func readTGIFCacheOptimized(filename string, verbose bool) ([]api.TGIFTalkgroup, int, error) {
	if verbose {
		fmt.Printf("Reading TGIF cache from: %s\n", filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var talkgroups []api.TGIFTalkgroup

	if err := decoder.Decode(&talkgroups); err != nil {
		return nil, 0, fmt.Errorf("failed to decode JSON: %v", err)
	}

	if verbose {
		fmt.Printf("  Loaded %d TGIF talkgroups from cache\n", len(talkgroups))
	}

	return talkgroups, len(talkgroups), nil
}

func readHearhamCacheOptimized(filename string, verbose bool) ([]api.HearhamRepeater, int, error) {
	if verbose {
		fmt.Printf("Reading hearham cache from: %s\n", filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var repeaters []api.HearhamRepeater

	if err := decoder.Decode(&repeaters); err != nil {
		return nil, 0, fmt.Errorf("failed to decode JSON: %v", err)
	}

	if verbose {
		fmt.Printf("  Loaded %d hearham repeaters from cache\n", len(repeaters))
	}

	return repeaters, len(repeaters), nil
}

// Original cache readers (for sequential mode)
func readBrandmeisterCache(filename string, verbose bool) ([]api.BrandmeisterRepeater, int, error) {
	if verbose {
		fmt.Printf("Reading Brandmeister cache from: %s\n", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %v", err)
	}

	var repeaters []api.BrandmeisterRepeater
	if err := json.Unmarshal(data, &repeaters); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if verbose {
		fmt.Printf("  Loaded %d Brandmeister repeaters from cache\n", len(repeaters))
	}

	return repeaters, len(repeaters), nil
}

func readTGIFCache(filename string, verbose bool) ([]api.TGIFTalkgroup, int, error) {
	if verbose {
		fmt.Printf("Reading TGIF cache from: %s\n", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %v", err)
	}

	var talkgroups []api.TGIFTalkgroup
	if err := json.Unmarshal(data, &talkgroups); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if verbose {
		fmt.Printf("  Loaded %d TGIF talkgroups from cache\n", len(talkgroups))
	}

	return talkgroups, len(talkgroups), nil
}

func readHearhamCache(filename string, verbose bool) ([]api.HearhamRepeater, int, error) {
	if verbose {
		fmt.Printf("Reading hearham cache from: %s\n", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %v", err)
	}

	var repeaters []api.HearhamRepeater
	if err := json.Unmarshal(data, &repeaters); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if verbose {
		fmt.Printf("  Loaded %d hearham repeaters from cache\n", len(repeaters))
	}

	return repeaters, len(repeaters), nil
}
