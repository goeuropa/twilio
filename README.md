# OneBusAway Twilio Integration

A Go web application that bridges Twilio SMS and voice services with OneBusAway transit APIs, allowing users to get real-time bus arrival information via text message or phone call.

## 🚌 Features

- **SMS Support**: Text a stop ID to get real-time arrival information
- **Voice/IVR Support**: Call and enter stop ID via keypad for spoken arrivals
- **Automatic Stop Resolution**: Handles both numeric stop IDs (e.g., `75403`) and full agency IDs (e.g., `1_75403`)
- **Multi-Agency Support**: Works with all Puget Sound area transit agencies
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
│   └── onebusaway_test.go    # Client unit tests
│
├── handlers/
│   ├── sms.go                # Twilio SMS webhook handler
│   └── voice.go              # Twilio voice/IVR webhook handlers
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
   # Edit .env with your preferred settings
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
2025/06/13 09:00:14 Starting server on port 8080
2025/06/13 09:00:14 OneBusAway API: https://api.pugetsound.onebusaway.org
[GIN-debug] Listening and serving HTTP on :8080
```

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
- **GET** `/` - Returns application info

### Twilio Webhooks
- **POST** `/sms` - Handle incoming SMS messages
- **POST** `/voice` - Handle incoming voice calls (initial menu)
- **POST** `/voice/input` - Handle voice input (DTMF digits)

## 💬 Usage Examples

### SMS Usage

Send an SMS to your Twilio number with a stop ID:

**Input**: `75403`

**Response**:
```
Stop: Pine St & 3rd Ave
Route 45 to Loyal Heights Greenwood: 2 min
Route 372 to U-District Station: 11 min
Route 67 to Northgate Station Roosevelt Station: 14 min
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

# OneBusAway API
ONEBUSAWAY_API_KEY=org.onebusaway.iphone           # API key (default provided)
ONEBUSAWAY_BASE_URL=https://api.pugetsound.onebusaway.org  # API base URL

# Twilio (optional - for outbound features)
TWILIO_ACCOUNT_SID=your_account_sid_here
TWILIO_AUTH_TOKEN=your_auth_token_here
```

### Stop ID Resolution

The app automatically resolves stop IDs by trying different agency prefixes:

- Input: `75403` → Tries `1_75403`, `40_75403`, `29_75403`, etc.
- Input: `1_75403` → Uses as-is (already has agency prefix)

**Supported Agencies**:
- `1_` - King County Metro
- `40_` - Sound Transit
- `29_` - Pierce Transit
- `95_` - Community Transit
- `97_` - Kitsap Transit
- `98_` - Everett Transit
- `3_` - Washington State Ferries
- `23_` - Other regional agencies

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
heroku config:set ONEBUSAWAY_API_KEY=org.onebusaway.iphone
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

- [OneBusAway Developer API](https://developer.onebusaway.org/)
- [Twilio SMS Webhooks](https://www.twilio.com/docs/messaging/guides/webhook-request)
- [Twilio Voice TwiML](https://www.twilio.com/docs/voice/twiml)
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [Puget Sound OneBusAway](https://pugetsound.onebusaway.org/)