package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// BrandmeisterClient handles API calls to the Brandmeister network
// Brandmeister is a popular DMR network with repeaters and hotspots worldwide
type BrandmeisterClient struct {
	baseURL    string                 // Base URL for API calls
	apiKey     string                 // API key for authentication
	httpClient *http.Client           // HTTP client for making requests
	allData    []BrandmeisterRepeater // Cache of all repeater data
	lastUpdate time.Time              // When we last fetched data
	cacheValid bool                   // Whether our cache is still valid
}

// BrandmeisterRepeater represents a single repeater/hotspot in the Brandmeister network
// BrandmeisterRepeater represents a DMR repeater as provided by the BrandMeister API.
// It contains identification, location, technical, and status information about the repeater.
type BrandmeisterRepeater struct {
	ID          int     `json:"id"`              // Unique repeater ID
	Callsign    string  `json:"callsign"`        // Repeater callsign
	City        string  `json:"city"`            // City location
	State       string  `json:"state"`           // State/province
	Country     string  `json:"country"`         // Country
	TxFreq      string  `json:"tx"`              // TX frequency as string
	RxFreq      string  `json:"rx"`              // RX frequency as string
	ColorCode   int     `json:"colorcode"`       // DMR color code (0-15)
	Latitude    float64 `json:"lat"`             // GPS latitude (note: "lat" not "latitude")
	Longitude   float64 `json:"lng"`             // GPS longitude (note: "lng" not "longitude")
	Status      int     `json:"status"`          // Status code (not boolean)
	Hardware    string  `json:"hardware"`        // Hardware type
	Firmware    string  `json:"firmware"`        // Firmware version (note: "firmware" not "software")
	Website     string  `json:"website"`         // Website URL
	PEP         int     `json:"pep"`             // Power in watts
	AGL         int     `json:"agl"`             // Height above ground level
	LastMaster  int     `json:"lastKnownMaster"` // Last known master ID
	Description string  `json:"description"`     // HTML description
}

// BrandmeisterResponse represents the API response format
type BrandmeisterResponse struct {
	Count     int                    `json:"count"`     // Number of repeaters returned
	Repeaters []BrandmeisterRepeater `json:"repeaters"` // Array of repeater data
}

// NewBrandmeisterClient creates a new Brandmeister API client
// This is the constructor function - it sets up the client with the provided API key
func NewBrandmeisterClient(apiKey string) *BrandmeisterClient {
	return &BrandmeisterClient{
		baseURL: "https://api.brandmeister.network", // Remove /v2 from base URL
		apiKey:  apiKey,                             // API key from configuration
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // 30 second timeout for API calls
		},
		allData:    make([]BrandmeisterRepeater, 0), // Initialize empty slice
		cacheValid: false,                           // Cache starts invalid
	}
}

// Initialize sets up the client and loads initial data if needed
// This checks if we need to refresh our cache based on age
func (c *BrandmeisterClient) Initialize() error {
	// Check if we need to refresh cache (if it's older than 24 hours)
	cacheAge := time.Since(c.lastUpdate)
	cacheMaxAge := 24 * time.Hour // Brandmeister data changes less frequently

	if !c.cacheValid || cacheAge > cacheMaxAge {
		fmt.Println("Cache is stale (>24h old), refreshing on startup...")
		return c.refreshData()
	}

	fmt.Printf("Cache is valid (age: %v), using cached data\n", cacheAge)
	return nil
}

