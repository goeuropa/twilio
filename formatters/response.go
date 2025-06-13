package formatters

import (
	"encoding/xml"
	"fmt"
	"strings"

	"oba-twilio/models"
)

func FormatSMSResponse(arrivals []models.Arrival, stopName string) string {
	if len(arrivals) == 0 {
		return "No upcoming arrivals found for this stop."
	}

	var response strings.Builder
	
	if stopName != "" {
		response.WriteString(fmt.Sprintf("Stop: %s\n", stopName))
	}

	for i, arrival := range arrivals {
		if i >= 3 {
			break
		}
		
		timeText := formatArrivalTime(arrival.MinutesUntilArrival)
		response.WriteString(fmt.Sprintf("Route %s to %s: %s\n", 
			arrival.RouteShortName, 
			arrival.TripHeadsign, 
			timeText))
	}

	return strings.TrimSpace(response.String())
}

func FormatVoiceResponse(arrivals []models.Arrival, stopName string) string {
	if len(arrivals) == 0 {
		return "No upcoming arrivals found for this stop."
	}

	var response strings.Builder
	
	if stopName != "" {
		response.WriteString(fmt.Sprintf("Arrivals for %s. ", stopName))
	}

	for i, arrival := range arrivals {
		if i >= 3 {
			break
		}
		
		timeText := formatArrivalTimeVoice(arrival.MinutesUntilArrival)
		response.WriteString(fmt.Sprintf("Route %s to %s %s. ", 
			arrival.RouteShortName, 
			arrival.TripHeadsign, 
			timeText))
	}

	return response.String()
}

func GenerateTwiMLSMS(message string) (string, error) {
	twiml := models.TwiMLResponse{
		Message: message,
	}

	output, err := xml.MarshalIndent(twiml, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(output), nil
}

func GenerateTwiMLVoice(message string) (string, error) {
	twiml := models.TwiMLResponse{
		Say: message,
	}

	output, err := xml.MarshalIndent(twiml, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(output), nil
}

func GenerateTwiMLGather(prompt string, action string, numDigits int) (string, error) {
	twiml := models.TwiMLResponse{
		Gather: &models.Gather{
			NumDigits: numDigits,
			Action:    action,
			Method:    "POST",
			Say:       prompt,
		},
	}

	output, err := xml.MarshalIndent(twiml, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(output), nil
}

func formatArrivalTime(minutes int) string {
	if minutes <= 0 {
		return "Now"
	} else if minutes == 1 {
		return "1 min"
	} else {
		return fmt.Sprintf("%d min", minutes)
	}
}

func formatArrivalTimeVoice(minutes int) string {
	if minutes <= 0 {
		return "arriving now"
	} else if minutes == 1 {
		return "in 1 minute"
	} else {
		return fmt.Sprintf("in %d minutes", minutes)
	}
}

func ExtractStopID(message string) string {
	message = strings.TrimSpace(message)
	
	if message == "" {
		return ""
	}
	
	fields := strings.Fields(message)
	if len(fields) == 0 {
		return ""
	}
	
	stopID := fields[0]
	
	if len(stopID) >= 3 && len(stopID) <= 10 {
		for _, char := range stopID {
			if char < '0' || char > '9' {
				return ""
			}
		}
		return stopID
	}
	
	return ""
}