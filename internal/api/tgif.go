package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TGIFTalkgroup struct {
	ID          string `json:"id"` // Changed from int to string
	Name        string `json:"name"`
	Website     string `json:"website"`
	Description string `json:"description"`
	// Add other fields as needed
}

// Add method to decode description
func (tg *TGIFTalkgroup) GetDecodedDescription() (string, error) {
	if tg.Description == "" {
		return "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(tg.Description)
	if err != nil {
		return tg.Description, nil // Return original if decode fails
	}

	return string(decoded), nil
}

// Add a helper method to get ID as integer
func (tg *TGIFTalkgroup) GetIDInt() (int, error) {
	return strconv.Atoi(tg.ID)
}

// Update your existing methods to work with string IDs
func (tg *TGIFTalkgroup) GetSlotInfo() string {
	// Implementation depends on how you determine slot info
	// For now, return a placeholder
	return "Slot 1" // Update based on your logic
}

func (tg *TGIFTalkgroup) IsActive() bool {
	// Implementation depends on how you determine if active
	// For now, return true
	return true // Update based on your logic
}

// TGIF API response structures
type TGIFTalkgroupResponse struct {
	Status     string          `json:"status,omitempty"`
	Count      int             `json:"count,omitempty"`
	Talkgroups []TGIFTalkgroup `json:"talkgroups,omitempty"`
}

// TGIF API client with intelligent caching
type TGIFClient struct {
	BaseURL        string
	client         *http.Client
	allData        []TGIFTalkgroup
	lastUpdate     time.Time
	cacheTime      time.Duration
	startupRefresh time.Duration
}

// Create new TGIF client with configurable caching
func NewTGIFClient() *TGIFClient {
	return &TGIFClient{
		BaseURL: "https://api.tgif.network/dmr/talkgroups/json",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheTime:      6 * time.Hour, // DMR data changes more frequently
		startupRefresh: 2 * time.Hour, // Refresh if older than 2 hours on startup
	}
}

// Fetch all talkgroup data from TGIF with change detection
func (c *TGIFClient) fetchAllData() error {
	fmt.Println("Fetching talkgroup data from TGIF.network...")

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

	// Try to decode as array first (most likely format)
	var newTalkgroups []TGIFTalkgroup
	if err := json.Unmarshal(body, &newTalkgroups); err == nil {
		// Check for changes if we have existing data
		if len(c.allData) > 0 {
			oldCount := len(c.allData)
			newCount := len(newTalkgroups)

			if newCount != oldCount {
				fmt.Printf("Talkgroup count changed: %d → %d (%+d)\n", oldCount, newCount, newCount-oldCount)
			} else {
				fmt.Printf("Talkgroup data refreshed: %d talkgroups (no count change)\n", newCount)
			}
		} else {
			fmt.Printf("Successfully loaded %d talkgroups from TGIF.network\n", len(newTalkgroups))
		}

		c.allData = newTalkgroups
		c.lastUpdate = time.Now()
		return nil
	}

	// If array decode fails, try as response object
	var response TGIFTalkgroupResponse
	if err := json.Unmarshal(body, &response); err == nil {
		c.allData = response.Talkgroups
		c.lastUpdate = time.Now()
		fmt.Printf("Successfully loaded %d talkgroups from TGIF.network (wrapped format)\n", len(response.Talkgroups))
		return nil
	}

	// If both fail, show what we got
	fmt.Printf("Raw response (first 500 chars): %s\n", string(body[:min(500, len(body))]))
	return fmt.Errorf("failed to decode JSON response")
}

// Ensure we have fresh data with intelligent refresh logic
func (c *TGIFClient) ensureData() error {
	now := time.Now()

	// If no data, always fetch
	if len(c.allData) == 0 {
		fmt.Println("No cached data, fetching from TGIF.network...")
		return c.fetchAllData()
	}

	// If cache is very old, force refresh
	if now.Sub(c.lastUpdate) > c.cacheTime {
		fmt.Printf("Cache expired (%v old), refreshing...\n", now.Sub(c.lastUpdate).Round(time.Minute))
		return c.fetchAllData()
	}

	// Cache is valid, use existing data
	return nil
}

// Force refresh of data (for manual updates)
func (c *TGIFClient) ForceRefresh() error {
	fmt.Println("Force refreshing TGIF.network data...")
	return c.fetchAllData()
}

// Check if data should be refreshed on startup
func (c *TGIFClient) ShouldRefreshOnStartup() bool {
	if len(c.allData) == 0 {
		return true
	}
	return time.Since(c.lastUpdate) > c.startupRefresh
}

// Initialize data with startup refresh logic
func (c *TGIFClient) Initialize() error {
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

	fmt.Printf("Using cached data (%v old, %d talkgroups)\n",
		time.Since(c.lastUpdate).Round(time.Minute), len(c.allData))
	return nil
}

// Get cache status information
func (c *TGIFClient) GetCacheStatus() map[string]interface{} {
	return map[string]interface{}{
		"count":         len(c.allData),
		"last_update":   c.lastUpdate,
		"age":           time.Since(c.lastUpdate),
		"cache_valid":   time.Since(c.lastUpdate) < c.cacheTime,
		"needs_refresh": c.ShouldRefreshOnStartup(),
	}
}

// Search talkgroups by ID (local filtering)
func (c *TGIFClient) GetTalkgroup(id int) (*TGIFTalkgroup, error) {
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	for _, tg := range c.allData {
		if tg.ID == strconv.Itoa(id) {
			return &tg, nil
		}
	}

	return nil, fmt.Errorf("talkgroup %d not found", id)
}

// Search talkgroups by name/description (local filtering)
func (c *TGIFClient) SearchTalkgroups(query string) (*TGIFTalkgroupResponse, error) {
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	var filtered []TGIFTalkgroup
	queryLower := strings.ToLower(query)

	for _, tg := range c.allData {
		if strings.Contains(strings.ToLower(tg.Name), queryLower) ||
			strings.Contains(strings.ToLower(tg.Description), queryLower) ||
			tg.ID == query {
			filtered = append(filtered, tg)
		}
	}

	return &TGIFTalkgroupResponse{
		Status:     "success",
		Count:      len(filtered),
		Talkgroups: filtered,
	}, nil
}

// Get all talkgroups
func (c *TGIFClient) GetAllTalkgroups() (*TGIFTalkgroupResponse, error) {
	if err := c.ensureData(); err != nil {
		return nil, err
	}

	return &TGIFTalkgroupResponse{
		Status:     "success",
		Count:      len(c.allData),
		Talkgroups: c.allData,
	}, nil
}

// Test the API connection
func (c *TGIFClient) TestConnection() error {
	err := c.fetchAllData()
	if err != nil {
		return err
	}
	if len(c.allData) == 0 {
		return fmt.Errorf("no talkgroup data received from API")
	}
	return nil
}
