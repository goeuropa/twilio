package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"oba-twilio/models"
)

func TestInitializeCoverage_Success(t *testing.T) {
	mockResponse := models.AgenciesWithCoverageResponse{
		Data: struct {
			LimitExceeded bool `json:"limitExceeded"`
			List []struct {
				AgencyID string  `json:"agencyId"`
				Lat      float64 `json:"lat"`
				LatSpan  float64 `json:"latSpan"`
				Lon      float64 `json:"lon"`
				LonSpan  float64 `json:"lonSpan"`
			} `json:"list"`
		}{
			LimitExceeded: false,
			List: []struct {
				AgencyID string  `json:"agencyId"`
				Lat      float64 `json:"lat"`
				LatSpan  float64 `json:"latSpan"`
				Lon      float64 `json:"lon"`
				LonSpan  float64 `json:"lonSpan"`
			}{
				{
					AgencyID: "metro",
					Lat:      47.6062,
					LatSpan:  0.5,
					Lon:      -122.3321,
					LonSpan:  0.8,
				},
				{
					AgencyID: "soundtransit",
					Lat:      47.5,
					LatSpan:  0.3,
					Lon:      -122.2,
					LonSpan:  0.4,
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "agencies-with-coverage")
		assert.Equal(t, "test-key", r.URL.Query().Get("key"))
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")
	
	err := client.InitializeCoverage()
	assert.NoError(t, err)
	
	coverage := client.GetCoverageArea()
	assert.NotNil(t, coverage)
	
	// Center should be the midpoint of all agencies
	// Agency 1: lat 47.6062±0.25, lon -122.3321±0.4 -> bounds: lat[47.3562, 47.8562], lon[-122.7321, -121.9321]
	// Agency 2: lat 47.5±0.15, lon -122.2±0.2 -> bounds: lat[47.35, 47.65], lon[-122.4, -122.0]
	// Combined bounds: lat[47.35, 47.8562], lon[-122.7321, -121.9321]
	// Center: lat=(47.35+47.8562)/2=47.6031, lon=(-122.7321+-121.9321)/2=-122.3321
	assert.InDelta(t, 47.6031, coverage.CenterLat, 0.01) 
	assert.InDelta(t, -122.3321, coverage.CenterLon, 0.01) 
	assert.Greater(t, coverage.Radius, 5000.0) 
	assert.Less(t, coverage.Radius, 100000.0) 
}

func TestInitializeCoverage_SingleAgency(t *testing.T) {
	mockResponse := models.AgenciesWithCoverageResponse{
		Data: struct {
			LimitExceeded bool `json:"limitExceeded"`
			List []struct {
				AgencyID string  `json:"agencyId"`
				Lat      float64 `json:"lat"`
				LatSpan  float64 `json:"latSpan"`
				Lon      float64 `json:"lon"`
				LonSpan  float64 `json:"lonSpan"`
			} `json:"list"`
		}{
			List: []struct {
				AgencyID string  `json:"agencyId"`
				Lat      float64 `json:"lat"`
				LatSpan  float64 `json:"latSpan"`
				Lon      float64 `json:"lon"`
				LonSpan  float64 `json:"lonSpan"`
			}{
				{
					AgencyID: "unitrans",
					Lat:      38.5553,
					LatSpan:  0.0356,
					Lon:      -121.7360,
					LonSpan:  0.1050,
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")
	
	err := client.InitializeCoverage()
	assert.NoError(t, err)
	
	coverage := client.GetCoverageArea()
	assert.NotNil(t, coverage)
	
	assert.InDelta(t, 38.5553, coverage.CenterLat, 0.01)
	assert.InDelta(t, -121.7360, coverage.CenterLon, 0.01)
	assert.GreaterOrEqual(t, coverage.Radius, 5000.0) 
}

func TestInitializeCoverage_NoAgencies(t *testing.T) {
	mockResponse := models.AgenciesWithCoverageResponse{
		Data: struct {
			LimitExceeded bool `json:"limitExceeded"`
			List []struct {
				AgencyID string  `json:"agencyId"`
				Lat      float64 `json:"lat"`
				LatSpan  float64 `json:"latSpan"`
				Lon      float64 `json:"lon"`
				LonSpan  float64 `json:"lonSpan"`
			} `json:"list"`
		}{
			List: []struct {
				AgencyID string  `json:"agencyId"`
				Lat      float64 `json:"lat"`
				LatSpan  float64 `json:"latSpan"`
				Lon      float64 `json:"lon"`
				LonSpan  float64 `json:"lonSpan"`
			}{},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")
	
	err := client.InitializeCoverage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no coverage areas found")
}

func TestInitializeCoverage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")
	
	err := client.InitializeCoverage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestCalculateRadius(t *testing.T) {
	client := NewOneBusAwayClient("test", "test")
	
	tests := []struct {
		name      string
		latSpan   float64
		lonSpan   float64
		centerLat float64
		minRadius float64
		maxRadius float64
	}{
		{
			name:      "Small area",
			latSpan:   0.01,
			lonSpan:   0.01,
			centerLat: 47.6,
			minRadius: 5000,
			maxRadius: 10000,
		},
		{
			name:      "Large area",
			latSpan:   5.0,
			lonSpan:   5.0,
			centerLat: 47.6,
			minRadius: 100000,
			maxRadius: 100000,
		},
		{
			name:      "Equatorial area",
			latSpan:   1.0,
			lonSpan:   1.0,
			centerLat: 0.0,
			minRadius: 50000,
			maxRadius: 120000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			radius := client.calculateRadius(tt.latSpan, tt.lonSpan, tt.centerLat)
			assert.GreaterOrEqual(t, radius, tt.minRadius)
			assert.LessOrEqual(t, radius, tt.maxRadius)
		})
	}
}

func TestSearchStops_WithCoverage(t *testing.T) {
	mockStopData := models.StopData{
		Data: struct {
			List []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Lat  float64 `json:"lat"`
				Lon  float64 `json:"lon"`
			} `json:"list"`
		}{
			List: []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Lat  float64 `json:"lat"`
				Lon  float64 `json:"lon"`
			}{
				{
					ID:   "1_12345",
					Name: "Test Stop",
					Lat:  47.6062,
					Lon:  -122.3321,
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/where/agencies-with-coverage.json" {
			mockCoverage := models.AgenciesWithCoverageResponse{
				Data: struct {
					LimitExceeded bool `json:"limitExceeded"`
					List []struct {
						AgencyID string  `json:"agencyId"`
						Lat      float64 `json:"lat"`
						LatSpan  float64 `json:"latSpan"`
						Lon      float64 `json:"lon"`
						LonSpan  float64 `json:"lonSpan"`
					} `json:"list"`
				}{
					List: []struct {
						AgencyID string  `json:"agencyId"`
						Lat      float64 `json:"lat"`
						LatSpan  float64 `json:"latSpan"`
						Lon      float64 `json:"lon"`
						LonSpan  float64 `json:"lonSpan"`
					}{
						{
							AgencyID: "test",
							Lat:      47.6062,
							LatSpan:  0.5,
							Lon:      -122.3321,
							LonSpan:  0.8,
						},
					},
				},
				Code: 200,
				Text: "OK",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockCoverage)
		} else if r.URL.Path == "/api/where/stops-for-location.json" {
			assert.Equal(t, "test search", r.URL.Query().Get("query"))
			assert.NotEmpty(t, r.URL.Query().Get("lat"))
			assert.NotEmpty(t, r.URL.Query().Get("lon"))
			assert.NotEmpty(t, r.URL.Query().Get("radius"))
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockStopData)
		}
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")
	
	err := client.InitializeCoverage()
	assert.NoError(t, err)
	
	stops, err := client.SearchStops("test search")
	assert.NoError(t, err)
	assert.Len(t, stops, 1)
	assert.Equal(t, "1_12345", stops[0].ID)
	assert.Equal(t, "Test Stop", stops[0].Name)
}

func TestSearchStops_WithoutCoverage(t *testing.T) {
	client := NewOneBusAwayClient("test", "test-key")
	
	_, err := client.SearchStops("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "coverage area not initialized")
}

func TestCalculateCoverageArea_EmptyAgencies(t *testing.T) {
	client := NewOneBusAwayClient("test", "test")
	
	agencies := []struct {
		AgencyID string  `json:"agencyId"`
		Lat      float64 `json:"lat"`
		LatSpan  float64 `json:"latSpan"`
		Lon      float64 `json:"lon"`
		LonSpan  float64 `json:"lonSpan"`
	}{}
	
	coverage := client.calculateCoverageArea(agencies)
	
	assert.Equal(t, 47.6062, coverage.CenterLat)
	assert.Equal(t, -122.3321, coverage.CenterLon)
	assert.Equal(t, 25000.0, coverage.Radius)
}