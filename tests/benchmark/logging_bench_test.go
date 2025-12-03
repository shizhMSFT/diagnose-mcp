// Package benchmark provides performance benchmarks for diagnose-mcp
package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/shizhMSFT/diagnose-mcp/internal/config"
	"github.com/shizhMSFT/diagnose-mcp/internal/logger"
)

// BenchmarkTextLogger benchmarks text logging performance
func BenchmarkTextLogger(b *testing.B) {
	buf := &bytes.Buffer{}
	log := logger.NewLogger(buf, false)

	entry := &logger.LogEntry{
		Level:   logger.LogLevelInfo,
		Type:    logger.LogEntryTypeRequest,
		Message: "Test message",
		Context: map[string]interface{}{
			"method": "tools/list",
			"id":     1,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Log(entry)
	}
}

// BenchmarkTextLoggerVerbose benchmarks verbose text logging with full payloads
func BenchmarkTextLoggerVerbose(b *testing.B) {
	buf := &bytes.Buffer{}
	log := logger.NewLogger(buf, true)

	entry := &logger.LogEntry{
		Level:   logger.LogLevelInfo,
		Type:    logger.LogEntryTypeRequest,
		Message: "Test message",
		Context: map[string]interface{}{
			"method": "tools/list",
			"id":     1,
			"params": map[string]interface{}{
				"query": "test",
				"limit": 10,
			},
		},
		Payload: map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"params": map[string]interface{}{
				"query": "test query string",
				"limit": 10,
			},
			"id": 1,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Log(entry)
	}
}

// BenchmarkJSONLogger benchmarks JSON logging performance
func BenchmarkJSONLogger(b *testing.B) {
	buf := &bytes.Buffer{}
	log := logger.NewJSONLogger(buf, false)

	entry := &logger.LogEntry{
		Level:   logger.LogLevelInfo,
		Type:    logger.LogEntryTypeRequest,
		Message: "Test message",
		Context: map[string]interface{}{
			"method": "tools/list",
			"id":     1,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Log(entry)
	}
}

// BenchmarkJSONLoggerVerbose benchmarks verbose JSON logging with full payloads
func BenchmarkJSONLoggerVerbose(b *testing.B) {
	buf := &bytes.Buffer{}
	log := logger.NewJSONLogger(buf, true)

	entry := &logger.LogEntry{
		Level:   logger.LogLevelInfo,
		Type:    logger.LogEntryTypeRequest,
		Message: "Test message",
		Context: map[string]interface{}{
			"method": "tools/list",
			"id":     1,
			"params": map[string]interface{}{
				"query": "test",
				"limit": 10,
			},
		},
		Payload: map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"params": map[string]interface{}{
				"query": "test query string",
				"limit": 10,
			},
			"id": 1,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Log(entry)
	}
}

// BenchmarkLoggerOutputFormats compares text vs JSON logging overhead
func BenchmarkLoggerOutputFormats(b *testing.B) {
	entry := &logger.LogEntry{
		Level:   logger.LogLevelInfo,
		Type:    logger.LogEntryTypeRequest,
		Message: "Test message",
		Context: map[string]interface{}{
			"method": "tools/list",
			"id":     1,
		},
	}

	b.Run("Text", func(b *testing.B) {
		buf := &bytes.Buffer{}
		log := logger.NewLogger(buf, false)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			log.Log(entry)
		}
	})

	b.Run("JSON", func(b *testing.B) {
		buf := &bytes.Buffer{}
		log := logger.NewJSONLogger(buf, false)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			log.Log(entry)
		}
	})
}

// BenchmarkFileLogging benchmarks logging to actual file
func BenchmarkFileLogging(b *testing.B) {
	tmpfile, err := os.CreateTemp("", "benchmark-log-*.json")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	log := logger.NewJSONLogger(tmpfile, false)

	entry := &logger.LogEntry{
		Level:   logger.LogLevelInfo,
		Type:    logger.LogEntryTypeRequest,
		Message: "Test message",
		Context: map[string]interface{}{
			"method": "tools/list",
			"id":     1,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Log(entry)
	}
}

// BenchmarkLogEntryCreation benchmarks the cost of creating log entries
func BenchmarkLogEntryCreation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = &logger.LogEntry{
			Level:   logger.LogLevelInfo,
			Type:    logger.LogEntryTypeRequest,
			Message: "Test message",
			Context: map[string]interface{}{
				"method": "tools/list",
				"id":     1,
			},
		}
	}
}

// BenchmarkJSONMarshaling benchmarks raw JSON marshaling (baseline for comparison)
func BenchmarkJSONMarshaling(b *testing.B) {
	data := map[string]interface{}{
		"level":   "INFO",
		"type":    "request",
		"message": "Test message",
		"context": map[string]interface{}{
			"method": "tools/list",
			"id":     1,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		json.Marshal(data)
	}
}

// Benchmark no-op proxy (baseline)
func BenchmarkNoOpProxy(b *testing.B) {
	// Create config with minimal overhead
	cfg := &config.Config{
		ConnectionType: config.ConnectionTypeLocal,
		Verbose:        false,
		OutputFormat:   config.OutputText,
	}

	// Measure the cost of just creating a proxy
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = context.Background()
		_ = cfg
	}
}
