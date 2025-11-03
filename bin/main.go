package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	RTMP_DEFAULT_CHUNK_SIZE = 128
	RTMP_HANDSHAKE_SIZE     = 1536
	RTMP_PORT               = 1935
)

// --- Message Types ---
const (
	RTMP_TYPE_SET_CHUNK_SIZE = 0x01
	RTMP_TYPE_COMMAND        = 0x14
	RTMP_TYPE_METADATA       = 0x12
	RTMP_TYPE_AUDIO          = 0x08
)

// --- AMF0 Types ---
const (
	AMF0_NUMBER     = 0x00
	AMF0_BOOL       = 0x01
	AMF0_STRING     = 0x02
	AMF0_OBJECT     = 0x03
	AMF0_NULL       = 0x05
	AMF0_OBJECT_END = 0x09
)

// --- Constants ---
var (
	av_onMetaData      = "onMetaData"
	av_duration        = "duration"
	av_width           = "width"
	av_height          = "height"
	av_videocodecid    = "videocodecid"
	av_videodatarate   = "videodatarate"
	av_framerate       = "framerate"
	av_audiocodecid    = "audiocodecid"
	av_audiodatarate   = "audiodatarate"
	av_audiosamplerate = "audiosamplerate"
	av_audiosamplesize = "audiosamplesize"
	av_audiochannels   = "audiochannels"
	av_stereo          = "stereo"
	av_encoder         = "encoder"
	av_avc1            = "avc1"
	av_mp4a            = "mp4a"
	av_OBSVersion      = "TwitchTest/1.4-qt"
)

// --- AMF0 Helpers ---
func amf0String(s string) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(AMF0_STRING)
	binary.Write(buf, binary.BigEndian, uint16(len(s)))
	buf.WriteString(s)
	return buf.Bytes()
}

func amf0Number(f float64) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(AMF0_NUMBER)
	binary.Write(buf, binary.BigEndian, f)
	return buf.Bytes()
}

func amf0Boolean(b bool) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(AMF0_BOOL)
	if b {
		buf.WriteByte(0x01)
	} else {
		buf.WriteByte(0x00)
	}
	return buf.Bytes()
}

func amf0Object(pairs map[string]interface{}) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(AMF0_OBJECT)
	for key, val := range pairs {
		binary.Write(buf, binary.BigEndian, uint16(len(key)))
		buf.WriteString(key)
		switch v := val.(type) {
		case string:
			buf.Write(amf0String(v))
		case float64:
			buf.Write(amf0Number(v))
		case int:
			buf.Write(amf0Number(float64(v)))
		case bool:
			buf.Write(amf0Boolean(v))
		default:
			buf.WriteByte(AMF0_NULL)
		}
	}
	buf.Write([]byte{0x00, 0x00, AMF0_OBJECT_END})
	return buf.Bytes()
}

// --- Messages ---
type Message struct {
	ChunkStreamID uint32
	Timestamp     uint32
	Size          uint32
	Type          uint8
	StreamID      uint32
	Buf           *bytes.Buffer
}

// TestResult stores test time, bytes count, average bitrate and success status
type TestResult struct {
	TestTime   time.Duration
	BytesCount int
	AvgBitrate float64
	Success    bool
}

func NewMessage(csi uint32, t uint8, sid uint32, ts uint32, data []byte) *Message {
	msg := &Message{
		ChunkStreamID: csi,
		Type:          t,
		StreamID:      sid,
		Timestamp:     ts,
		Buf:           bytes.NewBuffer(data),
		Size:          uint32(len(data)),
	}
	return msg
}

func (m *Message) SerializeChunked(chunkSize uint32) [][]byte {
	var chunks [][]byte
	payload := m.Buf.Bytes()
	remaining := len(payload)
	offset := 0

	for remaining > 0 {
		chunkLen := int(chunkSize)
		if remaining < chunkLen {
			chunkLen = remaining
		}

		var chunk bytes.Buffer
		if offset == 0 {
			// Type 0 header
			basicHeader := byte(0<<6) | byte(m.ChunkStreamID&0x3F)
			chunk.WriteByte(basicHeader)

			ts := m.Timestamp
			if ts >= 0xFFFFFF {
				ts = 0xFFFFFF
			}
			chunk.WriteByte(byte(ts >> 16))
			chunk.WriteByte(byte(ts >> 8))
			chunk.WriteByte(byte(ts))

			chunk.WriteByte(byte(m.Size >> 16))
			chunk.WriteByte(byte(m.Size >> 8))
			chunk.WriteByte(byte(m.Size))

			chunk.WriteByte(m.Type)

			binary.Write(&chunk, binary.LittleEndian, m.StreamID)
		} else {
			// Type 3 header
			basicHeader := byte(3<<6) | byte(m.ChunkStreamID&0x3F)
			chunk.WriteByte(basicHeader)
		}

		chunk.Write(payload[offset : offset+chunkLen])
		chunks = append(chunks, chunk.Bytes())

		offset += chunkLen
		remaining -= chunkLen
	}
	return chunks
}

