package mock

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/logger"
)

func TestNewZapLoggerWritesStructuredFields(t *testing.T) {
	output := captureStderr(t, func() {
		log, err := logger.NewZapLogger(logger.Config{
			DisableCaller:     true,
			DisableStacktrace: true,
			Encoding:          "json",
			Level:             "debug",
		})
		if err != nil {
			t.Fatalf("NewZapLogger() error = %v", err)
		}
		log.Infow("HTTP request completed", "method", "GET", "status", http.StatusOK)
		_ = log.Sync()
	})

	for _, field := range []string{`"message":"HTTP request completed"`, `"method":"GET"`, `"status":200`} {
		if !strings.Contains(output, field) {
			t.Errorf("logger output %q does not contain structured field %s", output, field)
		}
	}
}

func TestNewZapLoggerRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config logger.Config
	}{
		{name: "invalid level", config: logger.Config{Encoding: "json", Level: "verbose"}},
		{name: "invalid encoding", config: logger.Config{Encoding: "xml", Level: "info"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := logger.NewZapLogger(test.config); err == nil {
				t.Fatal("NewZapLogger() error = nil, want invalid configuration error")
			}
		})
	}
}

func captureStderr(t *testing.T, write func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	originalStderr := os.Stderr
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = originalStderr
		_ = reader.Close()
		_ = writer.Close()
	})

	write()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stderr pipe: %v", err)
	}
	os.Stderr = originalStderr
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read logger output: %v", err)
	}
	return string(output)
}
