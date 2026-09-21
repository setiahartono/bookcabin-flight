package utils

import (
	"fmt"
	"strings"
	"time"
)

var indonesianAirportCities = map[string]string{
	"CGK": "Jakarta",
	"DPS": "Denpasar",
	"SOC": "Solo",
	"SUB": "Surabaya",
	"UPG": "Makassar",
	"DJJ": "Jayapura",
}

type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

func AirportCity(code string) (string, bool) {
	city, ok := indonesianAirportCities[strings.ToUpper(strings.TrimSpace(code))]

	return city, ok
}

// DatetimeToTimestamp converts an RFC 3339 datetime
// For example "2025-12-15T15:15:00+07:00", into its Unix timestamp in seconds.
// Some providers send the offset without a colon, for example "2025-12-15T07:15:00+0700",
// which time.RFC3339 rejects, so that form is accepted as well.
func DatetimeToTimestamp(datetime string) (int64, error) {
	parsed, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		parsed, err = time.Parse("2006-01-02T15:04:05-0700", datetime)
		if err != nil {
			return 0, err
		}
	}

	return parsed.Unix(), nil
}

func FormatDuration(minutes int) string {
	return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
}

func AcquireBaggageInfo(baggageNote string) Baggage {
	parts := strings.SplitN(baggageNote, ",", 2)

	info := Baggage{}

	if len(parts) > 0 {
		info.CarryOn = strings.TrimSpace(parts[0])
	}

	if len(parts) > 1 {
		checked := strings.TrimSpace(parts[1])
		checked = strings.TrimPrefix(checked, "checked bags ")
		info.Checked = strings.TrimSpace(checked)
	}

	return info
}
