// Package benchmark provides performance benchmarks for diagnose-mcp
package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shizhMSFT/diagnose-mcp/internal/config"
	"github.com/shizhMSFT/diagnose-mcp/internal/proxy"
)

// BenchmarkMemoryAllocation measures memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	// Create mock echo server
	server := mockEchoServer(b)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	testMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"id":      1,
	}
	msgBytes, _ := json.Marshal(testMsg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
		if _, _, err := conn.ReadMessage(); err != nil {
			b.Fatalf("Failed to read: %v", err)
		}
	}
}

// BenchmarkHeapGrowth measures heap growth over extended session
func BenchmarkHeapGrowth(b *testing.B) {
	// Create mock echo server
	server := mockEchoServer(b)
	defer server.Close()

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

	// Start proxy
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	var runErr error
	go func() {
		defer wg.Done()
		runErr = p.Run(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Connect client
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer clientConn.Close()

	testMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"id":      1,
	}
	msgBytes, _ := json.Marshal(testMsg)

	// Measure initial heap
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	// Simulate extended session (scaled down for benchmarking)
	iterations := b.N
	if iterations < 1000 {
		iterations = 1000
	}

	for i := 0; i < iterations; i++ {
		if err := clientConn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
		if _, _, err := clientConn.ReadMessage(); err != nil {
			b.Fatalf("Failed to read: %v", err)
		}

		// Periodically check for memory leaks
		if i%100 == 0 {
			runtime.GC()
		}
	}

	b.StopTimer()

	// Measure final heap
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	b.ReportMetric(float64(heapGrowth)/1024/1024, "heap_mb")

	cancel()
	wg.Wait()

	if runErr != nil && runErr != context.Canceled {
		b.Logf("Proxy error: %v", runErr)
	}
}

// BenchmarkGoroutineLeak checks for goroutine leaks over extended usage
func BenchmarkGoroutineLeak(b *testing.B) {
	// Create mock echo server
	server := mockEchoServer(b)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Measure initial goroutines
	initialGoroutines := runtime.NumGoroutine()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create proxy config
		cfg := &config.Config{
			RemoteURL:      wsURL,
			ConnectionType: config.ConnectionTypeRemote,
			Verbose:        false,
			OutputFormat:   config.OutputText,
		}

		// Create and run proxy
		p := proxy.NewProxy(cfg)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Run(ctx)
		}()

		// Send some messages
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			cancel()
			wg.Wait()
			b.Fatalf("Failed to connect: %v", err)
		}

		testMsg := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"id":      1,
		}
		msgBytes, _ := json.Marshal(testMsg)

		for j := 0; j < 10; j++ {
			conn.WriteMessage(websocket.TextMessage, msgBytes)
			conn.ReadMessage()
		}

		conn.Close()
		cancel()
		wg.Wait()
	}

	b.StopTimer()

	// Allow goroutines to clean up
	runtime.Gosched()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	goroutineLeak := finalGoroutines - initialGoroutines

	b.ReportMetric(float64(goroutineLeak), "leaked_goroutines")

	if goroutineLeak > 10 {
		b.Logf("WARNING: Possible goroutine leak detected (%d leaked)", goroutineLeak)
	}
}

// BenchmarkLongRunningSession simulates a 1-minute session (scaled down for testing)
func BenchmarkLongRunningSession(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping long-running benchmark in short mode")
	}

	// Create mock echo server
	server := mockEchoServer(b)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	cfg := &config.Config{
		RemoteURL:      wsURL,
		ConnectionType: config.ConnectionTypeRemote,
		Verbose:        false,
		OutputFormat:   config.OutputText,
	}

	p := proxy.NewProxy(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	var runErr error
	go func() {
		defer wg.Done()
		runErr = p.Run(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	testMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"id":      1,
	}
	msgBytes, _ := json.Marshal(testMsg)

	// Memory snapshots
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	// Simulate realistic traffic: ~10 msg/sec for 1 minute = 600 messages (scaled down)
	duration := 10 * time.Second // Scaled down from 60s
	messagesPerSecond := 10
	totalMessages := int(duration.Seconds()) * messagesPerSecond

	start := time.Now()
	ticker := time.NewTicker(time.Second / time.Duration(messagesPerSecond))
	defer ticker.Stop()

	messageCount := 0
	for messageCount < totalMessages {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				b.Fatalf("Failed to write: %v", err)
			}
			if _, _, err := conn.ReadMessage(); err != nil {
				b.Fatalf("Failed to read: %v", err)
			}
			messageCount++

		case <-time.After(duration + time.Second):
			b.Fatalf("Timeout waiting for messages")
		}
	}

	elapsed := time.Since(start)

	b.StopTimer()

	// Memory snapshot after session
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	avgLatency := elapsed / time.Duration(totalMessages)

	b.ReportMetric(float64(heapGrowth)/1024/1024, "heap_mb")
	b.ReportMetric(float64(avgLatency.Microseconds())/1000.0, "avg_latency_ms")
	b.ReportMetric(float64(totalMessages)/elapsed.Seconds(), "throughput_msg/s")

	cancel()
	wg.Wait()

	if runErr != nil && runErr != context.Canceled {
		b.Logf("Proxy error: %v", runErr)
	}

	// Verify memory target: <100MB total heap growth
	if heapGrowth > 100*1024*1024 {
		b.Errorf("Memory target FAILED: heap grew by %.2f MB (target: <100 MB)", float64(heapGrowth)/1024/1024)
	} else {
		b.Logf("Memory target PASSED: heap grew by %.2f MB", float64(heapGrowth)/1024/1024)
	}
}

// BenchmarkConcurrentConnections measures behavior under concurrent load
func BenchmarkConcurrentConnections(b *testing.B) {
	// Create mock echo server
	server := mockEchoServer(b)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Test different concurrency levels
	concurrencyLevels := []int{1, 5, 10, 25}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Connections%d", concurrency), func(b *testing.B) {
			testMsg := map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "tools/list",
				"id":      1,
			}
			msgBytes, _ := json.Marshal(testMsg)

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
				if err != nil {
					b.Errorf("Failed to connect: %v", err)
					return
				}
				defer conn.Close()

				for pb.Next() {
					if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
						b.Errorf("Failed to write: %v", err)
						return
					}
					if _, _, err := conn.ReadMessage(); err != nil {
						b.Errorf("Failed to read: %v", err)
						return
					}
				}
			})
		})
	}
}
