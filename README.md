# OneBusAway Twilio Integration

A Go web application that bridges Twilio SMS and voice services with OneBusAway transit APIs, allowing users to get real-time bus arrival information via text message or phone call.

## 🚌 Features

- **SMS Support**: Text a stop ID to get real-time arrival information
- **Voice/IVR Support**: Call and enter stop ID via keypad for spoken arrivals
- **Smart Stop Disambiguation**: Detects multiple stops with same ID and asks user to choose
- **Automatic Stop Resolution**: Handles both numeric stop IDs (e.g., `75403`) and full agency IDs (e.g., `1_75403`)
- **Multi-Server Support**: Configurable to work with any OneBusAway server deployment
- **Dynamic Coverage Detection**: Automatically detects server coverage area at startup
- **Multi-Agency Support**: Works with multiple transit agencies within a region
- **Intelligent Stop Search**: Uses server-specific geographic bounds for stop searching
- **Real-time Data**: Fetches live arrival predictions from OneBusAway API
- **Production Ready**: Comprehensive error handling, logging, and health checks

## 📁 Project Structure

```
oba-twilio/
├── main.go                     # HTTP server with Gin routes and configuration
├── go.mod                      # Go module dependencies
├── go.sum                      # Dependency checksums
├── .env.example               # Environment configuration template
├── project-plan.md            # Original project specification
├── CLAUDE.md                  # Claude Code project instructions
├── README.md                  # This file
│
├── models/
│   └── types.go               # Data structures for Twilio requests and OBA responses
│
├── client/
│   ├── interface.go           # OneBusAway client interface (for testing)
│   ├── onebusaway.go         # OneBusAway API client implementation
│   ├── coverage_test.go      # Coverage area calculation tests
│   └── onebusaway_test.go    # Client unit tests
│
├── handlers/
│   ├── sms.go                # Twilio SMS webhook handler with disambiguation
│   ├── voice.go              # Twilio voice/IVR webhook handlers
│   ├── session_store.go      # In-memory session management for disambiguation
│   └── disambiguation_test.go # Tests for stop disambiguation logic
│
└── formatters/
    ├── response.go           # TwiML generation and text formatting
    └── response_test.go      # Formatter unit tests
```

## 🚀 Quick Start

### Prerequisites