// --- Payloads ---
func makeConnectPayload(app, tcURL string) []byte {
	buf := &bytes.Buffer{}
	buf.Write(amf0String("connect"))
	buf.Write(amf0Number(1.0)) // txID
	buf.Write(amf0Object(map[string]interface{}{
		"app":      app,
		"tcUrl":    tcURL,
		"flashVer": "FMLE/3.0 (compatible; FMSc/1.0)",
		"type":     "nonprivate",
	}))
	return buf.Bytes()
}

func makeCreateStreamPayload(txID float64) []byte {
	buf := &bytes.Buffer{}
	buf.Write(amf0String("createStream"))
	buf.Write(amf0Number(txID))
	buf.Write(amf0Null()) // null command object
	return buf.Bytes()
}

func amf0Null() []byte {
	return []byte{AMF0_NULL}
}

func makePublishPayload(streamName, publishType string) []byte {
	buf := &bytes.Buffer{}
	buf.Write(amf0String("publish"))
	buf.Write(amf0Number(0.0)) // txID = 0
	buf.Write(amf0Null())      // command object = null
	buf.Write(amf0String(streamName))
	buf.Write(amf0String(publishType))
	return buf.Bytes()
}

func makeMetadataPayload() []byte {
	buf := &bytes.Buffer{}
	buf.Write(amf0String(av_onMetaData)) // ← NOT @setDataFrame
	buf.Write(amf0Object(map[string]interface{}{
		av_duration:        0.0,
		av_width:           16.0,
		av_height:          16.0,
		av_videocodecid:    av_avc1,
		av_videodatarate:   10000.0,
		av_framerate:       30.0,
		av_audiocodecid:    av_mp4a,
		av_audiodatarate:   128.0,
		av_audiosamplerate: 44100.0,
		av_audiosamplesize: 16.0,
		av_audiochannels:   2.0,
		av_stereo:          true,
		av_encoder:         av_OBSVersion,
	}))
	return buf.Bytes()
}

func makeSetChunkSizePayload(newSize uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, newSize)
	return buf
}

