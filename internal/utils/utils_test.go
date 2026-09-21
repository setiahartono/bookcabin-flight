package utils

import "testing"

func TestAirportCity(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		wantCity string
		wantOK   bool
	}{
		{name: "CGK", code: "CGK", wantCity: "Jakarta", wantOK: true},
		{name: "DPS", code: "DPS", wantCity: "Denpasar", wantOK: true},
		{name: "SUB", code: "SUB", wantCity: "Surabaya", wantOK: true},
		{name: "DJJ", code: "DJJ", wantCity: "Jayapura", wantOK: true},
		{name: "lower case", code: "cgk", wantCity: "Jakarta", wantOK: true},
		{name: "unknown code", code: "XXX", wantCity: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			city, ok := AirportCity(tt.code)

			if city != tt.wantCity || ok != tt.wantOK {
				t.Errorf("AirportCity(%q) = (%q, %v), want (%q, %v)", tt.code, city, ok, tt.wantCity, tt.wantOK)
			}
		})
	}
}

func TestDatetimeToTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		datetime string
		want     int64
	}{
		{name: "Jakarta offset", datetime: "2025-12-15T15:15:00+07:00", want: 1765786500},
		{name: "Makassar offset", datetime: "2025-12-15T20:35:00+08:00", want: 1765802100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DatetimeToTimestamp(tt.datetime)
			if err != nil {
				t.Fatalf("DatetimeToTimestamp(%q) error = %v", tt.datetime, err)
			}

			if got != tt.want {
				t.Errorf("DatetimeToTimestamp(%q) = %d, want %d", tt.datetime, got, tt.want)
			}
		})
	}
}

func TestDatetimeToTimestampInvalid(t *testing.T) {
	if _, err := DatetimeToTimestamp("15-12-2025 15:15"); err == nil {
		t.Error(`DatetimeToTimestamp("15-12-2025 15:15") error = nil, want a parse error`)
	}
}
