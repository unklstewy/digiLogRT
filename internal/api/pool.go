package api

import (
	"fmt"
	"sync"
	"time"
)

// ClientPool manages reusable API client connections
type ClientPool struct {
	brandmeister *BrandmeisterClient
	tgif         *TGIFClient
	hearham      *HearhamClient
	initOnce     sync.Once
	initTime     time.Duration
}

var globalPool *ClientPool
var poolOnce sync.Once

// GetGlobalPool returns a singleton client pool
func GetGlobalPool() *ClientPool {
	poolOnce.Do(func() {
		globalPool = &ClientPool{}
	})
	return globalPool
}

// ...existing code...
// WarmCaches proactively refreshes caches if they're older than maxAge
func (p *ClientPool) WarmCaches(brandmeisterKey string, maxAge time.Duration) error {
	fmt.Printf("🔥 Checking cache freshness (max age: %v)\n", maxAge)

	var wg sync.WaitGroup
	errors := make(chan error, 3)

	// Check and warm each cache in parallel
	if brandmeisterKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := NewBrandmeisterClient(brandmeisterKey)
			if needsRefresh, age := client.CheckCacheAge(); needsRefresh && age > maxAge {
				fmt.Printf("  🔄 Brandmeister cache is %v old, refreshing...\n", age)
				if err := client.RefreshCache(); err != nil {
					errors <- fmt.Errorf("brandmeister cache refresh failed: %v", err)
					return
				}
				fmt.Printf("  ✓ Brandmeister cache refreshed\n")
			} else {
				fmt.Printf("  ✓ Brandmeister cache is fresh (%v old)\n", age)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		client := NewTGIFClient()
		if needsRefresh, age := client.CheckCacheAge(); needsRefresh && age > maxAge {
			fmt.Printf("  🔄 TGIF cache is %v old, refreshing...\n", age)
			if err := client.RefreshCache(); err != nil {
				errors <- fmt.Errorf("tgif cache refresh failed: %v", err)
				return
			}
			fmt.Printf("  ✓ TGIF cache refreshed\n")
		} else {
			fmt.Printf("  ✓ TGIF cache is fresh (%v old)\n", age)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		client := NewHearhamClient()
		if needsRefresh, age := client.CheckCacheAge(); needsRefresh && age > maxAge {
			fmt.Printf("  🔄 hearham cache is %v old, refreshing...\n", age)
			if err := client.RefreshCache(); err != nil {
				errors <- fmt.Errorf("hearham cache refresh failed: %v", err)
				return
			}
			fmt.Printf("  ✓ hearham cache refreshed\n")
		} else {
			fmt.Printf("  ✓ hearham cache is fresh (%v old)\n", age)
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

// ...existing code...

// Initialize all clients once
func (p *ClientPool) Initialize(brandmeisterKey string) error {
	var initErr error

	p.initOnce.Do(func() {
		start := time.Now()

		// Initialize all clients in parallel
		var wg sync.WaitGroup
		errors := make(chan error, 3)

		// Brandmeister
		if brandmeisterKey != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.brandmeister = NewBrandmeisterClient(brandmeisterKey)
				if err := p.brandmeister.Initialize(); err != nil {
					errors <- err
				}
			}()
		}

		// TGIF
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.tgif = NewTGIFClient()
			if err := p.tgif.Initialize(); err != nil {
				errors <- err
			}
		}()

		// Hearham
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.hearham = NewHearhamClient()
			if err := p.hearham.Initialize(); err != nil {
				errors <- err
			}
		}()

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			if err != nil && initErr == nil {
				initErr = err
			}
		}

		p.initTime = time.Since(start)
	})

	return initErr
}

// GetClients returns initialized clients
func (p *ClientPool) GetClients() (*BrandmeisterClient, *TGIFClient, *HearhamClient) {
	return p.brandmeister, p.tgif, p.hearham
}

// GetInitTime returns the total initialization time
func (p *ClientPool) GetInitTime() time.Duration {
	return p.initTime
}
