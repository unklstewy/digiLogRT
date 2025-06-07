package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// hearham.com repeater data structure (corrected based on actual API response)
type HearhamRepeater struct {
	ID           int     `json:"id"`
	Callsign     string  `json:"callsign"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	City         string  `json:"city"`
	Group        string  `json:"group"`
	InternetNode string  `json:"internet_node"`
	Mode         string  `json:"mode"`
	Encode       string  `json:"encode"`      // CTCSS encode tone
	Decode       string  `json:"decode"`      // CTCSS decode tone
	Frequency    int64   `json:"frequency"`   // Frequency in Hz
	Offset       int64   `json:"offset"`      // Offset in Hz
	Description  string  `json:"description"` // Notes/description
	Power        string  `json:"power"`
	Operational  int     `json:"operational"` // 1 = operational, 0 = not
	Restriction  string  `json:"restriction"`
}

// hearham.com API client with intelligent caching
type HearhamClient struct {
	BaseURL         string
	client          *http.Client
	allData         []HearhamRepeater
	lastUpdate      time.Time
	cacheValid      bool // Add this field
	cacheTime       time.Duration
	startupRefresh  time.Duration // How old cache can be before forcing refresh on startup
	backgroundCheck time.Duration // How often to check for updates in background
}

// getCacheFile returns the path to the cache file
func (c *HearhamClient) getCacheFile() string {
	// Use a cache directory in the user's cache folder or temp
	cacheDir := filepath.Join(os.TempDir(), "digiLogRT", "cache")
	os.MkdirAll(cacheDir, 0755) // Create directory if it doesn't exist
	return filepath.Join(cacheDir, "hearham_repeaters.json")
}

// CheckCacheAge returns whether cache needs refresh and current age
func (c *HearhamClient) CheckCacheAge() (bool, time.Duration) {
	cacheFile := c.getCacheFile()

	// Check if cache file exists
	info, err := os.Stat(cacheFile)
	if err != nil {
		// Cache doesn't exist, needs refresh
		return true, 0
	}

	age := time.Since(info.ModTime())
	maxAge := 6 * time.Hour // hearham cache valid for 6 hours

	return age > maxAge, age
}

// RefreshCache forces a cache refresh
func (c *HearhamClient) RefreshCache() error {
	// Simply delete the cache file, next GetAllRepeaters call will refresh
	cacheFile := c.getCacheFile()
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %v", err)
	}

	// Force a fresh fetch
	_, err := c.GetAllRepeaters()
	return err
}

// Helper methods for hearham data
func (h *HearhamRepeater) GetFrequencyMHz() float64 {
	return float64(h.Frequency) / 1000000.0
}

func (h *HearhamRepeater) GetOffsetMHz() float64 {
	return float64(h.Offset) / 1000000.0
}

func (h *HearhamRepeater) GetInputFrequencyMHz() float64 {
	return h.GetFrequencyMHz() + h.GetOffsetMHz()
}

func (h *HearhamRepeater) IsOperational() bool {
	return h.Operational == 1
}

func (h *HearhamRepeater) GetLocation() string {
	if h.City != "" {
		return h.City
	}
	return "Unknown"
}

func (h *HearhamRepeater) GetTone() string {
	if h.Encode != "" && h.Decode != "" {
		if h.Encode == h.Decode {
			return h.Encode
		}
		return fmt.Sprintf("Encode: %s, Decode: %s", h.Encode, h.Decode)
	}
	if h.Encode != "" {
		return fmt.Sprintf("Encode: %s", h.Encode)
	}
	if h.Decode != "" {
		return fmt.Sprintf("Decode: %s", h.Decode)
	}
	return ""
}

// Extract state from city field (for US/Canada locations)
func (h *HearhamRepeater) GetState() string {
	parts := strings.Split(h.City, ",")
	if len(parts) >= 2 {
		// Look for state/province in the last parts
		for i := len(parts) - 1; i >= 0; i-- {
			part := strings.TrimSpace(parts[i])
			// Check if it's a 2-letter state/province code
			if len(part) == 2 && strings.ToUpper(part) == part {
				return part
			}
			// Check for common state abbreviations in longer strings
			if strings.Contains(part, " ") {
				subParts := strings.Fields(part)
				for _, subPart := range subParts {
					if len(subPart) == 2 && strings.ToUpper(subPart) == subPart {
						return subPart
					}
				}
			}
		}
	}
	return ""
}

// Calculate distance between two points using Haversine formula
func (h *HearhamRepeater) DistanceFromPoint(lat, lng float64) float64 {
	const earthRadius = 6371 // Earth's radius in kilometers

	lat1 := h.Latitude * math.Pi / 180
	lng1 := h.Longitude * math.Pi / 180
	lat2 := lat * math.Pi / 180
	lng2 := lng * math.Pi / 180

	dlat := lat2 - lat1
	dlng := lng2 - lng1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// hearham.com API response structure
type HearhamResponse struct {
	Status    string            `json:"status,omitempty"`
	Count     int               `json:"count,omitempty"`
	Message   string            `json:"message,omitempty"`
	Repeaters []HearhamRepeater `json:"repeaters,omitempty"`
}

// Create new hearham client with configurable caching
func NewHearhamClient() *HearhamClient {
	return &HearhamClient{
		BaseURL: "https://hearham.com/api/repeaters/v1",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		cacheTime:       24 * time.Hour, // Cache valid for 24 hours
		startupRefresh:  6 * time.Hour,  // Force refresh if cache older than 6 hours on startup
		backgroundCheck: 12 * time.Hour, // Check for updates every 12 hours
	}
}

// Fetch all repeater data from hearham.com with change detection
func (c *HearhamClient) fetchAllData() error {
	fmt.Println("Fetching repeater data from hearham.com...")

	resp, err := c.client.Get(c.BaseURL)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Read the response body
	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Decode as array
	var newRepeaters []HearhamRepeater
	if err := json.Unmarshal(body, &newRepeaters); err != nil {
		fmt.Printf("Raw response (first 500 chars): %s\n", string(body[:min(500, len(body))]))
		return fmt.Errorf("failed to decode JSON response: %v", err)
	}

	// Check for changes if we have existing data
	if len(c.allData) > 0 {
		oldCount := len(c.allData)
		newCount := len(newRepeaters)

		if newCount != oldCount {
			fmt.Printf("Repeater count changed: %d → %d (%+d)\n", oldCount, newCount, newCount-oldCount)
		} else {
			fmt.Printf("Repeater data refreshed: %d repeaters (no count change)\n", newCount)
		}

		// Could add more sophisticated change detection here
		// (e.g., check for modified repeaters, new callsigns, etc.)
	} else {
		fmt.Printf("Successfully loaded %d repeaters from hearham.com\n", len(newRepeaters))
	}

	c.allData = newRepeaters
	c.lastUpdate = time.Now()
	return nil
}

// Ensure we have fresh data with intelligent refresh logic
func (c *HearhamClient) ensureData() error {
	now := time.Now()

	// If no data, always fetch
	if len(c.allData) == 0 {
		fmt.Println("No cached data, fetching from hearham.com...")
		return c.fetchAllData()
	}

	// If cache is very old (beyond cacheTime), force refresh
	if now.Sub(c.lastUpdate) > c.cacheTime {
		fmt.Printf("Cache expired (%v old), refreshing...\n", now.Sub(c.lastUpdate).Round(time.Minute))
		return c.fetchAllData()
	}

	// Cache is valid, use existing data
	return nil
}

// Force refresh of data (for manual updates)
func (c *HearhamClient) ForceRefresh() error {
	fmt.Println("Force refreshing hearham.com data...")
	return c.fetchAllData()
}

// Check if data should be refreshed on startup
func (c *HearhamClient) ShouldRefreshOnStartup() bool {
	if len(c.allData) == 0 {
		return true
	}
	return time.Since(c.lastUpdate) > c.startupRefresh
}

// Initialize data with startup refresh logic
func (c *HearhamClient) Initialize() error {
	// Check if we should refresh on startup
	if c.ShouldRefreshOnStartup() {
		fmt.Printf("Cache is stale (>%v old), refreshing on startup...\n", c.startupRefresh)
		return c.fetchAllData()
	}

	// Try to use cached data, fallback to fetch if problems
	if err := c.ensureData(); err != nil {
		fmt.Printf("Failed to use cached data, fetching fresh: %v\n", err)
		return c.fetchAllData()
	}

	fmt.Printf("Using cached data (%v old, %d repeaters)\n",
		time.Since(c.lastUpdate).Round(time.Minute), len(c.allData))
	return nil
}

// Get cache status information
func (c *HearhamClient) GetCacheStatus() map[string]interface{} {
	return map[string]interface{}{
		"count":         len(c.allData),
		"last_update":   c.lastUpdate,
		"age":           time.Since(c.lastUpdate),
		"cache_valid":   time.Since(c.lastUpdate) < c.cacheTime,
		"needs_refresh": c.ShouldRefreshOnStartup(),
	}
}

// Search repeaters by state (local filtering using extracted state from city field)
func (c *HearhamClient) SearchByState(state string) (*HearhamResponse, error) {
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	var filtered []HearhamRepeater
	stateUpper := strings.ToUpper(state)

	for _, repeater := range c.allData {
		repeaterState := strings.ToUpper(repeater.GetState())
		if repeaterState == stateUpper ||
			strings.Contains(strings.ToUpper(repeater.City), stateUpper) {
			filtered = append(filtered, repeater)
		}
	}

	return &HearhamResponse{
		Status:    "success",
		Count:     len(filtered),
		Repeaters: filtered,
	}, nil
}

// Search repeaters by frequency range (local filtering, converting from Hz to MHz)
func (c *HearhamClient) SearchByFrequency(minFreqMHz, maxFreqMHz float64) (*HearhamResponse, error) {
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	var filtered []HearhamRepeater
	for _, repeater := range c.allData {
		freqMHz := repeater.GetFrequencyMHz()
		if freqMHz >= minFreqMHz && freqMHz <= maxFreqMHz {
			filtered = append(filtered, repeater)
		}
	}

	return &HearhamResponse{
		Status:    "success",
		Count:     len(filtered),
		Repeaters: filtered,
	}, nil
}

// Search repeaters by location with radius (local filtering)
func (c *HearhamClient) SearchByLocation(lat, lng float64, radiusKm int) (*HearhamResponse, error) {
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	var filtered []HearhamRepeater
	for _, repeater := range c.allData {
		if repeater.Latitude != 0 && repeater.Longitude != 0 {
			distance := repeater.DistanceFromPoint(lat, lng)
			if distance <= float64(radiusKm) {
				filtered = append(filtered, repeater)
			}
		}
	}

	return &HearhamResponse{
		Status:    "success",
		Count:     len(filtered),
		Repeaters: filtered,
	}, nil
}

// ...existing code...

// GetAllRepeaters returns all cached repeater data with file caching
func (c *HearhamClient) GetAllRepeaters() ([]HearhamRepeater, error) {
	// Try to load from file cache first
	if data, err := c.loadFromCache(); err == nil {
		c.allData = data
		c.cacheValid = true
		return data, nil
	}

	// If file cache miss, fetch from API
	if err := c.fetchAllData(); err != nil { // Use fetchAllData instead of refreshData
		return nil, err
	}

	// Save to file cache
	if err := c.saveToCache(c.allData); err != nil {
		log.Printf("Warning: Failed to save cache to file: %v", err)
	}

	return c.allData, nil
}

// loadFromCache loads data from file cache
func (c *HearhamClient) loadFromCache() ([]HearhamRepeater, error) {
	cacheFile := c.getCacheFile()

	// Check if file exists and is fresh enough
	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil, err
	}

	// Check cache age
	age := time.Since(info.ModTime())
	if age > 6*time.Hour {
		return nil, fmt.Errorf("cache too old: %v", age)
	}

	// Read and parse cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var repeaters []HearhamRepeater
	err = json.Unmarshal(data, &repeaters)
	return repeaters, err
}

// saveToCache saves data to file cache
func (c *HearhamClient) saveToCache(data []HearhamRepeater) error {
	cacheFile := c.getCacheFile()

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return err
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(cacheFile, jsonData, 0644)
}

// ...existing code...

// Get all data (for testing)
func (c *HearhamClient) GetAllData() (*HearhamResponse, error) {
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	return &HearhamResponse{
		Status:    "success",
		Count:     len(c.allData),
		Repeaters: c.allData,
	}, nil
}

// Test the API connection
func (c *HearhamClient) TestConnection() error {
	err := c.fetchAllData()
	if err != nil {
		return err
	}
	if len(c.allData) == 0 {
		return fmt.Errorf("no data received from API")
	}
	return nil
}
