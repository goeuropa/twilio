package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"oba-twilio/client"
	"oba-twilio/handlers"
	"oba-twilio/models"
)

type MockOneBusAwayClient struct {
	mock.Mock
}

func (m *MockOneBusAwayClient) GetArrivalsAndDepartures(stopID string) (*models.OneBusAwayResponse, error) {
	args := m.Called(stopID)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	if response, ok := resp.(*models.OneBusAwayResponse); ok {
		return response, args.Error(1)
	}
	return nil, fmt.Errorf("mock returned invalid type for GetArrivalsAndDepartures")
}

func (m *MockOneBusAwayClient) ProcessArrivals(resp *models.OneBusAwayResponse) []models.Arrival {
	args := m.Called(resp)
	result := args.Get(0)
	if result == nil {
		return nil
	}
	if arrivals, ok := result.([]models.Arrival); ok {
		return arrivals
	}
	return nil
}

func (m *MockOneBusAwayClient) SearchStops(query string) ([]models.Stop, error) {
	args := m.Called(query)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	if stops, ok := result.([]models.Stop); ok {
		return stops, args.Error(1)
	}
	return nil, fmt.Errorf("mock returned invalid type for SearchStops")
}

func (m *MockOneBusAwayClient) InitializeCoverage() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockOneBusAwayClient) GetCoverageArea() *models.CoverageArea {
	args := m.Called()
	result := args.Get(0)
	if result == nil {
		return nil
	}
	if coverage, ok := result.(*models.CoverageArea); ok {
		return coverage
	}
	return nil
}

func (m *MockOneBusAwayClient) FindAllMatchingStops(stopID string) ([]models.StopOption, error) {
	args := m.Called(stopID)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	if stops, ok := result.([]models.StopOption); ok {
		return stops, args.Error(1)
	}
	return nil, fmt.Errorf("mock returned invalid type for FindAllMatchingStops")
}

func (m *MockOneBusAwayClient) GetStopInfo(fullStopID string) (*models.StopOption, error) {
	args := m.Called(fullStopID)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	if stopOption, ok := result.(*models.StopOption); ok {
		return stopOption, args.Error(1)
	}
	return nil, fmt.Errorf("mock returned invalid type for GetStopInfo")
}

func setupTestRouter() (*gin.Engine, *MockOneBusAwayClient) {
	gin.SetMode(gin.TestMode)

	mockClient := &MockOneBusAwayClient{}
	smsHandler := handlers.NewSMSHandler(mockClient)
	voiceHandler := handlers.NewVoiceHandler(mockClient)

	r := gin.New()
	r.POST("/sms", smsHandler.HandleSMS)
	r.POST("/voice", voiceHandler.HandleVoiceStart)
	r.POST("/voice/input", voiceHandler.HandleVoiceInput)

	return r, mockClient
}

func TestHealthEndpoint(t *testing.T) {
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestSMSHandler_ValidStopID(t *testing.T) {
	r, mockClient := setupTestRouter()

	mockResponse := &models.OneBusAwayResponse{
		Data: struct {
			Entry struct {
				ArrivalsAndDepartures []struct {
					RouteShortName       string `json:"routeShortName"`
					TripHeadsign         string `json:"tripHeadsign"`
					PredictedArrivalTime int64  `json:"predictedArrivalTime"`
					ScheduledArrivalTime int64  `json:"scheduledArrivalTime"`
					Status               string `json:"status"`
				} `json:"arrivalsAndDepartures"`
				Stop struct {
					ID        string  `json:"id"`
					Name      string  `json:"name"`
					Direction string  `json:"direction"`
					Lat       float64 `json:"lat"`
					Lon       float64 `json:"lon"`
				} `json:"stop"`
			} `json:"entry"`
		}{
			Entry: struct {
				ArrivalsAndDepartures []struct {
					RouteShortName       string `json:"routeShortName"`
					TripHeadsign         string `json:"tripHeadsign"`
					PredictedArrivalTime int64  `json:"predictedArrivalTime"`
					ScheduledArrivalTime int64  `json:"scheduledArrivalTime"`
					Status               string `json:"status"`
				} `json:"arrivalsAndDepartures"`
				Stop struct {
					ID        string  `json:"id"`
					Name      string  `json:"name"`
					Direction string  `json:"direction"`
					Lat       float64 `json:"lat"`
					Lon       float64 `json:"lon"`
				} `json:"stop"`
			}{
				Stop: struct {
					ID        string  `json:"id"`
					Name      string  `json:"name"`
					Direction string  `json:"direction"`
					Lat       float64 `json:"lat"`
					Lon       float64 `json:"lon"`
				}{
					Name: "Test Stop",
				},
			},
		},
		Code: 200,
	}

	mockArrivals := []models.Arrival{
		{
			RouteShortName:      "8",
			TripHeadsign:        "Seattle Center",
			MinutesUntilArrival: 3,
		},
	}

	mockStopOptions := []models.StopOption{
		{
			FullStopID:  "1_75403",
			AgencyName:  "King County Metro",
			StopName:    "Test Stop",
			DisplayText: "King County Metro: Test Stop",
		},
	}

	mockClient.On("FindAllMatchingStops", "75403").Return(mockStopOptions, nil)
	mockClient.On("GetArrivalsAndDepartures", "1_75403").Return(mockResponse, nil)
	mockClient.On("ProcessArrivals", mockResponse).Return(mockArrivals)

	form := url.Values{}
	form.Set("From", "+14444444444")
	form.Set("To", "+15555555555")
	form.Set("Body", "75403")
	form.Set("MessageSid", "test-message-sid")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sms", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Stop")
	assert.Contains(t, w.Body.String(), "Route 8")
	assert.Contains(t, w.Body.String(), "Seattle Center")
	mockClient.AssertExpectations(t)
}

