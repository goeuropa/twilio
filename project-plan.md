# OneBusAway Twilio Integration - Project Plan

## Project Overview
Build a Golang web application that receives SMS and IVR callbacks from Twilio, translates them into OneBusAway transit information requests, and returns formatted responses to users.

## User Stories

### SMS Flow
1. User sends SMS to Twilio number with stop ID (e.g., "75403") or stop name
2. App receives Twilio webhook, extracts stop information
3. App queries OneBusAway API for arrival/departure data, inferring the agency prefix (e.g. `1_` for a full OBA stop ID of `1_75403`) via cached copies of the stop-ids-for-agency API.
4. App formats response and sends back via SMS

### IVR Flow
1. User calls Twilio number
2. App presents voice menu: "Press 1 to enter stop ID, Press 2 for help"
3. User enters stop ID via keypad
4. App queries OneBusAway API
5. App reads back arrival information via text-to-speech

## Technical Architecture

### Core Components

1. **HTTP Server** (`main.go`)
   - Gin/Echo web framework
   - Routes for `/sms` and `/voice` webhooks
   - Environment configuration (port, API keys)

2. **SMS Handler** (`handlers/sms.go`)
   - Parse incoming Twilio SMS webhooks
   - Extract stop ID from message body
   - Return TwiML SMS response

3. **Voice Handler** (`handlers/voice.go`)
   - Generate TwiML for IVR menus
   - Handle DTMF input collection
   - Convert transit data to speech

4. **OneBusAway Client** (`client/onebusaway.go`)
   - Initialize official OneBusAway Go SDK client
   - Wrapper functions for common transit queries
   - Handle SDK errors gracefully

5. **Models** (`models/`)
   - `Stop` struct for stop information
   - `Arrival` struct for arrival/departure data
   - `TwilioRequest` struct for webhook data

6. **Response Formatters** (`formatters/`)
   - Convert OneBusAway data to readable SMS text
   - Generate TwiML for voice responses

## API Integrations

### OneBusAway Go SDK
- **Installation**: `go get github.com/OneBusAway/go-sdk`
- **Requirements**: Go 1.18+
- **Authentication**: API key via environment variable `ONEBUSAWAY_API_KEY` or client option
- **Usage**:
  ```go
  client := onebusaway.NewClient(
      option.WithAPIKey("My API Key"),
  )
  arrivals, err := client.ArrivalsAndDeparturesForStop.Get(context.TODO(), stopID)
  ```

### Twilio Webhooks

#### SMS Webhook Parameters
- `From`: Sender's phone number (+14444444444)
- `To`: Twilio number (+15555555555)
- `Body`: SMS message content
- `MessageSid`: Unique message identifier

#### Voice Webhook Parameters
- `From`: Caller's phone number
- `To`: Twilio number
- `CallSid`: Unique call identifier
- `Digits`: DTMF input (when using `<Gather>`)

## Implementation Steps

### Phase 1: Project Setup
1. Initialize Go module: `go mod init oba-twilio`
2. Add dependencies:
   ```bash
   go get github.com/gin-gonic/gin
   go get github.com/joho/godotenv
   go get github.com/OneBusAway/go-sdk
   ```
3. Create directory structure:
   ```
   /
   ├── main.go
   ├── handlers/
   │   ├── sms.go
   │   └── voice.go
   ├── client/
   │   └── onebusaway.go
   ├── models/
   │   └── types.go
   ├── formatters/
   │   └── response.go
   └── .env
   ```

### Phase 2: OneBusAway Integration
1. Install official OneBusAway Go SDK: `go get github.com/OneBusAway/go-sdk`
2. Initialize client with API key authentication
3. Implement arrival/departure queries using SDK methods
4. Test with known stop IDs

### Phase 3: SMS Handler
1. Create `/sms` endpoint
2. Parse Twilio webhook parameters
3. Extract stop ID from SMS body (support both numeric IDs and stop names)
4. Query OneBusAway API
5. Format response as SMS-friendly text
6. Return TwiML `<Message>` response

### Phase 4: Voice/IVR Handler
1. Create `/voice` endpoint for incoming calls
2. Generate initial TwiML menu with `<Gather>` and `<Say>`
3. Create `/voice/input` endpoint to handle DTMF input
4. Query OneBusAway API with collected stop ID
5. Convert arrival times to speech-friendly format
6. Return TwiML with `<Say>` responses

### Phase 5: Enhanced Features
1. **Stop Name Lookup**: Support text queries like "Pine & 3rd"
2. **Favorites**: Allow users to save frequently used stops
3. **Service Alerts**: Include transit alerts in responses
4. **Multiple Stops**: Handle queries for nearby stops
5. **Route Filtering**: Allow filtering by specific bus routes

### Phase 6: Deployment & Operations
1. Add health check endpoint
2. Implement logging and metrics
3. Add rate limiting for API calls
4. Deploy to cloud platform (Heroku, AWS, etc.)
5. Configure Twilio webhook URLs

## Environment Configuration

Required environment variables:
```bash
# Server
PORT=8080

# OneBusAway API
ONEBUSAWAY_API_KEY=your_api_key_here

# Twilio (for outbound functionality)
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token
```

## Example API Responses

### OneBusAway Response Format
```json
{
  "data": {
    "entry": {
      "arrivalsAndDepartures": [
        {
          "routeShortName": "8",
          "tripHeadsign": "Seattle Center",
          "predictedArrivalTime": 1641234567000,
          "scheduledArrivalTime": 1641234500000
        }
      ]
    }
  }
}
```

### SMS Response Example
```
Route 8 to Seattle Center: 3 min
Route 43 to Capitol Hill: 8 min
Route 49 to U District: 12 min
```

### TwiML Voice Response Example
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Say>Next arrivals: Route 8 to Seattle Center in 3 minutes. Route 43 to Capitol Hill in 8 minutes.</Say>
</Response>
```

## Testing Strategy

1. **Unit Tests**: Test each handler and client function
2. **Integration Tests**: Test full SMS/Voice flows with mock APIs
3. **Manual Testing**: Use ngrok for local webhook testing
4. **Load Testing**: Ensure application handles concurrent requests

## Resources

- [OneBusAway Developer Documentation](https://developer.onebusaway.org)
- [Twilio SMS Webhooks](https://www.twilio.com/docs/messaging/guides/webhook-request)
- [Twilio Voice TwiML](https://www.twilio.com/docs/voice/twiml)
- [Gin Web Framework](https://gin-gonic.com/docs/)

## Common Challenges & Solutions

1. **Stop ID Resolution**: OneBusAway uses agency-specific stop IDs (e.g., "1_75403"). Implement fuzzy matching for stop names.
2. **API Rate Limiting**: Cache frequent requests and implement exponential backoff.
3. **Webhook Security**: Validate Twilio signatures to prevent unauthorized requests.
4. **Time Formatting**: Convert Unix timestamps to human-readable relative times ("3 minutes").
5. **Error Handling**: Provide helpful error messages when stops don't exist or API is down.