// refreshData fetches fresh data from the Brandmeister API
// refreshData fetches fresh data from the Brandmeister API
func (c *BrandmeisterClient) refreshData() error {
	fmt.Println("Fetching repeater data from Brandmeister.network...")

	// Based on Brandmeister API docs, let's try these endpoints:
	endpoints := []string{
		"/v2/device",         // Devices (repeaters/hotspots) endpoint
		"/v1.0/repeater",     // Alternative repeater endpoint
		"/v1.0/device",       // Alternative device endpoint
		"/api/v1.0/repeater", // Full path with api prefix
		"/api/v1.0/device",   // Full path with api prefix for devices
	}

	var lastErr error

	// Try each endpoint until one works
	for _, endpoint := range endpoints {
		url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
		fmt.Printf("Trying endpoint: %s\n", url)

		// Create the HTTP request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create HTTP request for %s: %v", endpoint, err)
			continue
		}

		// Add headers - Brandmeister might need specific headers
		req.Header.Add("Authorization", "Bearer "+c.apiKey)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/json")

		// Make the HTTP request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to make HTTP request to %s: %v", endpoint, err)
			continue
		}

		fmt.Printf("Response status for %s: %d\n", endpoint, resp.StatusCode)

		// If we get a 404, try the next endpoint
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			lastErr = fmt.Errorf("endpoint %s returned 404", endpoint)
			continue
		}

		// If we get a 401, it's an authentication issue
		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			lastErr = fmt.Errorf("endpoint %s returned 401 - check API key", endpoint)
			continue
		}

		// If we get a different error, show it but keep trying
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Printf("Error response from %s: %s\n", endpoint, string(body))
			lastErr = fmt.Errorf("endpoint %s returned status %d", endpoint, resp.StatusCode)
			continue
		}

		// Success! Read the response
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body from %s: %v", endpoint, err)
			continue
		}

		// Show first part of response for debugging
		bodyStr := string(body)
		fmt.Printf("✓ SUCCESS with endpoint: %s\n", endpoint)
		if len(bodyStr) > 500 {
			fmt.Printf("Raw response (first 500 chars): %s\n", bodyStr[:500])
		} else {
			fmt.Printf("Raw response: %s\n", bodyStr)
		}

		// Try to parse the JSON response
		// First, let's see if it's an array or an object
		var rawResult interface{}
		if err := json.Unmarshal(body, &rawResult); err != nil {
			fmt.Printf("JSON parsing failed: %v\n", err)
			fmt.Printf("Response was: %s\n", bodyStr[:min(1000, len(bodyStr))])
			return fmt.Errorf("failed to decode JSON response from %s: %v", endpoint, err)
		}

		// Now try to parse as our expected structure
		var repeaters []BrandmeisterRepeater
		if err := json.Unmarshal(body, &repeaters); err != nil {
			// Maybe it's wrapped in an object? Let's try a different structure
			var wrappedResponse struct {
				Data    []BrandmeisterRepeater `json:"data"`
				Results []BrandmeisterRepeater `json:"results"`
				Items   []BrandmeisterRepeater `json:"items"`
			}

			if err2 := json.Unmarshal(body, &wrappedResponse); err2 != nil {
				fmt.Printf("Both parsing attempts failed:\n")
				fmt.Printf("  Direct array parse: %v\n", err)
				fmt.Printf("  Wrapped object parse: %v\n", err2)
				fmt.Printf("Raw response structure: %+v\n", rawResult)
				return fmt.Errorf("failed to decode JSON response from %s", endpoint)
			}

			// Use whichever array has data
			if len(wrappedResponse.Data) > 0 {
				repeaters = wrappedResponse.Data
			} else if len(wrappedResponse.Results) > 0 {
				repeaters = wrappedResponse.Results
			} else if len(wrappedResponse.Items) > 0 {
				repeaters = wrappedResponse.Items
			} else {
				return fmt.Errorf("no repeater data found in response from %s", endpoint)
			}
		}

		// Update our cache
		c.allData = repeaters
		c.lastUpdate = time.Now()
		c.cacheValid = true

		fmt.Printf("Successfully loaded %d repeaters from Brandmeister.network using %s\n", len(repeaters), endpoint)
		return nil
	}

	// If we get here, none of the endpoints worked
	return fmt.Errorf("all API endpoints failed, last error: %v", lastErr)
}

// GetAllRepeaters returns all cached repeater data
func (c *BrandmeisterClient) GetAllRepeaters() (*BrandmeisterResponse, error) {
	// Make sure we have data
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	return &BrandmeisterResponse{
		Count:     len(c.allData),
		Repeaters: c.allData,
	}, nil
}

// SearchRepeaters searches for repeaters by callsign, city, or state
func (c *BrandmeisterClient) SearchRepeaters(query string) (*BrandmeisterResponse, error) {
	// Make sure we have data
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	// Convert search query to lowercase for case-insensitive search
	query = strings.ToLower(query)
	var results []BrandmeisterRepeater

	// Search through all repeaters
	for _, repeater := range c.allData {
		// Check if query matches callsign, city, state, or country
		if strings.Contains(strings.ToLower(repeater.Callsign), query) ||
			strings.Contains(strings.ToLower(repeater.City), query) ||
			strings.Contains(strings.ToLower(repeater.State), query) ||
			strings.Contains(strings.ToLower(repeater.Country), query) {
			results = append(results, repeater)
		}
	}

	return &BrandmeisterResponse{
		Count:     len(results),
		Repeaters: results,
	}, nil
}

