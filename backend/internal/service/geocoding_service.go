package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// GeocodingService handles geocoding operations using Google Maps API
type GeocodingService interface {
	GeocodeAddress(address string) (*GeocodingResult, error)
	ReverseGeocode(lat, lng float64) (*GeocodingResult, error)
}

// GeocodingResult contains the result of a geocoding operation
type GeocodingResult struct {
	Latitude          float64           `json:"latitude"`
	Longitude         float64           `json:"longitude"`
	FormattedAddress  string            `json:"formatted_address"`
	PlaceID           string            `json:"place_id"`
	AddressComponents map[string]string `json:"address_components"`
}

type geocodingServiceImpl struct {
	apiKey     string
	httpClient *http.Client
}

// NewGeocodingService creates a new geocoding service instance
func NewGeocodingService(apiKey string) GeocodingService {
	return &geocodingServiceImpl{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GeocodeAddress converts an address string to geographic coordinates
func (s *geocodingServiceImpl) GeocodeAddress(address string) (*GeocodingResult, error) {
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	// Build API request URL
	baseURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Add("address", address)
	params.Add("key", s.apiKey)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := s.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make geocoding request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read geocoding response: %w", err)
	}

	// Parse response
	var apiResponse geocodingAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse geocoding response: %w", err)
	}

	// Handle API errors
	if apiResponse.Status != "OK" {
		return nil, s.handleAPIError(apiResponse.Status, apiResponse.ErrorMessage)
	}

	// Check if we have results
	if len(apiResponse.Results) == 0 {
		return nil, fmt.Errorf("no results found for address: %s", address)
	}

	// Parse first result (most relevant)
	return s.parseGeocodingResult(apiResponse.Results[0])
}

// ReverseGeocode converts geographic coordinates to an address
func (s *geocodingServiceImpl) ReverseGeocode(lat, lng float64) (*GeocodingResult, error) {
	if lat < -90 || lat > 90 {
		return nil, fmt.Errorf("invalid latitude: %f (must be between -90 and 90)", lat)
	}
	if lng < -180 || lng > 180 {
		return nil, fmt.Errorf("invalid longitude: %f (must be between -180 and 180)", lng)
	}

	// Build API request URL
	baseURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Add("latlng", fmt.Sprintf("%f,%f", lat, lng))
	params.Add("key", s.apiKey)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := s.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make reverse geocoding request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read reverse geocoding response: %w", err)
	}

	// Parse response
	var apiResponse geocodingAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse reverse geocoding response: %w", err)
	}

	// Handle API errors
	if apiResponse.Status != "OK" {
		return nil, s.handleAPIError(apiResponse.Status, apiResponse.ErrorMessage)
	}

	// Check if we have results
	if len(apiResponse.Results) == 0 {
		return nil, fmt.Errorf("no results found for coordinates: %f, %f", lat, lng)
	}

	// Parse first result (most relevant)
	return s.parseGeocodingResult(apiResponse.Results[0])
}

// parseGeocodingResult converts a Google API result to our internal format
func (s *geocodingServiceImpl) parseGeocodingResult(result geocodingAPIResult) (*GeocodingResult, error) {
	// Extract address components
	components := make(map[string]string)
	for _, component := range result.AddressComponents {
		// Map common component types
		if len(component.Types) > 0 {
			componentType := component.Types[0]
			switch componentType {
			case "street_number":
				components["street_number"] = component.LongName
			case "route":
				components["street"] = component.LongName
			case "locality":
				components["city"] = component.LongName
			case "administrative_area_level_1":
				components["state"] = component.ShortName
			case "country":
				components["country"] = component.LongName
				components["country_code"] = component.ShortName
			case "postal_code":
				components["postal_code"] = component.LongName
			}
		}
	}

	return &GeocodingResult{
		Latitude:          result.Geometry.Location.Lat,
		Longitude:         result.Geometry.Location.Lng,
		FormattedAddress:  result.FormattedAddress,
		PlaceID:           result.PlaceID,
		AddressComponents: components,
	}, nil
}

// handleAPIError converts Google API error status to descriptive error
func (s *geocodingServiceImpl) handleAPIError(status, errorMessage string) error {
	switch status {
	case "ZERO_RESULTS":
		return fmt.Errorf("no results found for the given address")
	case "OVER_QUERY_LIMIT":
		return fmt.Errorf("geocoding API quota exceeded, please try again later")
	case "REQUEST_DENIED":
		if errorMessage != "" {
			return fmt.Errorf("geocoding request denied: %s", errorMessage)
		}
		return fmt.Errorf("geocoding request denied, check API key configuration")
	case "INVALID_REQUEST":
		return fmt.Errorf("invalid geocoding request: %s", errorMessage)
	case "UNKNOWN_ERROR":
		return fmt.Errorf("geocoding service temporarily unavailable, please try again")
	default:
		return fmt.Errorf("geocoding failed with status: %s", status)
	}
}

// Google Geocoding API response structures
type geocodingAPIResponse struct {
	Results      []geocodingAPIResult `json:"results"`
	Status       string               `json:"status"`
	ErrorMessage string               `json:"error_message,omitempty"`
}

type geocodingAPIResult struct {
	FormattedAddress  string                   `json:"formatted_address"`
	PlaceID           string                   `json:"place_id"`
	Geometry          geocodingGeometry        `json:"geometry"`
	AddressComponents []geocodingAddressComponent `json:"address_components"`
}

type geocodingGeometry struct {
	Location geocodingLocation `json:"location"`
}

type geocodingLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type geocodingAddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}
