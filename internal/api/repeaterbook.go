package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// RepeaterBook repeater data structure
type RepeaterBookRepeater struct {
	StateID    string `json:"State_ID"`
	Rptr_ID    string `json:"Rptr_ID"`
	Frequency  string `json:"Frequency"`
	InputFreq  string `json:"Input_Freq"`
	AccessTone string `json:"Access_Tone"`
	Use        string `json:"Use"`
	Callsign   string `json:"Callsign"`
	Nearest    string `json:"Nearest_City"`
	Landmark   string `json:"Landmark"`
	County     string `json:"County"`
	State      string `json:"State"`
	Country    string `json:"Country"`
	Latitude   string `json:"Lat"`
	Longitude  string `json:"Long"`
	Status     string `json:"Operational_Status"`
	ARES       string `json:"ARES"`
	RACES      string `json:"RACES"`
	SKYWARN    string `json:"SKYWARN"`
	Canopy     string `json:"Canopy"`
	DSTAR      string `json:"DSTAR"`
	DMR        string `json:"DMR"`
	YSF        string `json:"YSF"`
	NXDN       string `json:"NXDN"`
	P25        string `json:"P25"`
	TETRA      string `json:"TETRA"`
	Notes      string `json:"Notes"`
	LastUpdate string `json:"Last_Update"`
}

// Helper methods for RepeaterBook data
func (r *RepeaterBookRepeater) GetLatitude() (float64, error) {
	if r.Latitude == "" {
		return 0, fmt.Errorf("no latitude data")
	}
	return strconv.ParseFloat(r.Latitude, 64)
}

func (r *RepeaterBookRepeater) GetLongitude() (float64, error) {
	if r.Longitude == "" {
		return 0, fmt.Errorf("no longitude data")
	}
	return strconv.ParseFloat(r.Longitude, 64)
}

func (r *RepeaterBookRepeater) GetFrequencyFloat() (float64, error) {
	if r.Frequency == "" {
		return 0, fmt.Errorf("no frequency data")
	}
	return strconv.ParseFloat(r.Frequency, 64)
}

func (r *RepeaterBookRepeater) IsDigital() bool {
	return r.DSTAR != "" || r.DMR != "" || r.YSF != "" || r.NXDN != "" || r.P25 != "" || r.TETRA != ""
}

func (r *RepeaterBookRepeater) GetDigitalModes() []string {
	var modes []string
	if r.DSTAR != "" && r.DSTAR != "No" {
		modes = append(modes, "D-STAR")
	}
	if r.DMR != "" && r.DMR != "No" {
		modes = append(modes, "DMR")
	}
	if r.YSF != "" && r.YSF != "No" {
		modes = append(modes, "YSF")
	}
	if r.NXDN != "" && r.NXDN != "No" {
		modes = append(modes, "NXDN")
	}
	if r.P25 != "" && r.P25 != "No" {
		modes = append(modes, "P25")
	}
	if r.TETRA != "" && r.TETRA != "No" {
		modes = append(modes, "TETRA")
	}
	return modes
}

// RepeaterBook API response structure
type RepeaterBookResponse struct {
	Count   int                    `json:"count"`
	Results []RepeaterBookRepeater `json:"results"`
}

// RepeaterBook API client
type RepeaterBookClient struct {
	APIKey    string
	BaseURL   string
	UserAgent string
	client    *http.Client
}

// Create new RepeaterBook client
func NewRepeaterBookClient(apiKey string) *RepeaterBookClient {
	return &RepeaterBookClient{
		APIKey:    apiKey,
		BaseURL:   "https://www.repeaterbook.com/api",
		UserAgent: "DigiLogRT/0.1.0 Amateur Radio Digital Logging Tool (https://github.com/unklstewy/digiLog, unklstewy@example.com)",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search repeaters by state
func (c *RepeaterBookClient) SearchByState(state string) (*RepeaterBookResponse, error) {
	u, err := url.Parse(c.BaseURL + "/export.php")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	params := url.Values{}
	params.Add("state", state)
	params.Add("format", "json")
	if c.APIKey != "" {
		params.Add("api_key", c.APIKey)
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set required User-Agent header
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var rbResp RepeaterBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&rbResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &rbResp, nil
}

// Search repeaters by location (lat/lon with radius)
func (c *RepeaterBookClient) SearchByLocation(lat, lng float64, radius int) (*RepeaterBookResponse, error) {
	u, err := url.Parse(c.BaseURL + "/proximity.php")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	params := url.Values{}
	params.Add("lat", fmt.Sprintf("%.6f", lat))
	params.Add("long", fmt.Sprintf("%.6f", lng))
	params.Add("distance", fmt.Sprintf("%d", radius))
	params.Add("format", "json")
	if c.APIKey != "" {
		params.Add("api_key", c.APIKey)
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var rbResp RepeaterBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&rbResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &rbResp, nil
}

// Test the API connection
func (c *RepeaterBookClient) TestConnection() error {
	// Test with a small search in Pennsylvania (should return results)
	_, err := c.SearchByState("Pennsylvania")
	return err
}
