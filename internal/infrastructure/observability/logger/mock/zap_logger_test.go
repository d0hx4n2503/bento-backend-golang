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
	if strings.Contains(output, "\x1b[") {
		t.Errorf("JSON logger output contains ANSI color codes: %q", output)
	}
}

func TestNewZapLoggerColorsConsoleLevels(t *testing.T) {
	output := captureStderr(t, func() {
		log, err := logger.NewZapLogger(logger.Config{
			DisableStacktrace: true,
			Encoding:          "console",
			Level:             "debug",
		})
		if err != nil {
			t.Fatalf("NewZapLogger() error = %v", err)
		}
		log.Info("colored console log")
		log.Warn("warning console log")
		log.Error("error console log")
		_ = log.Sync()
	})

	for _, color := range []string{
		"\x1b[32mINFO\x1b[0m",
		"\x1b[33mWARN\x1b[0m",
		"\x1b[31mERROR\x1b[0m",
	} {
		if !strings.Contains(output, color) {
			t.Errorf("console logger output %q does not contain level color %q", output, color)
		}
	}
	if !strings.Contains(output, "colored console log") {
		t.Errorf("console logger output %q does not contain the log message", output)
	}
	if !strings.Contains(output, "\x1b[9") {
		t.Errorf("console logger output %q does not contain a colored caller", output)
	}
}

func TestNewZapLoggerColorsConsoleFieldKeysOnly(t *testing.T) {
	output := captureStderr(t, func() {
		log, err := logger.NewZapLogger(logger.Config{
			DisableCaller:     true,
			DisableStacktrace: true,
			Encoding:          "console",
			Level:             "debug",
		})
		if err != nil {
			t.Fatalf("NewZapLogger() error = %v", err)
		}
		log.Infow("structured console log",
			"service_name", "bento",
			"status", 200,
		)
		_ = log.Sync()
	})

	coloredKeys := []string{
		"\x1b[96m\"service_name\"\x1b[0m: \"bento\"",
		"\x1b[96m\"status\"\x1b[0m: 200",
	}
	for _, key := range coloredKeys {
		if !strings.Contains(output, key) {
			t.Errorf("console logger output %q does not contain colored field key %q", output, key)
		}
	}
	if strings.Contains(output, "\x1b[96m\"bento\"") {
		t.Errorf("console logger output colors a field value: %q", output)
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
