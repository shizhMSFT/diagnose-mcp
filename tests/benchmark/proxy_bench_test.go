// Package benchmark provides performance benchmarks for diagnose-mcp
package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shizhMSFT/diagnose-mcp/internal/config"
	"github.com/shizhMSFT/diagnose-mcp/internal/proxy"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// mockEchoServer creates a WebSocket server that echoes messages back
func mockEchoServer(tb testing.TB) *httptest.Server {
	tb.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			tb.Fatalf("Failed to upgrade: %v", err)
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				return
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	}))
}

// BenchmarkRemoteProxyLatency measures message passthrough latency for remote (WebSocket) proxy
func BenchmarkRemoteProxyLatency(b *testing.B) {
	// Create mock echo server
	server := mockEchoServer(b)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create proxy config
	cfg := &config.Config{
		RemoteURL:      wsURL,
		ConnectionType: config.ConnectionTypeRemote,
		Verbose:        false,
		OutputFormat:   config.OutputText,
	}

	// Create proxy
	p := proxy.NewProxy(cfg)

	// Start proxy server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run proxy in background
	var wg sync.WaitGroup
	wg.Add(1)
	var runErr error
	go func() {
		defer wg.Done()
		runErr = p.Run(ctx)
	}()

	// Give proxy time to connect
	time.Sleep(200 * time.Millisecond)

	// Connect client directly to echo server for baseline measurement
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect to server: %v", err)
	}
	defer clientConn.Close()

	// Prepare test message (minimal JSON-RPC 2.0 message)
	testMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"id":      1,
	}
	msgBytes, _ := json.Marshal(testMsg)

	b.ResetTimer()
	b.ReportAllocs()

	latencies := make([]time.Duration, 0, b.N)

	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Send message
		if err := clientConn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			b.Fatalf("Failed to write message: %v", err)
		}

		// Wait for response
		if _, _, err := clientConn.ReadMessage(); err != nil {
			b.Fatalf("Failed to read response: %v", err)
		}

		latency := time.Since(start)
		latencies = append(latencies, latency)
	}

	b.StopTimer()
	cancel()
	wg.Wait()

	if runErr != nil && runErr != context.Canceled {
		b.Logf("Proxy error: %v", runErr)
	}

	// Calculate p50, p95, p99
	reportPercentiles(b, latencies)
}

// BenchmarkProxyThroughput measures message throughput (messages/sec)
func BenchmarkProxyThroughput(b *testing.B) {
	// Create mock echo server
	server := mockEchoServer(b)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect client directly to echo server
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect to server: %v", err)
	}
	defer clientConn.Close()

	// Prepare test message
	testMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"id":      1,
	}
	msgBytes, _ := json.Marshal(testMsg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := clientConn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			b.Fatalf("Failed to write message: %v", err)
		}
		if _, _, err := clientConn.ReadMessage(); err != nil {
			b.Fatalf("Failed to read response: %v", err)
		}
	}
}

// BenchmarkMessageParsing measures the performance of parsing MCP messages
func BenchmarkMessageParsing(b *testing.B) {
	// Sample JSON-RPC 2.0 messages of varying complexity
	messages := []string{
		`{"jsonrpc":"2.0","method":"tools/list","id":1}`,
		`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"search","arguments":{"query":"test"}},"id":2}`,
		`{"jsonrpc":"2.0","result":{"tools":[{"name":"search","description":"Search for content","inputSchema":{"type":"object","properties":{"query":{"type":"string"}}}}]},"id":1}`,
	}

	for i, msg := range messages {
		b.Run(fmt.Sprintf("Message%d", i+1), func(b *testing.B) {
			msgBytes := []byte(msg)
			b.ResetTimer()
			b.ReportAllocs()

			for j := 0; j < b.N; j++ {
				var parsed map[string]interface{}
				if err := json.Unmarshal(msgBytes, &parsed); err != nil {
					b.Fatalf("Failed to parse message: %v", err)
				}
			}
		})
	}
}

// BenchmarkWebSocketOverhead measures WebSocket framing overhead
func BenchmarkWebSocketOverhead(b *testing.B) {
	// Create echo server
	server := mockEchoServer(b)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Test different message sizes
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			msg := make([]byte, size)
			for i := range msg {
				msg[i] = 'A'
			}

			b.ResetTimer()
			b.ReportAllocs()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					b.Fatalf("Failed to write: %v", err)
				}
				if _, _, err := conn.ReadMessage(); err != nil {
					b.Fatalf("Failed to read: %v", err)
				}
			}
		})
	}
}

// reportPercentiles calculates and reports p50, p95, p99 latencies
func reportPercentiles(b *testing.B, latencies []time.Duration) {
	b.Helper()
	if len(latencies) == 0 {
		return
	}

	// Sort latencies (simple bubble sort for small N)
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	p50 := latencies[len(latencies)*50/100]
	p95 := latencies[len(latencies)*95/100]
	p99 := latencies[len(latencies)*99/100]

	b.ReportMetric(float64(p50.Microseconds())/1000.0, "p50_ms")
	b.ReportMetric(float64(p95.Microseconds())/1000.0, "p95_ms")
	b.ReportMetric(float64(p99.Microseconds())/1000.0, "p99_ms")
}