// GetRepeater looks up a specific repeater by ID
func (c *BrandmeisterClient) GetRepeater(id int) (*BrandmeisterRepeater, error) {
	// Make sure we have data
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	// Search for the repeater with matching ID
	for _, repeater := range c.allData {
		if repeater.ID == id {
			return &repeater, nil
		}
	}

	return nil, fmt.Errorf("repeater %d not found", id)
}

// GetCacheStatus returns information about the current cache
func (c *BrandmeisterClient) GetCacheStatus() map[string]interface{} {
	cacheAge := time.Since(c.lastUpdate)
	needsRefresh := cacheAge > (24 * time.Hour)

	return map[string]interface{}{
		"count":         len(c.allData),
		"last_update":   c.lastUpdate,
		"age":           cacheAge,
		"cache_valid":   c.cacheValid,
		"needs_refresh": needsRefresh,
	}
}

// ForceRefresh forces a refresh of the cache regardless of age
func (c *BrandmeisterClient) ForceRefresh() error {
	fmt.Println("Force refreshing Brandmeister.network data...")
	oldCount := len(c.allData)

	if err := c.refreshData(); err != nil {
		return err
	}

	newCount := len(c.allData)
	if newCount != oldCount {
		fmt.Printf("Repeater data refreshed: %d repeaters (changed from %d)\n", newCount, oldCount)
	} else {
		fmt.Printf("Repeater data refreshed: %d repeaters (no count change)\n", newCount)
	}

	return nil
}

// ensureData makes sure we have valid data, refreshing if necessary
func (c *BrandmeisterClient) ensureData() error {
	if !c.cacheValid || len(c.allData) == 0 {
		return c.refreshData()
	}
	return nil
}

// Helper methods for the repeater struct

// GetLocationString returns a formatted location string
func (r *BrandmeisterRepeater) GetLocationString() string {
	parts := []string{}
	if r.City != "" {
		parts = append(parts, r.City)
	}
	if r.State != "" {
		parts = append(parts, r.State)
	}
	if r.Country != "" {
		parts = append(parts, r.Country)
	}
	return strings.Join(parts, ", ")
}

// GetFrequencyString returns a formatted frequency string
func (r *BrandmeisterRepeater) GetFrequencyString() string {
	// Convert string frequencies to display format
	if r.TxFreq != "" && r.RxFreq != "" {
		return fmt.Sprintf("%s MHz (RX: %s MHz)", r.TxFreq, r.RxFreq)
	} else if r.TxFreq != "" {
		return fmt.Sprintf("%s MHz", r.TxFreq)
	}
	return "Unknown frequency"
}

// GetTxFrequencyFloat returns TX frequency as float64
func (r *BrandmeisterRepeater) GetTxFrequencyFloat() (float64, error) {
	if r.TxFreq == "" {
		return 0, fmt.Errorf("no TX frequency available")
	}
	// Remove any non-numeric characters and parse
	return strconv.ParseFloat(r.TxFreq, 64)
}

// GetRxFrequencyFloat returns RX frequency as float64
func (r *BrandmeisterRepeater) GetRxFrequencyFloat() (float64, error) {
	if r.RxFreq == "" {
		return 0, fmt.Errorf("no RX frequency available")
	}
	return strconv.ParseFloat(r.RxFreq, 64)
}

// IsOnline returns whether the repeater is currently online
// Based on Brandmeister status codes (this might need adjustment)
func (r *BrandmeisterRepeater) IsOnline() bool {
	// Status 1 might mean online, but we need to check the API docs
	// For now, let's assume status > 0 means some level of activity
	return r.Status > 0
}

// GetCoordinates returns latitude and longitude as strings
func (r *BrandmeisterRepeater) GetCoordinates() (string, string) {
	return fmt.Sprintf("%.6f", r.Latitude), fmt.Sprintf("%.6f", r.Longitude)
}

// GetSlotInfo returns slot information (Brandmeister-specific)
func (r *BrandmeisterRepeater) GetSlotInfo() string {
	// Brandmeister repeaters typically have 2 slots
	return "Dual Slot DMR"
}

// GetPowerInfo returns power and antenna information
func (r *BrandmeisterRepeater) GetPowerInfo() string {
	if r.PEP > 0 && r.AGL > 0 {
		return fmt.Sprintf("%d watts, %d ft AGL", r.PEP, r.AGL)
	} else if r.PEP > 0 {
		return fmt.Sprintf("%d watts", r.PEP)
	} else if r.AGL > 0 {
		return fmt.Sprintf("%d ft AGL", r.AGL)
	}
	return "Power info not available"
}
