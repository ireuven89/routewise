package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockHTTPClient is a mock HTTP client for testing
type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// mockRoundTripper implements http.RoundTripper for mocking HTTP responses
type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestGeocodeAddress_Success(t *testing.T) {
	tests := []struct {
		name             string
		address          string
		mockResponse     string
		expectedLat      float64
		expectedLng      float64
		expectedPlaceID  string
		expectedAddress  string
		expectedCity     string
		expectedCountry  string
	}{
		{
			name:    "successful geocoding with full address",
			address: "123 Main St, Tel Aviv, Israel",
			mockResponse: `{
				"results": [
					{
						"formatted_address": "123 Main St, Tel Aviv-Yafo, Israel",
						"place_id": "ChIJ1234567890",
						"geometry": {
							"location": {
								"lat": 32.0853,
								"lng": 34.7818
							}
						},
						"address_components": [
							{
								"long_name": "123",
								"short_name": "123",
								"types": ["street_number"]
							},
							{
								"long_name": "Main Street",
								"short_name": "Main St",
								"types": ["route"]
							},
							{
								"long_name": "Tel Aviv-Yafo",
								"short_name": "Tel Aviv",
								"types": ["locality"]
							},
							{
								"long_name": "Israel",
								"short_name": "IL",
								"types": ["country"]
							},
							{
								"long_name": "12345",
								"short_name": "12345",
								"types": ["postal_code"]
							}
						]
					}
				],
				"status": "OK"
			}`,
			expectedLat:     32.0853,
			expectedLng:     34.7818,
			expectedPlaceID: "ChIJ1234567890",
			expectedAddress: "123 Main St, Tel Aviv-Yafo, Israel",
			expectedCity:    "Tel Aviv-Yafo",
			expectedCountry: "Israel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := &http.Client{
				Transport: &mockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(tt.mockResponse)),
						}, nil
					},
				},
			}

			// Create service with mock client
			service := &geocodingServiceImpl{
				apiKey:     "test-api-key",
				httpClient: mockClient,
			}

			// Test geocoding
			result, err := service.GeocodeAddress(tt.address)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if result.Latitude != tt.expectedLat {
				t.Errorf("expected latitude %f, got %f", tt.expectedLat, result.Latitude)
			}
			if result.Longitude != tt.expectedLng {
				t.Errorf("expected longitude %f, got %f", tt.expectedLng, result.Longitude)
			}
			if result.PlaceID != tt.expectedPlaceID {
				t.Errorf("expected place_id %s, got %s", tt.expectedPlaceID, result.PlaceID)
			}
			if result.FormattedAddress != tt.expectedAddress {
				t.Errorf("expected formatted_address %s, got %s", tt.expectedAddress, result.FormattedAddress)
			}
			if result.AddressComponents["city"] != tt.expectedCity {
				t.Errorf("expected city %s, got %s", tt.expectedCity, result.AddressComponents["city"])
			}
			if result.AddressComponents["country"] != tt.expectedCountry {
				t.Errorf("expected country %s, got %s", tt.expectedCountry, result.AddressComponents["country"])
			}
		})
	}
}

func TestGeocodeAddress_EmptyAddress(t *testing.T) {
	service := NewGeocodingService("test-api-key")

	_, err := service.GeocodeAddress("")
	if err == nil {
		t.Fatal("expected error for empty address, got nil")
	}
	if err.Error() != "address cannot be empty" {
		t.Errorf("expected 'address cannot be empty', got '%s'", err.Error())
	}
}