- **Go 1.18+** ([installation guide](https://golang.org/doc/install))
- **Internet connection** (for OneBusAway API calls)
- **Optional**: Twilio account for production deployment

### Installation

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd oba-twilio
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Set up environment** (optional):
   ```bash
   cp .env.example .env
   # Edit .env to configure your OneBusAway server and settings
   # Default: Puget Sound (Seattle area)
   # See "Supported OneBusAway Servers" section for other options
   ```

4. **Build the application**:
   ```bash
   go build -o oba-twilio .
   ```

### Running the Application

**Development mode**:
```bash
go run main.go
```

**Production mode**:
```bash
./oba-twilio
```

The server will start on port 8080 by default. You should see:
```
2025/06/13 12:02:45 Initializing coverage area for OneBusAway server...
2025/06/13 12:02:45 Coverage area initialized: center=(47.7655,-122.3079), radius=92564m
2025/06/13 12:02:45 Starting server on port 8080
2025/06/13 12:02:45 OneBusAway API: https://api.pugetsound.onebusaway.org
[GIN-debug] Listening and serving HTTP on :8080
```

The coverage area initialization shows:
- **Center coordinates**: Geographic center of all transit agencies
- **Radius**: Calculated search radius in meters for stop queries
- **Automatic fallback**: If initialization fails, the app continues with limited functionality

## 🧪 Testing

### Run All Tests
```bash
go test ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...
```

### Run Integration Tests (includes live API calls)
```bash
go test -v ./...
```

### Skip Integration Tests (faster for development)
```bash
go test -short ./...
```

### Test Individual Packages
```bash
go test ./client         # Test OneBusAway client
go test ./formatters     # Test response formatting
go test .               # Test main application logic
```

## 📞 API Endpoints

### Health Check
- **GET** `/health` - Returns server health status
- **GET** `/` - Returns application info and coverage area status

### Twilio Webhooks
- **POST** `/sms` - Handle incoming SMS messages
- **POST** `/voice` - Handle incoming voice calls (initial menu)
- **POST** `/voice/input` - Handle voice input (DTMF digits)

### Internal API Methods (OneBusAway Client)
- `InitializeCoverage()` - Fetches server coverage area at startup
- `GetCoverageArea()` - Returns calculated center point and radius
- `SearchStops(query)` - Searches stops using dynamic geographic bounds
- `FindAllMatchingStops(stopID)` - Finds all stops matching a numeric ID across agencies
- `GetStopInfo(fullStopID)` - Gets detailed information for a specific stop

## 💬 Usage Examples

### SMS Usage

Send an SMS to your Twilio number with a stop ID:

**Input**: `75403`

**Response** (if single stop found):
```
Stop: Pine St & 3rd Ave
Route 45 to Loyal Heights Greenwood: 2 min
Route 372 to U-District Station: 11 min
Route 67 to Northgate Station Roosevelt Station: 14 min
```

**Response** (if multiple stops found):
```
Multiple stops found for 1234:
1) King County Metro: Pine St & 3rd Ave
2) Sound Transit: University Street Station
Reply with the number to choose.
```

**Follow-up**: User texts `2`

**Response**:
```
Stop: University Street Station
Route Link to SeaTac Airport: 3 min
Route Link to Northgate: 8 min
```

### Voice Usage

1. Call your Twilio number
2. Listen to: "Welcome to OneBusAway transit information. Please enter your stop ID followed by the pound key."
3. Enter stop ID on keypad: `75403#`
4. Hear: "Arrivals for Pine St & 3rd Ave. Route 45 to Loyal Heights Greenwood in 2 minutes. Route 372 to U-District Station in 11 minutes..."

### Manual Testing (curl)

**Test SMS endpoint**:
```bash
curl -X POST \
  -d "From=%2B14444444444&To=%2B15555555555&Body=75403&MessageSid=test" \
  http://localhost:8080/sms
```

**Test Voice endpoint**:
```bash
curl -X POST \
  -d "From=%2B14444444444&To=%2B15555555555&CallSid=test" \
  http://localhost:8080/voice
```

**Test Voice input**:
```bash
curl -X POST \
  -d "From=%2B14444444444&To=%2B15555555555&CallSid=test&Digits=75403" \
  http://localhost:8080/voice/input
```

## ⚙️ Configuration

### Environment Variables

Create a `.env` file or set environment variables:

```bash
# Server Configuration
PORT=8080                    # Server port (default: 8080)

# OneBusAway API Configuration
ONEBUSAWAY_API_KEY=test                                    # API key (default provided)
ONEBUSAWAY_BASE_URL=https://api.pugetsound.onebusaway.org  # API base URL (default: Puget Sound)

# Twilio (optional - for outbound features)
TWILIO_ACCOUNT_SID=your_account_sid_here
TWILIO_AUTH_TOKEN=your_auth_token_here
```

### Supported OneBusAway Servers

The application can be configured to work with any OneBusAway server deployment by setting the `ONEBUSAWAY_BASE_URL` environment variable:

| Region | Server URL | Coverage Area | Transit Agencies |
|--------|------------|---------------|------------------|
| **Puget Sound** (default) | `https://api.pugetsound.onebusaway.org` | Seattle, WA metro | King County Metro, Sound Transit, Pierce Transit, Community Transit, Kitsap Transit, Everett Transit, Washington State Ferries |
| **Tampa** | `https://api.tampa.onebusaway.org` | Tampa Bay, FL | Hillsborough Area Regional Transit (HART) |
| **Davis/UC Davis** | `https://api.unitrans.onebusawaycloud.com` | Davis, CA | Unitrans |
| **Local Development** | `http://localhost:8080` | Your local setup | Custom GTFS data |

**Example Configuration for Tampa**:
```bash
export ONEBUSAWAY_BASE_URL=https://api.tampa.onebusaway.org
export ONEBUSAWAY_API_KEY=test
go run main.go
```

### Stop ID Resolution

The app automatically resolves stop IDs by trying different agency prefixes. The prefixes used are optimized for the Puget Sound region by default, but work across different OneBusAway deployments:

- Input: `75403` → Tries `1_75403`, `40_75403`, `29_75403`, etc.
- Input: `1_75403` → Uses as-is (already has agency prefix)

**Note**: Different OneBusAway servers use different agency ID schemes.

The app will attempt all prefixes and return the first successful match, making it flexible across different deployments.

### Server Discovery

To find available OneBusAway servers and their coverage areas:

1. **OneBusAway Regions API**: `http://regions.onebusaway.org/regions-v3.json`
2. **Test server connectivity**:
   ```bash
   curl https://api.pugetsound.onebusaway.org/api/where/agencies-with-coverage.json?key=test
   ```
3. **Check agency IDs for a server**:
   ```bash
   # Replace with your target server
   curl https://api.tampa.onebusaway.org/api/where/agencies-with-coverage.json?key=test
   ```

### Testing Different Servers

**Test Tampa server**:
```bash
export ONEBUSAWAY_BASE_URL=https://api.tampa.onebusaway.org
go run main.go

# Test with a Tampa stop ID (example)
curl -X POST -d "From=%2B14444444444&Body=1234" http://localhost:8080/sms
```

**Test Unitrans server**:
```bash
export ONEBUSAWAY_BASE_URL=https://api.unitrans.onebusawaycloud.com
go run main.go

# Test with a Unitrans stop ID (example)
curl -X POST -d "From=%2B14444444444&Body=22136" http://localhost:8080/sms
```

## 🚀 Deployment

### Local Development with ngrok

For testing with real Twilio webhooks:

1. Install [ngrok](https://ngrok.com/)
2. Start your app: `go run main.go`
3. Expose via ngrok: `ngrok http 8080`
4. Configure Twilio webhooks to point to ngrok URL:
   - SMS: `https://your-id.ngrok.io/sms`
   - Voice: `https://your-id.ngrok.io/voice`

### Production Deployment

**Heroku**:
```bash
# Create Procfile
echo "web: ./oba-twilio" > Procfile

# Deploy
heroku create your-app-name
heroku config:set ONEBUSAWAY_API_KEY=test
git push heroku main
```

**Docker**:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o oba-twilio .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/oba-twilio .
EXPOSE 8080
CMD ["./oba-twilio"]
```

**Digital Ocean/AWS/GCP**:
- Build binary: `go build -o oba-twilio .`
- Upload to server
- Set environment variables
- Run with process manager (systemd, supervisor, etc.)

## 🔧 Troubleshooting

### Common Issues

**1. "No upcoming arrivals found"**
- **Cause**: Invalid stop ID or no current service
- **Solution**: Verify stop ID exists at [OneBusAway website](https://pugetsound.onebusaway.org/)
- **Example**: Stop 75403 should work, but 12345 might not exist

**2. "API returned status 404"**
- **Cause**: Stop ID doesn't exist with any agency prefix
- **Solution**: Double-check the stop ID number
- **Debug**: Check OneBusAway website for correct stop ID

**3. Tests failing with network errors**
- **Cause**: No internet connection or OneBusAway API down
- **Solution**: Run tests with `-short` flag to skip integration tests
- **Command**: `go test -short ./...`

**4. "Port already in use"**
- **Cause**: Another process using port 8080
- **Solution**: Set different port with `PORT=3000` or kill existing process

**5. XML parsing errors in Twilio**
- **Cause**: Invalid TwiML response format
- **Solution**: Check logs for malformed XML, ensure proper escaping

**6. "API returned status 404" with different servers**
- **Cause**: Server not responding or incorrect URL
- **Solution**: Verify server URL and test connectivity:
  ```bash
  curl https://your-server.com/api/where/agencies-with-coverage.json?key=test
  ```

**7. Different agency ID schemes**
- **Cause**: Each OneBusAway deployment uses different agency prefixes
- **Solution**: Check agency IDs for your target server:
  ```bash
  curl https://api.tampa.onebusaway.org/api/where/agencies-with-coverage.json?key=test
  ```
- **Debugging**: Look for `"id"` fields in the agency response to understand the prefix scheme

**8. "Coverage area initialization failed"**
- **Cause**: Server doesn't support agencies-with-coverage endpoint or API key issues
- **Solution**: App will continue with limited stop search functionality
- **Example**: `Warning: Failed to initialize coverage area: API returned status 401`
- **Workaround**: Use full stop IDs (with agency prefix) instead of stop name searches

**9. Stop disambiguation not working**
- **Cause**: Multiple stops exist with same ID but system returns first match
- **Solution**: Check logs for "FindAllMatchingStops" calls
- **Debug**: Test with known conflicting stop IDs like transit station stops
- **Example**: Try stop IDs that exist in both Metro (1_) and Sound Transit (40_) systems

**10. Session timeouts for disambiguation**
- **Cause**: User doesn't respond to disambiguation choice within 10 minutes
- **Solution**: App automatically clears old sessions, user needs to send stop ID again
- **Prevention**: Respond to disambiguation messages promptly

### Debug Mode

Enable verbose logging:
```bash
export GIN_MODE=debug
go run main.go
```

View detailed request/response logs in console.

### Performance Tuning

**For high traffic**:
- Set `GIN_MODE=release` in production
- Implement caching for frequent stop ID queries
- Add rate limiting middleware
- Use connection pooling for OneBusAway API calls

### API Rate Limits

OneBusAway API limits:
- No hard limits documented for Puget Sound instance
- Recommended: Don't exceed 10 requests/second
- Cache responses for 30-60 seconds for same stop ID

## 🤝 Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature-name`
3. Write tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit pull request

### Code Style

- Follow `gofmt` formatting
- Use `golint` for style checking
- Add tests for new functions
- Keep functions small and focused
- Document exported functions

## 📄 License

This project is open source. See project repository for license details.

## 🆘 Support

- **Issues**: Create GitHub issue with reproduction steps
- **Questions**: Check existing issues or create new one
- **Documentation**: See `project-plan.md` for detailed technical specs

## 🔗 Related Links

### OneBusAway Resources
- [OneBusAway Developer API](https://developer.onebusaway.org/)
- [OneBusAway Multi-Region Documentation](https://github.com/OneBusAway/onebusaway/wiki/Multi-Region)
- [OneBusAway Server Directory](http://regions.onebusaway.org/regions-v3.json)
- [OneBusAway Deployments](https://github.com/OneBusAway/onebusaway/wiki/OneBusAway-Deployments)

### Regional OneBusAway Instances
- [Puget Sound OneBusAway](https://pugetsound.onebusaway.org/) (Seattle area)
- [Tampa OneBusAway](https://tampa.onebusaway.org/) (Tampa Bay area)
- [OneBusAway Atlanta](https://atlanta.onebusaway.org/) (Atlanta metro)

### Twilio & Development Resources
- [Twilio SMS Webhooks](https://www.twilio.com/docs/messaging/guides/webhook-request)
- [Twilio Voice TwiML](https://www.twilio.com/docs/voice/twiml)
- [Gin Web Framework](https://gin-gonic.com/docs/)