// --- Send Helpers ---
func sendChunks(conn net.Conn, chunks [][]byte) (int, error) {
	total := 0
	for _, chunk := range chunks {
		n, err := conn.Write(chunk)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// --- Handshake ---
func rtmpHandshake(conn net.Conn) error {
	c0 := []byte{0x03}
	c1 := make([]byte, RTMP_HANDSHAKE_SIZE)
	binary.BigEndian.PutUint32(c1[0:4], uint32(time.Now().Unix()))
	// Rest can be zero
	conn.Write(c0)
	conn.Write(c1)

	handshake := make([]byte, 1+RTMP_HANDSHAKE_SIZE*2)
	conn.Read(handshake)

	conn.Write(c1) // C2 = C1
	return nil
}

func ParseRTMPURL(rtmpURL string) (server, app string, err error) {
	parsed, err := url.Parse(rtmpURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse URL: %w", err)
	}

	server = parsed.Host
	if server == "" {
		return "", "", fmt.Errorf("missing host in URL")
	}

	// Extract first path segment
	path := strings.TrimPrefix(parsed.Path, "/")
	if path == "" {
		// Default app name if none provided (common in RTMP)
		app = "live"
	} else {
		// Take only the first segment (e.g., "/live/abc" → "live")
		if i := strings.Index(path, "/"); i >= 0 {
			app = path[:i]
		} else {
			app = path
		}
	}
	return server, app, nil
}

// bandwidthtest performs the RTMP bandwidth test
func bandwidthtest(tcURL, streamKey string, durationSeconds int) TestResult {
	var streamID uint32 = 1

	server, app, err := ParseRTMPURL(tcURL)
	if err != nil {
		log.Fatal("Failed to parse URL:", err)
	}
	addr := fmt.Sprintf("%s:1935", server)
	log.Printf("connect to %s", addr)
	log.Printf("server %s", server)
	// log.Printf("tcURL %s", tcURL)

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		log.Fatal("Dial failed:", err)
	}
	defer conn.Close()

	log.Println("Starting handshake...")
	rtmpHandshake(conn)

	// Set chunk size to 4096
	setChunkMsg := NewMessage(2, RTMP_TYPE_SET_CHUNK_SIZE, 0, 0, makeSetChunkSizePayload(4096))
	sendChunks(conn, setChunkMsg.SerializeChunked(128))

	// Send connect
	connectMsg := NewMessage(3, RTMP_TYPE_COMMAND, 0, 0, makeConnectPayload(app, tcURL))
	sendChunks(conn, connectMsg.SerializeChunked(4096))

	// Send createStream
	createStreamMsg := NewMessage(3, RTMP_TYPE_COMMAND, 0, 0, makeCreateStreamPayload(2.0))
	sendChunks(conn, createStreamMsg.SerializeChunked(4096))

	// Send publish with stream key
	publishMsg := NewMessage(3, RTMP_TYPE_COMMAND, streamID, 0, makePublishPayload(streamKey, app))
	sendChunks(conn, publishMsg.SerializeChunked(4096))

	// Now send metadata
	metaMsg := NewMessage(3, RTMP_TYPE_METADATA, streamID, 0, makeMetadataPayload())
	sendChunks(conn, metaMsg.SerializeChunked(4096))

	// Send junk audio (with AAC header)
	audioPayload := make([]byte, 4096)
	audioPayload[0] = 0xAF // AAC, 44.1kHz, stereo
	audioPayload[1] = 0x01 // raw frame
	for i := 2; i < len(audioPayload); i++ {
		audioPayload[i] = 0xDE
	}

	start := time.Now()

	// Handle duration: 0 means infinite, otherwise use specified seconds
	var end time.Time
	if durationSeconds == 0 {
		// Infinite test - we'll set a very long timeout to avoid hanging
		end = time.Now().Add(24 * time.Hour) // 24 hours should be enough
	} else {
		end = start.Add(time.Duration(durationSeconds) * time.Second)
	}

	var timestamp uint32 = 0
	c := 0
	lastPrintTime := start

	// Limit average bitrate to 10000 kbps (10,000,000 bits per second)
	maxAvgBitrate := 10000.0                   // in kbps
	maxAvgBitrateBps := maxAvgBitrate * 1000.0 // convert to bps

	// Create ticker for printing every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(end) {
		audioMsg := NewMessage(5, RTMP_TYPE_AUDIO, streamID, timestamp, audioPayload)
		new_count, _ := sendChunks(conn, audioMsg.SerializeChunked(4096))
		c += new_count
		timestamp += 10 // 10ms per frame

		// Check if we're exceeding the average bitrate limit
		elapsed := time.Since(start).Seconds()
		if elapsed > 0 {
			currentAvgBitrateBps := float64(c*8) / elapsed
			if currentAvgBitrateBps > maxAvgBitrateBps {
				// Calculate how long we should wait to stay within the limit
				requiredBytes := int(maxAvgBitrateBps * elapsed / 8)
				if c > requiredBytes {
					// Calculate delay needed to maintain average bitrate
					sleepDuration := time.Duration((float64(c-requiredBytes)*8)/maxAvgBitrateBps) * time.Second
					if sleepDuration > 0 {
						time.Sleep(sleepDuration)
					}
				}
			}
		}

		// Check if it's time to print the current bitrate
		now := time.Now()
		if now.Sub(lastPrintTime) >= 1*time.Second {
			elapsed := now.Sub(start).Seconds()
			if elapsed > 0 {
				currentBitrateBps := float64(c*8) / elapsed
				currentBitrateKbps := currentBitrateBps / 1000.0
				log.Printf("Current bitrate: %.2f kbps", currentBitrateKbps)
			}
			lastPrintTime = now
		}

		// Check for ticker events to print every second
		select {
		case <-ticker.C:
			elapsed := time.Since(start).Seconds()
			if elapsed > 0 {
				currentBitrateBps := float64(c*8) / elapsed
				currentBitrateKbps := currentBitrateBps / 1000.0
				log.Printf("Current bitrate: %.2f kbps", currentBitrateKbps)
			}
		default:
		}
	}

	elapsed := time.Since(start)
	if elapsed.Seconds() <= 0 {
		log.Println("Elapsed time too short to measure bitrate")
		return TestResult{
			TestTime:   elapsed,
			BytesCount: c,
			AvgBitrate: 0.0,
			Success:    false,
		}
	}
	avgBitrateBps := float64(c*8) / elapsed.Seconds()
	avgBitrateKbps := avgBitrateBps / 1000.0

	return TestResult{
		TestTime:   elapsed,
		BytesCount: c,
		AvgBitrate: avgBitrateKbps,
		Success:    true,
	}
}

// --- Main ---
func main() {
	var streamKey string
	var tcURL string
	var durationSeconds int

	// Parse command line arguments
	if len(os.Args) >= 3 {
		tcURL = os.Args[1]
		streamKey = os.Args[2]
		if len(os.Args) >= 4 {
			// Try to parse duration if provided
			fmt.Sscanf(os.Args[3], "%d", &durationSeconds)
		}
	} else {
		streamKey = "SOMEKEY"
		tcURL = "rtmp://localhost/stream"
	}

	result := bandwidthtest(tcURL, streamKey, durationSeconds)

	// Print in JSON format
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonResult))
	}
}
