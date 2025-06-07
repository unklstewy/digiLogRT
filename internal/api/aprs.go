package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// FlexibleTime handles both string and int64 timestamps from the API
type FlexibleTime struct {
	Value int64
}

func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as int64 first
	var intValue int64
	if err := json.Unmarshal(data, &intValue); err == nil {
		ft.Value = intValue
		return nil
	}

	// If that fails, try as string
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err != nil {
		return err
	}

	// Convert string to int64
	if stringValue == "" {
		ft.Value = 0
		return nil
	}

	intValue, err := strconv.ParseInt(stringValue, 10, 64)
	if err != nil {
		ft.Value = 0
		return nil // Don't fail on bad time data
	}

	ft.Value = intValue
	return nil
}

// FlexibleFloat handles both string and float64 coordinates from the API
type FlexibleFloat struct {
	Value float64
}

func (ff *FlexibleFloat) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as float64 first
	var floatValue float64
	if err := json.Unmarshal(data, &floatValue); err == nil {
		ff.Value = floatValue
		return nil
	}

	// If that fails, try as string
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err != nil {
		return err
	}

	// Convert string to float64
	if stringValue == "" {
		ff.Value = 0.0
		return nil
	}

	floatValue, err := strconv.ParseFloat(stringValue, 64)
	if err != nil {
		ff.Value = 0.0
		return nil // Don't fail on bad coordinate data
	}

	ff.Value = floatValue
	return nil
}

// FlexibleInt handles both string and int values from the API
type FlexibleInt struct {
	Value int
}

func (fi *FlexibleInt) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as int first
	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		fi.Value = intValue
		return nil
	}

	// If that fails, try as string
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err != nil {
		return err
	}

	// Convert string to int
	if stringValue == "" {
		fi.Value = 0
		return nil
	}

	intValue, err := strconv.Atoi(stringValue)
	if err != nil {
		fi.Value = 0
		return nil // Don't fail on bad data
	}

	fi.Value = intValue
	return nil
}

// APRS station data structure with flexible field handling
type APRSStation struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Time        FlexibleTime  `json:"time"`
	LastTime    FlexibleTime  `json:"lasttime"`
	Lat         FlexibleFloat `json:"lat"`
	Lng         FlexibleFloat `json:"lng"`
	Course      FlexibleInt   `json:"course"`
	Speed       FlexibleInt   `json:"speed"`
	Altitude    FlexibleInt   `json:"altitude"`
	Comment     string        `json:"comment"`
	Path        string        `json:"path"`
	PHG         string        `json:"phg"`
	Status      string        `json:"status"`
	Symbol      string        `json:"symbol"`
	SymbolTable string        `json:"srccall"`
}

// Helper method to convert Unix timestamp to readable time
func (s *APRSStation) GetTimeString() string {
	if s.Time.Value == 0 {
		return "Unknown"
	}
	return time.Unix(s.Time.Value, 0).Format("2006-01-02 15:04:05")
}

func (s *APRSStation) GetLastTimeString() string {
	if s.LastTime.Value == 0 {
		return "Unknown"
	}
	return time.Unix(s.LastTime.Value, 0).Format("2006-01-02 15:04:05")
}

// Helper methods for coordinates
func (s *APRSStation) GetLatitude() float64 {
	return s.Lat.Value
}

func (s *APRSStation) GetLongitude() float64 {
	return s.Lng.Value
}

// APRS API response structure
type APRSResponse struct {
	Command string        `json:"command"`
	Result  string        `json:"result"`
	What    string        `json:"what"`
	Found   int           `json:"found"`
	Entries []APRSStation `json:"entries"`
}

// APRS API client
type APRSClient struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

// Create new APRS client
func NewAPRSClient(apiKey string) *APRSClient {
	return &APRSClient{
		APIKey:  apiKey,
		BaseURL: "https://api.aprs.fi/api",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get station information by callsign
func (c *APRSClient) GetStation(callsign string) (*APRSResponse, error) {
	// Build the URL with parameters
	u, err := url.Parse(c.BaseURL + "/get")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	// Add query parameters
	params := url.Values{}
	params.Add("name", callsign)
	params.Add("what", "loc")
	params.Add("apikey", c.APIKey)
	params.Add("format", "json")
	u.RawQuery = params.Encode()

	// Make the request
	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Parse JSON response
	var aprsResp APRSResponse
	if err := json.NewDecoder(resp.Body).Decode(&aprsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &aprsResp, nil
}

// Get stations within a radius of coordinates
func (c *APRSClient) GetStationsInRadius(lat, lng float64, radius int) (*APRSResponse, error) {
	u, err := url.Parse(c.BaseURL + "/get")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	params := url.Values{}
	params.Add("lat", fmt.Sprintf("%.6f", lat))
	params.Add("lng", fmt.Sprintf("%.6f", lng))
	params.Add("distance", fmt.Sprintf("%d", radius))
	params.Add("what", "loc")
	params.Add("apikey", c.APIKey)
	params.Add("format", "json")
	u.RawQuery = params.Encode()

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var aprsResp APRSResponse
	if err := json.NewDecoder(resp.Body).Decode(&aprsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &aprsResp, nil
}

// Test the API connection
func (c *APRSClient) TestConnection() error {
	// Test with a known callsign
	_, err := c.GetStation("OH7RDA")
	return err
}
