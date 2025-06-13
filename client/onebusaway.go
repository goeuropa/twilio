package client

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"oba-twilio/models"
)

const (
	// apiTimeoutSeconds defines the maximum time to wait for API operations
	// Set to 30 seconds to balance responsiveness with reliability for mobile users
	apiTimeoutSeconds = 30
	
	// maxConcurrentRequests limits parallel API calls to prevent overwhelming the server
	// Set to 10 to balance performance with server resource conservation
	maxConcurrentRequests = 10
)

type OneBusAwayClient struct {
	BaseURL      string
	APIKey       string
	Client       *http.Client
	coverageArea *models.CoverageArea
}

func NewOneBusAwayClient(baseURL, apiKey string) *OneBusAwayClient {
	return &OneBusAwayClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *OneBusAwayClient) InitializeCoverage() error {
	endpoint := fmt.Sprintf("%s/api/where/agencies-with-coverage.json", c.BaseURL)
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("key", c.APIKey)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var coverageResp models.AgenciesWithCoverageResponse
	if err := json.NewDecoder(resp.Body).Decode(&coverageResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if coverageResp.Code != 200 {
		return fmt.Errorf("API error: %s (code %d)", coverageResp.Text, coverageResp.Code)
	}

	if len(coverageResp.Data.List) == 0 {
		return fmt.Errorf("no coverage areas found")
	}

	c.coverageArea = c.calculateCoverageArea(coverageResp.Data.List)
	return nil
}

func (c *OneBusAwayClient) GetCoverageArea() *models.CoverageArea {
	return c.coverageArea
}

func (c *OneBusAwayClient) calculateCoverageArea(agencies []struct {
	AgencyID string  `json:"agencyId"`
	Lat      float64 `json:"lat"`
	LatSpan  float64 `json:"latSpan"`
	Lon      float64 `json:"lon"`
	LonSpan  float64 `json:"lonSpan"`
}) *models.CoverageArea {
	if len(agencies) == 0 {
		return &models.CoverageArea{
			CenterLat: 47.6062,
			CenterLon: -122.3321,
			Radius:    25000,
		}
	}

	var minLat, maxLat, minLon, maxLon float64
	first := true

	for _, agency := range agencies {
		agencyMinLat := agency.Lat - agency.LatSpan/2
		agencyMaxLat := agency.Lat + agency.LatSpan/2
		agencyMinLon := agency.Lon - agency.LonSpan/2
		agencyMaxLon := agency.Lon + agency.LonSpan/2

		if first {
			minLat, maxLat = agencyMinLat, agencyMaxLat
			minLon, maxLon = agencyMinLon, agencyMaxLon
			first = false
		} else {
			if agencyMinLat < minLat {
				minLat = agencyMinLat
			}
			if agencyMaxLat > maxLat {
				maxLat = agencyMaxLat
			}
			if agencyMinLon < minLon {
				minLon = agencyMinLon
			}
			if agencyMaxLon > maxLon {
				maxLon = agencyMaxLon
			}
		}
	}

	centerLat := (minLat + maxLat) / 2
	centerLon := (minLon + maxLon) / 2

	latSpan := maxLat - minLat
	lonSpan := maxLon - minLon

	radius := c.calculateRadius(latSpan, lonSpan, centerLat)

	return &models.CoverageArea{
		CenterLat: centerLat,
		CenterLon: centerLon,
		Radius:    radius,
	}
}

func (c *OneBusAwayClient) calculateRadius(latSpan, lonSpan, centerLat float64) float64 {
	const earthRadiusMeters = 6371000

	latRadians := latSpan * math.Pi / 180
	lonRadians := lonSpan * math.Pi / 180
	centerLatRadians := centerLat * math.Pi / 180

	latDistanceMeters := latRadians * earthRadiusMeters
	lonDistanceMeters := lonRadians * earthRadiusMeters * math.Cos(centerLatRadians)

	maxDistance := math.Max(latDistanceMeters, lonDistanceMeters)

	radius := maxDistance / 2

	const minRadius = 5000
	const maxRadius = 100000

	if radius < minRadius {
		return minRadius
	}
	if radius > maxRadius {
		return maxRadius
	}

	return radius
}

func (c *OneBusAwayClient) GetArrivalsAndDepartures(stopID string) (*models.OneBusAwayResponse, error) {
	fullStopID, err := c.resolveStopID(stopID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stop ID: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/where/arrivals-and-departures-for-stop/%s.json", c.BaseURL, url.QueryEscape(fullStopID))
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("key", c.APIKey)
	q.Add("minutesBefore", "0")
	q.Add("minutesAfter", "30")
	req.URL.RawQuery = q.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var obaResp models.OneBusAwayResponse
	if err := json.NewDecoder(resp.Body).Decode(&obaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if obaResp.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", obaResp.Text, obaResp.Code)
	}

	return &obaResp, nil
}

func (c *OneBusAwayClient) resolveStopID(stopID string) (string, error) {
	stopID = strings.TrimSpace(stopID)
	if stopID == "" {
		return "", fmt.Errorf("stop ID cannot be empty")
	}

	if strings.Contains(stopID, "_") {
		return stopID, nil
	}

	agencies := []string{"1", "40", "29", "95", "97", "98", "3", "23"}
	
	for _, agency := range agencies {
		fullStopID := fmt.Sprintf("%s_%s", agency, stopID)
		if c.stopExists(fullStopID) {
			return fullStopID, nil
		}
	}

	return fmt.Sprintf("1_%s", stopID), nil
}

func (c *OneBusAwayClient) FindAllMatchingStops(stopID string) ([]models.StopOption, error) {
	stopID = strings.TrimSpace(stopID)
	if stopID == "" {
		return nil, fmt.Errorf("stop ID cannot be empty")
	}

	if strings.Contains(stopID, "_") {
		stopOption, err := c.GetStopInfo(stopID)
		if err != nil {
			return nil, err
		}
		if stopOption != nil {
			return []models.StopOption{*stopOption}, nil
		}
		return []models.StopOption{}, nil
	}

	agencies := []string{"1", "40", "29", "95", "97", "98", "3", "23"}
	
	// Use a channel to collect results and limit concurrency
	semaphore := make(chan struct{}, maxConcurrentRequests)
	resultChan := make(chan *models.StopOption, len(agencies))
	var wg sync.WaitGroup

	for _, agency := range agencies {
		wg.Add(1)
		go func(agencyID string) {
			defer wg.Done()
			
			// Acquire semaphore with panic recovery
			semaphore <- struct{}{}
			defer func() {
				if r := recover(); r != nil {
					<-semaphore
					panic(r)
				}
				<-semaphore
			}()
			
			fullStopID := fmt.Sprintf("%s_%s", agencyID, stopID)
			stopOption, err := c.GetStopInfo(fullStopID)
			if err == nil && stopOption != nil {
				resultChan <- stopOption
			} else {
				resultChan <- nil
			}
		}(agency)
	}

	// Close result channel when all goroutines complete or timeout
	go func() {
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		
		select {
		case <-done:
			close(resultChan)
		case <-time.After(apiTimeoutSeconds * time.Second):
			close(resultChan)
		}
	}()

	// Collect results
	var matchingStops []models.StopOption
	for stopOption := range resultChan {
		if stopOption != nil {
			matchingStops = append(matchingStops, *stopOption)
		}
	}

	return matchingStops, nil
}

func (c *OneBusAwayClient) GetStopInfo(fullStopID string) (*models.StopOption, error) {
	endpoint := fmt.Sprintf("%s/api/where/stop/%s.json", c.BaseURL, url.QueryEscape(fullStopID))
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("key", c.APIKey)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var stopResp struct {
		Data struct {
			Entry struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"entry"`
			References struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			} `json:"references"`
		} `json:"data"`
		Code int    `json:"code"`
		Text string `json:"text"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&stopResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if stopResp.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", stopResp.Text, stopResp.Code)
	}

	agencyName := c.getAgencyNameFromID(fullStopID, stopResp.Data.References.Agencies)
	
	return &models.StopOption{
		FullStopID:  fullStopID,
		AgencyName:  agencyName,
		StopName:    stopResp.Data.Entry.Name,
		DisplayText: fmt.Sprintf("%s: %s", agencyName, stopResp.Data.Entry.Name),
	}, nil
}

func (c *OneBusAwayClient) getAgencyNameFromID(stopID string, agencies []struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}) string {
	parts := strings.Split(stopID, "_")
	if len(parts) < 2 {
		return "Unknown"
	}
	
	agencyID := parts[0]
	
	for _, agency := range agencies {
		if agency.ID == agencyID {
			return agency.Name
		}
	}

	switch agencyID {
	case "1":
		return "King County Metro"
	case "40":
		return "Sound Transit"
	case "29":
		return "Pierce Transit"
	case "95":
		return "Community Transit"
	case "97":
		return "Kitsap Transit"
	case "98":
		return "Everett Transit"
	case "3":
		return "Washington State Ferries"
	case "23":
		return "Other Agency"
	default:
		return fmt.Sprintf("Agency %s", agencyID)
	}
}

func (c *OneBusAwayClient) stopExists(stopID string) bool {
	endpoint := fmt.Sprintf("%s/api/where/stop/%s.json", c.BaseURL, url.QueryEscape(stopID))
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false
	}

	q := req.URL.Query()
	q.Add("key", c.APIKey)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (c *OneBusAwayClient) SearchStops(query string) ([]models.Stop, error) {
	coverage := c.GetCoverageArea()
	if coverage == nil {
		return nil, fmt.Errorf("coverage area not initialized - call InitializeCoverage() first")
	}

	endpoint := fmt.Sprintf("%s/api/where/stops-for-location.json", c.BaseURL)
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("key", c.APIKey)
	q.Add("lat", fmt.Sprintf("%.6f", coverage.CenterLat))
	q.Add("lon", fmt.Sprintf("%.6f", coverage.CenterLon))
	q.Add("radius", fmt.Sprintf("%.0f", coverage.Radius))
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var stopData models.StopData
	if err := json.NewDecoder(resp.Body).Decode(&stopData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if stopData.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", stopData.Text, stopData.Code)
	}

	stops := make([]models.Stop, len(stopData.Data.List))
	for i, s := range stopData.Data.List {
		stops[i] = models.Stop{
			ID:        s.ID,
			Name:      s.Name,
			Latitude:  s.Lat,
			Longitude: s.Lon,
		}
	}

	return stops, nil
}

func (c *OneBusAwayClient) ProcessArrivals(obaResp *models.OneBusAwayResponse) []models.Arrival {
	arrivals := make([]models.Arrival, 0)
	now := time.Now().Unix() * 1000

	for _, ad := range obaResp.Data.Entry.ArrivalsAndDepartures {
		arrivalTime := ad.PredictedArrivalTime
		if arrivalTime == 0 {
			arrivalTime = ad.ScheduledArrivalTime
		}

		if arrivalTime <= now {
			continue
		}

		minutesUntil := int((arrivalTime - now) / (1000 * 60))
		if minutesUntil > 60 {
			continue
		}

		arrival := models.Arrival{
			RouteShortName:        ad.RouteShortName,
			TripHeadsign:         ad.TripHeadsign,
			PredictedArrivalTime: ad.PredictedArrivalTime,
			ScheduledArrivalTime: ad.ScheduledArrivalTime,
			MinutesUntilArrival:  minutesUntil,
			Status:               ad.Status,
		}

		arrivals = append(arrivals, arrival)
	}

	return arrivals
}