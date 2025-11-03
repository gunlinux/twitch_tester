# RTMP Tester

RTMP Tester is a Go-based application designed to test RTMP streaming connections by connecting to an RTMP server, establishing a stream, and sending test data to simulate a live video stream.

## Features

- Implements RTMP handshake protocol
- Supports RTMP message serialization with chunking
- Handles AMF0 (Action Message Format 0) encoding for metadata
- Sends test audio data to simulate streaming
- Calculates and reports average upload bitrate
- Outputs test results in JSON format

## Prerequisites

- Go 1.25.3 or later

## Building

```bash
go build -o rtmp_tester main.go
```

## Running

The application accepts three arguments:
1. RTMP URL (e.g., `rtmp://localhost/live`)
2. Stream key (e.g., `stream_key`)
3. Test duration in seconds (optional, 0 = infinite)

Example usage:
```bash
go run main.go rtmp://localhost/live stream_key 10
```

Or using the provided test script:
```bash
export STREAM_KEY="TWITCH_STREAM_KEY"
./test.sh
```

This will connect to the specified RTMP server, establish a stream, and send test audio data for 10 seconds, reporting the average upload bitrate.

## Output

The application outputs test results in JSON format with the following fields:
- `TestTime`: Duration of the test in nanoseconds
- `BytesCount`: Total number of bytes transmitted
- `AvgBitrate`: Average upload bitrate in kbps
- `Success`: Boolean indicating whether the test was successful

## Project Structure

- `main.go` - Main application logic implementing RTMP protocol handling
- `test.sh` - Shell script with example command to run the application
- `go.mod` - Go module definition with Go version 1.25.3
- `go.sum` - Go module checksums

