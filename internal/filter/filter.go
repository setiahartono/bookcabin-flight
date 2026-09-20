package filter

import (
	"strings"

	"bookcabin-flight/internal/provider"
)

type SearchCriteria struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	Passengers    int    `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
	// RoundTrip is request only: a search adds the way back when it is true, and
	// the criteria echoed in a response never carries it.
	RoundTrip *bool `json:"-"`
}

func FilterFlights(flights []provider.FlightData, criteria SearchCriteria) []provider.FlightData {
	filtered := make([]provider.FlightData, 0, len(flights))

	for _, flight := range flights {
		if !matchesCriteria(flight, criteria) {
			continue
		}

		filtered = append(filtered, flight)
	}

	return filtered
}

func matchesCriteria(flight provider.FlightData, criteria SearchCriteria) bool {
	if criteria.Origin != "" && !strings.EqualFold(flight.Departure.Airport, criteria.Origin) {
		return false
	}

	if criteria.Destination != "" && !strings.EqualFold(flight.Arrival.Airport, criteria.Destination) {
		return false
	}

	if criteria.DepartureDate != "" && departureDate(flight) != criteria.DepartureDate {
		return false
	}

	if criteria.Passengers > 0 && flight.AvailableSeats < criteria.Passengers {
		return false
	}

	if criteria.CabinClass != "" && !strings.EqualFold(flight.CabinClass, criteria.CabinClass) {
		return false
	}

	return true
}

func departureDate(flight provider.FlightData) string {
	date, _, _ := strings.Cut(flight.Departure.Datetime, "T")

	return date
}
