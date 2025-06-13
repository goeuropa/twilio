# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Golang web application that receives SMS and IVR-based callbacks from the Twilio API, translates them into requests for transit information, and forwards them along to a OneBusAway server.

## Development Status

This appears to be a new or minimal project with only a README.markdown file present. The Go source files, dependencies (go.mod), and build configuration are not yet implemented.

## Expected Architecture

Based on the project description, this application will likely include:
- HTTP handlers for Twilio webhook callbacks (SMS and IVR)
- Request translation logic to convert Twilio requests to OneBusAway API calls
- HTTP client for communicating with OneBusAway servers
- Response formatting to return appropriate Twilio-compatible responses

## Common Commands

Once the Go project is initialized, typical commands will include:
- `go mod init` - Initialize Go module
- `go build` - Build the application
- `go run main.go` - Run the application
- `go test ./...` - Run tests