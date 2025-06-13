package models

type TwilioSMSRequest struct {
	From       string `form:"From" json:"From"`
	To         string `form:"To" json:"To"`
	Body       string `form:"Body" json:"Body"`
	MessageSid string `form:"MessageSid" json:"MessageSid"`
}

type TwilioVoiceRequest struct {
	From    string `form:"From" json:"From"`
	To      string `form:"To" json:"To"`
	CallSid string `form:"CallSid" json:"CallSid"`
	Digits  string `form:"Digits" json:"Digits,omitempty"`
}

type Stop struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Direction   string  `json:"direction"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lon"`
	RouteShortNames []string `json:"routeShortNames"`
}

type Arrival struct {
	RouteShortName        string    `json:"routeShortName"`
	TripHeadsign         string    `json:"tripHeadsign"`
	PredictedArrivalTime int64     `json:"predictedArrivalTime"`
	ScheduledArrivalTime int64     `json:"scheduledArrivalTime"`
	MinutesUntilArrival  int       `json:"minutesUntilArrival"`
	Status               string    `json:"status"`
}

type OneBusAwayResponse struct {
	Data struct {
		Entry struct {
			ArrivalsAndDepartures []struct {
				RouteShortName        string `json:"routeShortName"`
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
	} `json:"data"`
	Code int    `json:"code"`
	Text string `json:"text"`
}

type StopData struct {
	Data struct {
		List []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Lat  float64 `json:"lat"`
			Lon  float64 `json:"lon"`
		} `json:"list"`
	} `json:"data"`
	Code int    `json:"code"`
	Text string `json:"text"`
}

type AgenciesWithCoverageResponse struct {
	Data struct {
		LimitExceeded bool `json:"limitExceeded"`
		List []struct {
			AgencyID string  `json:"agencyId"`
			Lat      float64 `json:"lat"`
			LatSpan  float64 `json:"latSpan"`
			Lon      float64 `json:"lon"`
			LonSpan  float64 `json:"lonSpan"`
		} `json:"list"`
	} `json:"data"`
	Code int    `json:"code"`
	Text string `json:"text"`
}

type CoverageArea struct {
	CenterLat float64
	CenterLon float64
	Radius    float64
}

type TwiMLResponse struct {
	XMLName string `xml:"Response"`
	Say     string `xml:"Say,omitempty"`
	Message string `xml:"Message,omitempty"`
	Gather  *Gather `xml:"Gather,omitempty"`
}

type Gather struct {
	NumDigits int    `xml:"numDigits,attr,omitempty"`
	Action    string `xml:"action,attr,omitempty"`
	Method    string `xml:"method,attr,omitempty"`
	Say       string `xml:"Say,omitempty"`
}