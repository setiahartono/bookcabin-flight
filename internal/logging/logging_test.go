package logging

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMiddlewareLogsEveryRequest(t *testing.T) {
	var access, failures, providers bytes.Buffer

	handler := New(&access, &failures, &providers).Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"flights":[]}`))
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/search", nil)
	request.RemoteAddr = "192.0.2.10:54321"

	handler.ServeHTTP(httptest.NewRecorder(), request)

	line := access.String()
	for _, want := range []string{
		"remote=192.0.2.10",
		"method=POST",
		"path=/api/v1/search",
		"status=200",
		"bytes=14",
		"duration_ms=",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("access log = %q, want it to hold %q", line, want)
		}
	}

	if failures.Len() != 0 {
		t.Errorf("error log = %q, want nothing for a request that answered 200", failures.String())
	}
}

func TestMiddlewareLogsAFailedRequestToTheErrorLog(t *testing.T) {
	var access, failures, providers bytes.Buffer

	handler := New(&access, &failures, &providers).Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid search criteria"}`))
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/search", nil))

	if line := access.String(); !strings.Contains(line, "status=400") {
		t.Errorf("access log = %q, want it to hold status=400", line)
	}

	line := failures.String()
	for _, want := range []string{
		"status=400",
		`error="{\"error\":\"invalid search criteria\"}"`,
	} {
		if !strings.Contains(line, want) {
			t.Errorf("error log = %q, want it to hold %q", line, want)
		}
	}
}

func TestMiddlewareLogsTheForwardedAddress(t *testing.T) {
	var access, failures, providers bytes.Buffer

	handler := New(&access, &failures, &providers).Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	request.Header.Set("X-Forwarded-For", "10.0.0.7, 10.0.0.1")

	handler.ServeHTTP(httptest.NewRecorder(), request)

	if line := access.String(); !strings.Contains(line, "remote=10.0.0.7") {
		t.Errorf("access log = %q, want the address the proxy forwarded", line)
	}
}

func TestMiddlewareLogsAnImplicitStatus(t *testing.T) {
	var access, failures, providers bytes.Buffer

	handler := New(&access, &failures, &providers).Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

	if line := access.String(); !strings.Contains(line, "status=200") {
		t.Errorf("access log = %q, want status=200 for a body written without a status", line)
	}
	if failures.Len() != 0 {
		t.Errorf("error log = %q, want nothing for a request that answered 200", failures.String())
	}
}

func TestOpenWritesTheAccessLogToStdoutAndToFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")

	printed, restore := captureStdout(t)
	logger, err := Open(dir)
	restore()

	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	handler := logger.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limit exceeded: 3 requests per 5s"}`))
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/search", nil))

	logger.ProviderFailed("Batik Air", "CGK", "DPS", time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
		errors.New("Batik Air: provider unavailable"))

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	access := readFile(t, filepath.Join(dir, AccessLogName))
	failure := readFile(t, filepath.Join(dir, ErrorLogName))
	provider := readFile(t, filepath.Join(dir, ProviderLogName))
	stdout := printed()

	if !strings.Contains(access, "status=429") {
		t.Errorf("%s = %q, want the access line", AccessLogName, access)
	}
	if !strings.Contains(stdout, "status=429") {
		t.Errorf("stdout = %q, want the access line there as well", stdout)
	}
	if !strings.Contains(failure, "status=429") {
		t.Errorf("%s = %q, want the request that failed", ErrorLogName, failure)
	}
	if !strings.Contains(provider, "provider=Batik Air") {
		t.Errorf("%s = %q, want the provider that failed", ProviderLogName, provider)
	}
	if strings.Contains(failure, "provider=Batik Air") {
		t.Errorf("%s = %q, want a provider failure kept out of the error log", ErrorLogName, failure)
	}
}

func TestProviderFailedWritesTheProviderLog(t *testing.T) {
	var access, failures, providers bytes.Buffer

	New(&access, &failures, &providers).ProviderFailed("Batik Air", "CGK", "DPS", time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
		errors.New("Batik Air: provider unavailable"))

	line := providers.String()
	for _, want := range []string{
		"provider=Batik Air",
		"route=CGK-DPS",
		"departure_date=2025-12-15",
		`error="Batik Air: provider unavailable"`,
	} {
		if !strings.Contains(line, want) {
			t.Errorf("provider log = %q, want it to hold %q", line, want)
		}
	}

	if access.Len() != 0 {
		t.Errorf("access log = %q, want nothing for a provider failure", access.String())
	}
	if failures.Len() != 0 {
		t.Errorf("error log = %q, want the provider failure in its own log", failures.String())
	}
}

// captureStdout prints what a test writes to stdout through the returned function,
// which restores os.Stdout when it is called.
func captureStdout(t *testing.T) (func() string, func()) {
	t.Helper()

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	original := os.Stdout
	os.Stdout = write

	restore := func() { os.Stdout = original }

	printed := func() string {
		t.Helper()

		if err := write.Close(); err != nil {
			t.Fatalf("close the stdout pipe: %v", err)
		}

		body, err := io.ReadAll(read)
		if err != nil {
			t.Fatalf("read the stdout pipe: %v", err)
		}

		return string(body)
	}

	return printed, restore
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(body)
}
