package client

import "oba-twilio/models"

type OneBusAwayClientInterface interface {
	GetArrivalsAndDepartures(stopID string) (*models.OneBusAwayResponse, error)
	ProcessArrivals(resp *models.OneBusAwayResponse) []models.Arrival
	SearchStops(query string) ([]models.Stop, error)
	InitializeCoverage() error
	GetCoverageArea() *models.CoverageArea
}