func TestGeocodeAddress_APIErrors(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		errorMessage string
		expectedErr  string
	}{
		{
			name:        "ZERO_RESULTS",
			status:      "ZERO_RESULTS",
			expectedErr: "no results found for the given address",
		},
		{
			name:        "OVER_QUERY_LIMIT",
			status:      "OVER_QUERY_LIMIT",
			expectedErr: "geocoding API quota exceeded, please try again later",
		},
		{
			name:        "REQUEST_DENIED without message",
			status:      "REQUEST_DENIED",
			expectedErr: "geocoding request denied, check API key configuration",
		},
		{
			name:         "REQUEST_DENIED with message",
			status:       "REQUEST_DENIED",
			errorMessage: "API key is invalid",
			expectedErr:  "geocoding request denied: API key is invalid",
		},
		{
			name:         "INVALID_REQUEST",
			status:       "INVALID_REQUEST",
			errorMessage: "Missing address parameter",
			expectedErr:  "invalid geocoding request: Missing address parameter",
		},
		{
			name:        "UNKNOWN_ERROR",
			status:      "UNKNOWN_ERROR",
			expectedErr: "geocoding service temporarily unavailable, please try again",
		},
		{
			name:        "OTHER_ERROR",
			status:      "SOME_OTHER_STATUS",
			expectedErr: "geocoding failed with status: SOME_OTHER_STATUS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResponse := fmt.Sprintf(`{
				"results": [],
				"status": "%s"`, tt.status)
			if tt.errorMessage != "" {
				mockResponse += fmt.Sprintf(`,
				"error_message": "%s"`, tt.errorMessage)
			}
			mockResponse += `}`

			mockClient := &http.Client{
				Transport: &mockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(mockResponse)),
						}, nil
					},
				},
			}

			service := &geocodingServiceImpl{
				apiKey:     "test-api-key",
				httpClient: mockClient,
			}

			_, err := service.GeocodeAddress("123 Main St")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("expected error '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestGeocodeAddress_HTTPError(t *testing.T) {
	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("connection timeout")
			},
		},
	}

	service := &geocodingServiceImpl{
		apiKey:     "test-api-key",
		httpClient: mockClient,
	}

	_, err := service.GeocodeAddress("123 Main St")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to make geocoding request") {
		t.Errorf("expected error to contain 'failed to make geocoding request', got '%s'", err.Error())
	}
}

func TestGeocodeAddress_InvalidJSON(t *testing.T) {
	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("invalid json")),
				}, nil
			},
		},
	}

	service := &geocodingServiceImpl{
		apiKey:     "test-api-key",
		httpClient: mockClient,
	}

	_, err := service.GeocodeAddress("123 Main St")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse geocoding response") {
		t.Errorf("expected error to contain 'failed to parse geocoding response', got '%s'", err.Error())
	}
}

func TestReverseGeocode_Success(t *testing.T) {
	tests := []struct {
		name            string
		lat             float64
		lng             float64
		mockResponse    string
		expectedAddress string
		expectedPlaceID string
	}{
		{
			name: "successful reverse geocoding",
			lat:  32.0853,
			lng:  34.7818,
			mockResponse: `{
				"results": [
					{
						"formatted_address": "Dizengoff St 100, Tel Aviv-Yafo, Israel",
						"place_id": "ChIJReverseTest123",
						"geometry": {
							"location": {
								"lat": 32.0853,
								"lng": 34.7818
							}
						},
						"address_components": []
					}
				],
				"status": "OK"
			}`,
			expectedAddress: "Dizengoff St 100, Tel Aviv-Yafo, Israel",
			expectedPlaceID: "ChIJReverseTest123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &http.Client{
				Transport: &mockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(tt.mockResponse)),
						}, nil
					},
				},
			}

			service := &geocodingServiceImpl{
				apiKey:     "test-api-key",
				httpClient: mockClient,
			}

			result, err := service.ReverseGeocode(tt.lat, tt.lng)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if result.Latitude != tt.lat {
				t.Errorf("expected latitude %f, got %f", tt.lat, result.Latitude)
			}
			if result.Longitude != tt.lng {
				t.Errorf("expected longitude %f, got %f", tt.lng, result.Longitude)
			}
			if result.FormattedAddress != tt.expectedAddress {
				t.Errorf("expected address %s, got %s", tt.expectedAddress, result.FormattedAddress)
			}
			if result.PlaceID != tt.expectedPlaceID {
				t.Errorf("expected place_id %s, got %s", tt.expectedPlaceID, result.PlaceID)
			}
		})
	}
}

func TestReverseGeocode_InvalidCoordinates(t *testing.T) {
	tests := []struct {
		name        string
		lat         float64
		lng         float64
		expectedErr string
	}{
		{
			name:        "latitude too high",
			lat:         91.0,
			lng:         34.7818,
			expectedErr: "invalid latitude: 91.000000 (must be between -90 and 90)",
		},
		{
			name:        "latitude too low",
			lat:         -91.0,
			lng:         34.7818,
			expectedErr: "invalid latitude: -91.000000 (must be between -90 and 90)",
		},
		{
			name:        "longitude too high",
			lat:         32.0853,
			lng:         181.0,
			expectedErr: "invalid longitude: 181.000000 (must be between -180 and 180)",
		},
		{
			name:        "longitude too low",
			lat:         32.0853,
			lng:         -181.0,
			expectedErr: "invalid longitude: -181.000000 (must be between -180 and 180)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGeocodingService("test-api-key")

			_, err := service.ReverseGeocode(tt.lat, tt.lng)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("expected error '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestReverseGeocode_HTTPError(t *testing.T) {
	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("network error")
			},
		},
	}

	service := &geocodingServiceImpl{
		apiKey:     "test-api-key",
		httpClient: mockClient,
	}

	_, err := service.ReverseGeocode(32.0853, 34.7818)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to make reverse geocoding request") {
		t.Errorf("expected error to contain 'failed to make reverse geocoding request', got '%s'", err.Error())
	}
}

func TestParseGeocodingResult_AddressComponents(t *testing.T) {
	tests := []struct {
		name       string
		apiResult  geocodingAPIResult
		expectCity string
		expectZip  string
	}{
		{
			name: "parse all address components",
			apiResult: geocodingAPIResult{
				FormattedAddress: "123 Main St, Tel Aviv, Israel",
				PlaceID:          "ChIJ123",
				Geometry: geocodingGeometry{
					Location: geocodingLocation{
						Lat: 32.0853,
						Lng: 34.7818,
					},
				},
				AddressComponents: []geocodingAddressComponent{
					{
						LongName:  "123",
						ShortName: "123",
						Types:     []string{"street_number"},
					},
					{
						LongName:  "Main Street",
						ShortName: "Main St",
						Types:     []string{"route"},
					},
					{
						LongName:  "Tel Aviv-Yafo",
						ShortName: "Tel Aviv",
						Types:     []string{"locality"},
					},
					{
						LongName:  "Tel Aviv District",
						ShortName: "TA",
						Types:     []string{"administrative_area_level_1"},
					},
					{
						LongName:  "Israel",
						ShortName: "IL",
						Types:     []string{"country"},
					},
					{
						LongName:  "12345",
						ShortName: "12345",
						Types:     []string{"postal_code"},
					},
				},
			},
			expectCity: "Tel Aviv-Yafo",
			expectZip:  "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &geocodingServiceImpl{
				apiKey: "test-api-key",
			}

			result, err := service.parseGeocodingResult(tt.apiResult)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if result.AddressComponents["city"] != tt.expectCity {
				t.Errorf("expected city %s, got %s", tt.expectCity, result.AddressComponents["city"])
			}
			if result.AddressComponents["postal_code"] != tt.expectZip {
				t.Errorf("expected postal_code %s, got %s", tt.expectZip, result.AddressComponents["postal_code"])
			}
			if result.AddressComponents["street_number"] != "123" {
				t.Errorf("expected street_number 123, got %s", result.AddressComponents["street_number"])
			}
			if result.AddressComponents["street"] != "Main Street" {
				t.Errorf("expected street 'Main Street', got %s", result.AddressComponents["street"])
			}
			if result.AddressComponents["state"] != "TA" {
				t.Errorf("expected state TA, got %s", result.AddressComponents["state"])
			}
			if result.AddressComponents["country"] != "Israel" {
				t.Errorf("expected country Israel, got %s", result.AddressComponents["country"])
			}
			if result.AddressComponents["country_code"] != "IL" {
				t.Errorf("expected country_code IL, got %s", result.AddressComponents["country_code"])
			}
		})
	}
}
