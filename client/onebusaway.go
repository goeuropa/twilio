package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"oba-twilio/models"
)

type OneBusAwayClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
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
	endpoint := fmt.Sprintf("%s/api/where/stops-for-location.json", c.BaseURL)
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("key", c.APIKey)
	q.Add("lat", "47.6062")
	q.Add("lon", "-122.3321")
	q.Add("radius", "50000")
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var stopData models.StopData
	if err := json.NewDecoder(resp.Body).Decode(&stopData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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