// Package logging writes an access log to stdout and to a file, and the errors a
// request answers with to a file of their own.
package logging

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AccessLogName, ErrorLogName and ProviderLogName are the files Open keeps in its
// directory.
const (
	AccessLogName   = "access.log"
	ErrorLogName    = "error.log"
	ProviderLogName = "provider.log"
)

// Logger writes what a request did, what it answered with, and which providers
// could not answer.
type Logger struct {
	access    *log.Logger
	errors    *log.Logger
	providers *log.Logger

	files []io.Closer
}

// New writes the access log to access, the error log to errors, and the providers
// that failed a search to providers.
func New(access, errors, providers io.Writer) *Logger {
	return &Logger{
		access:    log.New(access, "", log.LstdFlags),
		errors:    log.New(errors, "", log.LstdFlags),
		providers: log.New(providers, "", log.LstdFlags),
	}
}

// Open prepares the logs of a directory: the access log is written to stdout and
// to access.log, the error log to error.log, and the providers that failed a
// search to provider.log. Close closes the files.
func Open(dir string) (*Logger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	access, err := openFile(filepath.Join(dir, AccessLogName))
	if err != nil {
		return nil, err
	}

	errors, err := openFile(filepath.Join(dir, ErrorLogName))
	if err != nil {
		_ = access.Close()

		return nil, err
	}

	providers, err := openFile(filepath.Join(dir, ProviderLogName))
	if err != nil {
		_ = access.Close()
		_ = errors.Close()

		return nil, err
	}

	logger := New(io.MultiWriter(os.Stdout, access), errors, providers)
	logger.files = []io.Closer{access, errors, providers}

	return logger, nil
}

func openFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}

// Close closes the files Open opened.
func (l *Logger) Close() error {
	closed := make([]error, 0, len(l.files))

	for _, file := range l.files {
		closed = append(closed, file.Close())
	}

	return errors.Join(closed...)
}

// Middleware logs every request it passes on. The access log records what the
// request was and what it answered, and a request that failed is written to the
// error log as well, with the body it answered with.
func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		response := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(response, r)

		status := response.statusCode()
		duration := time.Since(start).Milliseconds()

		l.access.Printf("remote=%s method=%s path=%s status=%d bytes=%d duration_ms=%d",
			remote(r), r.Method, r.URL.Path, status, response.bytes, duration)

		if status < http.StatusBadRequest {
			return
		}

		l.errors.Printf("remote=%s method=%s path=%s status=%d duration_ms=%d error=%q",
			remote(r), r.Method, r.URL.Path, status, duration, strings.TrimSpace(response.body.String()))
	})
}

// ProviderFailed writes a provider that could not answer a search to the provider
// log.
func (l *Logger) ProviderFailed(provider, origin, destination string, departureDate time.Time, err error) {
	l.providers.Printf("provider=%s route=%s-%s departure_date=%s error=%q",
		provider, origin, destination, departureDate.Format(time.DateOnly), err.Error())
}

// responseWriter remembers the status and the size of what a handler answered, and
// keeps the body of an answer that failed, which is the one worth reporting.
type responseWriter struct {
	http.ResponseWriter

	status int
	bytes  int
	body   bytes.Buffer
}

func (r *responseWriter) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}

	r.ResponseWriter.WriteHeader(status)
}

func (r *responseWriter) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	if r.status >= http.StatusBadRequest {
		r.body.Write(body)
	}

	written, err := r.ResponseWriter.Write(body)
	r.bytes += written

	return written, err
}

// Unwrap hands the writer underneath to whatever asks for it, the way
// http.ResponseController expects.
func (r *responseWriter) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// statusCode is what the request answered with, 200 when a handler wrote a body
// without saying so.
func (r *responseWriter) statusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}

	return r.status
}

// remote names who a request came from: the address a proxy forwarded, or the
// address of the connection when there is no proxy in front.
func remote(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		address, _, _ := strings.Cut(forwarded, ",")

		return strings.TrimSpace(address)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