func TestSMSHandler_InvalidStopID(t *testing.T) {
	r, _ := setupTestRouter()

	form := url.Values{}
	form.Set("From", "+14444444444")
	form.Set("To", "+15555555555")
	form.Set("Body", "invalid")
	form.Set("MessageSid", "test-message-sid")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sms", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "valid stop ID")
}

func TestVoiceHandler_Start(t *testing.T) {
	r, _ := setupTestRouter()

	form := url.Values{}
	form.Set("From", "+14444444444")
	form.Set("To", "+15555555555")
	form.Set("CallSid", "test-call-sid")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/voice", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Welcome to OneBusAway")
	assert.Contains(t, w.Body.String(), "<Gather")
}

func TestVoiceHandler_Input(t *testing.T) {
	r, mockClient := setupTestRouter()

	mockResponse := &models.OneBusAwayResponse{
		Data: struct {
			Entry struct {
				ArrivalsAndDepartures []struct {
					RouteShortName       string `json:"routeShortName"`
					TripHeadsign         string `json:"tripHeadsign"`
					PredictedArrivalTime int64  `json:"predictedArrivalTime"`
					ScheduledArrivalTime int64  `json:"scheduledArrivalTime"`
					Status               string `json:"status"`
				} `json:"arrivalsAndDepartures"`
				Stop struct {
					ID        string  `json:"id"`
					Name      string  `json:"name"`
					Direction string  `json:"direction"`
					Lat       float64 `json:"lat"`
					Lon       float64 `json:"lon"`
				} `json:"stop"`
			} `json:"entry"`
		}{
			Entry: struct {
				ArrivalsAndDepartures []struct {
					RouteShortName       string `json:"routeShortName"`
					TripHeadsign         string `json:"tripHeadsign"`
					PredictedArrivalTime int64  `json:"predictedArrivalTime"`
					ScheduledArrivalTime int64  `json:"scheduledArrivalTime"`
					Status               string `json:"status"`
				} `json:"arrivalsAndDepartures"`
				Stop struct {
					ID        string  `json:"id"`
					Name      string  `json:"name"`
					Direction string  `json:"direction"`
					Lat       float64 `json:"lat"`
					Lon       float64 `json:"lon"`
				} `json:"stop"`
			}{
				Stop: struct {
					ID        string  `json:"id"`
					Name      string  `json:"name"`
					Direction string  `json:"direction"`
					Lat       float64 `json:"lat"`
					Lon       float64 `json:"lon"`
				}{
					Name: "Voice Test Stop",
				},
			},
		},
		Code: 200,
	}

	mockArrivals := []models.Arrival{
		{
			RouteShortName:      "43",
			TripHeadsign:        "Capitol Hill",
			MinutesUntilArrival: 5,
		},
	}

	mockClient.On("GetArrivalsAndDepartures", "12345").Return(mockResponse, nil)
	mockClient.On("ProcessArrivals", mockResponse).Return(mockArrivals)

	form := url.Values{}
	form.Set("From", "+14444444444")
	form.Set("To", "+15555555555")
	form.Set("CallSid", "test-call-sid")
	form.Set("Digits", "12345")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/voice/input", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Voice Test Stop")
	assert.Contains(t, w.Body.String(), "Route 43")
	assert.Contains(t, w.Body.String(), "Capitol Hill")
	mockClient.AssertExpectations(t)
}

func TestOneBusAwayClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := client.NewOneBusAwayClient("https://api.pugetsound.onebusaway.org", "test")

	resp, err := client.GetArrivalsAndDepartures("1_75403")
	if err != nil {
		t.Logf("Integration test failed (this is expected if the API is down): %v", err)
		return
	}

	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.Code)
}

func TestJSONResponse(t *testing.T) {
	type TestResponse struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}

	expected := TestResponse{
		Message: "OneBusAway Twilio Integration",
		Status:  "healthy",
	}

	jsonData, err := json.Marshal(expected)
	assert.NoError(t, err)

	var actual TestResponse
	err = json.Unmarshal(jsonData, &actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestTwiMLGeneration(t *testing.T) {
	r, _ := setupTestRouter()

	form := url.Values{}
	form.Set("From", "+14444444444")
	form.Set("To", "+15555555555")
	form.Set("Body", "abc")
	form.Set("MessageSid", "test-message-sid")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sms", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "<?xml version=\"1.0\"")
	assert.Contains(t, body, "<Response>")
	assert.Contains(t, body, "</Response>")
}
