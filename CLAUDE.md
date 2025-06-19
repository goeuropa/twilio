# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OneBusAway Twilio Integration - A Go web application that bridges Twilio SMS and voice services with OneBusAway transit APIs, allowing users to get real-time bus arrival information via text message or phone call.

## Important Rules

Before saying your work is finished, you must always run these commands and ensure they pass:

* make lint
* make vet
* make test
* make fmt

## Common Development Commands

### Building and Running
```bash
# Build the application
go build -o oba-twilio .

# Run in development mode
go run main.go

# Run with environment variables
PORT=3000 ONEBUSAWAY_API_KEY=your_key go run main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output (includes live API calls)
go test -v ./...

# Skip integration tests (faster for development)
go test -short ./...

# Test individual packages
go test ./client         # Test OneBusAway client
go test ./formatters     # Test response formatting
go test .               # Test main application logic
```

### Dependencies
```bash
# Install/update dependencies
go mod download

# Tidy up go.mod and go.sum
go mod tidy
```

## Architecture

### Core Components

**main.go** - HTTP server setup with Gin routes and initialization
- Loads environment configuration
- Initializes OneBusAway client with coverage area detection
- Sets up SMS and voice handlers
- Configures health check endpoints

**client/onebusaway.go** - OneBusAway API client with caching and circuit breaker
- Manages API calls to OneBusAway REST API
- Handles stop ID resolution across multiple agencies (1_, 40_, 29_ prefixes for Puget Sound)
- Implements coverage area detection and geographic bounds for stop searching
- Includes caching (5min general, 1min arrivals, 60min coverage) and circuit breaker pattern

**handlers/** - Request handlers for Twilio webhooks
- `sms.go` - SMS message processing with disambiguation support
- `voice.go` - Voice/IVR call handling with DTMF input
- `session_store.go` - In-memory session management for multi-step interactions

**formatters/** - Response generation
- `response.go` - TwiML generation and text formatting for SMS/Voice responses
- `voice_templates.go` - Template-based voice response system using embedded XML templates
- `templates/*.xml` - Voice response templates (start, error, disambiguation, find_stop)

**models/types.go** - Data structures for Twilio requests and OneBusAway responses

### Key Features

**Stop ID Resolution**: Automatically tries agency prefixes (1_75403, 40_75403, etc.) when user provides numeric stop ID (75403)

**Disambiguation System**: When multiple stops match the same ID, presents numbered choices via SMS and stores session state

**Coverage Area Detection**: Calculates geographic center and radius from all transit agencies at startup to optimize stop searches

**Template-based Voice Responses**: Uses embedded XML templates for consistent TwiML generation

**Caching Strategy**: Multi-tier caching with different TTLs - 5min for general data, 1min for time-sensitive arrivals, 60min for static coverage data

## Environment Configuration

Required:
- `ONEBUSAWAY_API_KEY` - API key for OneBusAway server (must not be "test" or placeholder values)

Optional:
- `PORT` - Server port (default: 8080)
- `ONEBUSAWAY_BASE_URL` - API base URL (default: https://api.pugetsound.onebusaway.org)
- `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN` - For outbound Twilio features

## API Endpoints

- `POST /sms` - Handle incoming SMS messages (Twilio webhook)
- `POST /voice` - Handle incoming voice calls (Twilio webhook)
- `POST /voice/find_stop` - Handle voice input/DTMF (Twilio webhook)
- `GET /health` - Health check endpoint
- `GET /` - Application info with coverage area status

## Testing Approach

**Unit Tests**: Focus on individual package functionality
- `client/*_test.go` - API client and coverage calculation tests
- `formatters/*_test.go` - Response formatting and template tests
- `handlers/*_test.go` - Request handling and disambiguation logic tests

**Integration Tests**: Include live API calls (skipped with `-short` flag)
- Test real OneBusAway API responses
- Verify stop ID resolution across agencies
- Test coverage area calculation

**Manual Testing**: cURL commands for webhook simulation provided in README

## Multi-Server Support

Designed to work with any OneBusAway deployment by configuring `ONEBUSAWAY_BASE_URL`:
- Puget Sound (default): https://api.pugetsound.onebusaway.org
- Tampa: https://api.tampa.onebusaway.org
- UC Davis: https://api.unitrans.onebusawaycloud.com

Agency ID schemes vary by deployment - client automatically tries common prefixes and uses first successful match.