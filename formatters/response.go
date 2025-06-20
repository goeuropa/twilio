package formatters

import (
	"encoding/xml"
	"fmt"
	"strings"

	"oba-twilio/localization"
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

// RouteGroup represents arrivals grouped by route and headsign
type RouteGroup struct {
	RouteShortName string
	TripHeadsign   string
	ArrivalTimes   []int
}

// groupArrivalsByRoute groups arrivals by route short name and headsign
func groupArrivalsByRoute(arrivals []models.Arrival) []RouteGroup {
	groups := make(map[string]*RouteGroup)
	var groupOrder []string

	for _, arrival := range arrivals {
		key := arrival.RouteShortName + "|" + arrival.TripHeadsign
		if group, exists := groups[key]; exists {
			group.ArrivalTimes = append(group.ArrivalTimes, arrival.MinutesUntilArrival)
		} else {
			groups[key] = &RouteGroup{
				RouteShortName: arrival.RouteShortName,
				TripHeadsign:   arrival.TripHeadsign,
				ArrivalTimes:   []int{arrival.MinutesUntilArrival},
			}
			groupOrder = append(groupOrder, key)
		}
	}

	result := make([]RouteGroup, len(groupOrder))
	for i, key := range groupOrder {
		result[i] = *groups[key]
	}
	return result
}

func FormatVoiceResponse(arrivals []models.Arrival, stopName string, lm *localization.LocalizationManager, language string) string {
	if len(arrivals) == 0 {
		return lm.GetString("voice.arrival.no_arrivals", language)
	}

	var response strings.Builder

	if stopName != "" {
		response.WriteString(lm.GetString("voice.arrival.arrivals_for", language, stopName))
		response.WriteString(" ")
	}

	// Group arrivals by route
	routeGroups := groupArrivalsByRoute(arrivals)

	for _, group := range routeGroups {
		// Format "Route X to Y"
		routeText := lm.GetString("voice.arrival.route_to", language, group.RouteShortName, group.TripHeadsign)
		response.WriteString(routeText)
		response.WriteString(" ")

		// Format the arrival times
		for i, minutes := range group.ArrivalTimes {
			if i > 0 {
				response.WriteString(", ")
			}
			timeText := formatArrivalTimeVoiceLocalized(minutes, lm, language)
			response.WriteString(timeText)
		}
		response.WriteString(". ")
	}

	return response.String()
}

type TwiMLSMSResponse struct {
	XMLName string `xml:"Response"`
	Message string `xml:"Message"`
}

func GenerateTwiMLSMS(message string) (string, error) {
	twiml := TwiMLSMSResponse{
		Message: message,
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

func formatArrivalTimeVoiceLocalized(minutes int, lm *localization.LocalizationManager, language string) string {
	if minutes <= 0 {
		return lm.GetString("voice.arrival.arriving_now", language)
	} else if minutes == 1 {
		return lm.GetString("voice.arrival.in_one_minute", language)
	} else {
		return lm.GetString("voice.arrival.in_minutes", language, minutes)
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

func IsDisambiguationChoice(message string) int {
	message = strings.TrimSpace(message)

	// Support 1-99 for better scalability
	if len(message) >= 1 && len(message) <= 2 {
		// Check if all characters are digits
		for _, char := range message {
			if char < '0' || char > '9' {
				return 0
			}
		}

		// Parse the number
		if num := parseInteger(message); num >= 1 && num <= 99 {
			return num
		}
	}

	return 0
}

func parseInteger(s string) int {
	result := 0
	for _, char := range s {
		// Check for overflow before multiplication
		if result > (2147483647-int(char-'0'))/10 {
			return 0 // Return 0 for overflow to indicate invalid choice
		}
		result = result*10 + int(char-'0')
	}
	return result
}

func FormatDisambiguationMessage(stopOptions []models.StopOption, originalStopID string) string {
	if len(stopOptions) == 0 {
		return fmt.Sprintf("No stops found for ID %s.", originalStopID)
	}

	if len(stopOptions) == 1 {
		return ""
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("Multiple stops found for %s:\n", originalStopID))

	for i, option := range stopOptions {
		response.WriteString(fmt.Sprintf("%d) %s\n", i+1, option.DisplayText))
	}

	response.WriteString("Reply with the number to choose.")

	return response.String()
}
