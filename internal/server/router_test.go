package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const searchBody = `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy"}`

func postSearch(handler http.Handler) int {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/search", strings.NewReader(searchBody))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder.Code
}

func TestSearchRouteIsRateLimited(t *testing.T) {
	handler := NewRouter()

	tests := []struct {
		name       string
		wantStatus int
	}{
		{name: "first search", wantStatus: http.StatusOK},
		{name: "second search", wantStatus: http.StatusOK},
		{name: "third search", wantStatus: http.StatusOK},
		{name: "fourth search", wantStatus: http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := postSearch(handler); got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
		})
	}
}

func TestPingIsNotRateLimited(t *testing.T) {
	handler := NewRouter()

	for request := 1; request <= 5; request++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

		if got := recorder.Code; got != http.StatusOK {
			t.Fatalf("status = %d on request %d, want %d", got, request, http.StatusOK)
		}
	}
}
