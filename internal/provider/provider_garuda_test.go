package provider

import (
	"context"
	"strings"
	"testing"
)

func TestGarudaSearchReturnsEveryRecordedFlight(t *testing.T) {
	tests := []struct {
		id                 string
		origin             string
		destination        string
		durationFormatted  string
		stops              int
		priceIDR           int
		seats              int
		amenities          int
		departureTimestamp int64
		arrivalTimestamp   int64
	}{
		{
			id:                 "GA400",
			origin:             "CGK",
			destination:        "DPS",
			durationFormatted:  "1h 50m",
			priceIDR:           1250000,
			seats:              28,
			amenities:          3,
			departureTimestamp: 1765753200,
			arrivalTimestamp:   1765759800,
		},
		{
			id:                 "GA410",
			origin:             "CGK",
			destination:        "DPS",
			durationFormatted:  "1h 55m",
			priceIDR:           1450000,
			seats:              15,
			amenities:          4,
			departureTimestamp: 1765765800,
			arrivalTimestamp:   1765772700,
		},
		{
			id:                 "GA315",
			origin:             "CGK",
			destination:        "SUB",
			durationFormatted:  "1h 30m",
			priceIDR:           1850000,
			seats:              22,
			departureTimestamp: 1765782000,
			arrivalTimestamp:   1765787400,
		}, {
			id:                 "GA512",
			origin:             "CGK",
			destination:        "UPG",
			durationFormatted:  "2h 15m",
			stops:              0,
			priceIDR:           1380000,
			seats:              24,
			amenities:          3,
			departureTimestamp: 1765761600,
			arrivalTimestamp:   1765769700,
		},
		{
			id:                 "GA516",
			origin:             "CGK",
			destination:        "UPG",
			durationFormatted:  "4h 45m",
			stops:              1,
			priceIDR:           1180000,
			seats:              19,
			amenities:          2,
			departureTimestamp: 1765780800,
			arrivalTimestamp:   1765797900,
		},
		{
			id:                 "GA652",
			origin:             "CGK",
			destination:        "DJJ",
			durationFormatted:  "4h 55m",
			stops:              0,
			priceIDR:           2150000,
			seats:              16,
			amenities:          3,
			departureTimestamp: 1765754100,
			arrivalTimestamp:   1765771800,
		},
		{
			id:                 "GA656",
			origin:             "CGK",
			destination:        "DJJ",
			durationFormatted:  "5h 55m",
			stops:              1,
			priceIDR:           1890000,
			seats:              21,
			amenities:          1,
			departureTimestamp: 1765787400,
			arrivalTimestamp:   1765808700,
		},
		{
			id:                 "GA401",
			origin:             "DPS",
			destination:        "CGK",
			durationFormatted:  "1h 55m",
			stops:              0,
			priceIDR:           1120000,
			seats:              23,
			amenities:          2,
			departureTimestamp: 1766194800,
			arrivalTimestamp:   1766201700,
		},
		{
			id:                 "GA405",
			origin:             "DPS",
			destination:        "CGK",
			durationFormatted:  "4h 25m",
			stops:              1,
			priceIDR:           980000,
			seats:              17,
			amenities:          1,
			departureTimestamp: 1766218800,
			arrivalTimestamp:   1766234700,
		},
		{
			id:                 "GA409",
			origin:             "DPS",
			destination:        "CGK",
			durationFormatted:  "1h 55m",
			stops:              0,
			priceIDR:           1150000,
			seats:              26,
			amenities:          2,
			departureTimestamp: 1766358300,
			arrivalTimestamp:   1766365200,
		},
		{
			id:                 "GA513",
			origin:             "UPG",
			destination:        "CGK",
			durationFormatted:  "2h 15m",
			stops:              0,
			priceIDR:           1250000,
			seats:              20,
			amenities:          3,
			departureTimestamp: 1766204700,
			arrivalTimestamp:   1766212800,
		},
		{
			id:                 "GA517",
			origin:             "UPG",
			destination:        "CGK",
			durationFormatted:  "4h 25m",
			stops:              1,
			priceIDR:           1050000,
			seats:              15,
			amenities:          1,
			departureTimestamp: 1766227800,
			arrivalTimestamp:   1766243700,
		},
		{
			id:                 "GA521",
			origin:             "UPG",
			destination:        "CGK",
			durationFormatted:  "2h 15m",
			stops:              0,
			priceIDR:           1290000,
			seats:              27,
			amenities:          2,
			departureTimestamp: 1766369400,
			arrivalTimestamp:   1766377500,
		},
	}

	provider, err := NewGaruda()
	if err != nil {
		t.Fatalf("NewGaruda() error = %v", err)
	}

	flights, _, err := provider.Search(context.Background(), SearchRequest{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got, want := len(flights), len(tests); got != want {
		t.Fatalf("len(Search()) = %d, want %d", got, want)
	}

	byID := make(map[string]FlightData, len(flights))
	for _, flight := range flights {
		byID[flight.FlightNumber] = flight
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			flight, ok := byID[tt.id]
			if !ok {
				t.Fatalf("Search() answered no flight %q", tt.id)
			}

			if got := flight.Id; got != tt.id+"_Garuda Indonesia" {
				t.Errorf("Id = %q, want %q", got, tt.id+"_Garuda Indonesia")
			}
			if got := flight.Provider; got != "Garuda Indonesia" {
				t.Errorf("Provider = %q, want %q", got, "Garuda Indonesia")
			}
			if got := flight.FlightNumber; got != tt.id {
				t.Errorf("FlightNumber = %q, want %q", got, tt.id)
			}
			if got := flight.Departure.Airport; got != tt.origin {
				t.Errorf("Departure.Airport = %q, want %q", got, tt.origin)
			}
			if got := flight.Arrival.Airport; got != tt.destination {
				t.Errorf("Arrival.Airport = %q, want %q", got, tt.destination)
			}
			if got := flight.Duration.Formatted; got != tt.durationFormatted {
				t.Errorf("Duration.Formatted = %q, want %q", got, tt.durationFormatted)
			}
			if got := flight.Stops; got != tt.stops {
				t.Errorf("Stops = %d, want %d", got, tt.stops)
			}
			if got := flight.Price.Amount; got != tt.priceIDR {
				t.Errorf("Price.Amount = %d, want %d", got, tt.priceIDR)
			}
			if got := flight.AvailableSeats; got != tt.seats {
				t.Errorf("AvailableSeats = %d, want %d", got, tt.seats)
			}
			if got := len(flight.Amenities); got != tt.amenities {
				t.Errorf("len(Amenities) = %d, want %d", got, tt.amenities)
			}
			if got := flight.Departure.Timestamp; got != tt.departureTimestamp {
				t.Errorf("Departure.Timestamp = %d, want %d", got, tt.departureTimestamp)
			}
			if got := flight.Arrival.Timestamp; got != tt.arrivalTimestamp {
				t.Errorf("Arrival.Timestamp = %d, want %d", got, tt.arrivalTimestamp)
			}
			if got := flight.Baggage; got != (baggage{CarryOn: "1 piece", Checked: "2 pieces"}) {
				t.Errorf("Baggage = %+v, want 1 piece and 2 pieces", got)
			}
		})
	}
}

func TestGarudaNormalizeMapsRecordedFlight(t *testing.T) {
	tests := []struct {
		name               string
		flight             GarudaFlight
		departureCity      string
		arrivalCity        string
		departureTimestamp int64
		arrivalTimestamp   int64
		durationMinutes    int
		durationFormatted  string
		stops              int
		amenities          string
		baggage            baggage
		cabinClass         string
		aircraft           string
	}{
		{
			name: "GA400 direct flight",
			flight: GarudaFlight{
				FlightID:        "GA400",
				Airline:         "Garuda Indonesia",
				AirlineCode:     "GA",
				Departure:       GarudaEndpoint{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T06:00:00+07:00", Terminal: "3"},
				Arrival:         GarudaEndpoint{Airport: "DPS", City: "Denpasar", Time: "2025-12-15T08:50:00+08:00", Terminal: "I"},
				Price:           GarudaPrice{Amount: 1250000, Currency: "IDR"},
				Aircraft:        "Boeing 737-800",
				Baggage:         GarudaBaggage{CarryOn: 1, Checked: 2},
				Amenities:       []string{"wifi", "meal", "entertainment"},
				FareClass:       "economy",
				DurationMinutes: 110,
				AvailableSeats:  28,
			},
			departureCity:      "Jakarta",
			arrivalCity:        "Denpasar",
			departureTimestamp: 1765753200,
			arrivalTimestamp:   1765759800,
			durationMinutes:    110,
			durationFormatted:  "1h 50m",
			stops:              0,
			amenities:          "wifi, meal, entertainment",
			baggage:            baggage{CarryOn: "1 piece", Checked: "2 pieces"},
			cabinClass:         "economy",
			aircraft:           "Boeing 737-800",
		},
		{
			name: "GA315 connecting itinerary without amenities",
			flight: GarudaFlight{
				FlightID:        "GA315",
				Airline:         "Garuda Indonesia",
				AirlineCode:     "GA",
				Departure:       GarudaEndpoint{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T14:00:00+07:00", Terminal: "3"},
				Arrival:         GarudaEndpoint{Airport: "SUB", City: "Surabaya", Time: "2025-12-15T15:30:00+07:00", Terminal: "2"},
				Price:           GarudaPrice{Amount: 1850000, Currency: "IDR"},
				Aircraft:        "Boeing 737",
				Baggage:         GarudaBaggage{CarryOn: 1, Checked: 2},
				FareClass:       "economy",
				DurationMinutes: 90,
				AvailableSeats:  22,
				Segments: []GarudaSegment{
					{FlightNumber: "GA315", DurationMinutes: 90},
					{FlightNumber: "GA332", DurationMinutes: 90, LayoverMinutes: 105},
				},
			},
			departureCity:      "Jakarta",
			arrivalCity:        "Surabaya",
			departureTimestamp: 1765782000,
			arrivalTimestamp:   1765787400,
			durationMinutes:    90,
			durationFormatted:  "1h 30m",
			stops:              0,
			baggage:            baggage{CarryOn: "1 piece", Checked: "2 pieces"},
			cabinClass:         "economy",
			aircraft:           "Boeing 737",
		},

		{
			name: "unknown airport code",
			flight: GarudaFlight{
				FlightID:        "GA800",
				Airline:         "Garuda Indonesia",
				AirlineCode:     "GA",
				Departure:       GarudaEndpoint{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T14:00:00+07:00", Terminal: "3"},
				Arrival:         GarudaEndpoint{Airport: "XXX", Time: "2025-12-15T18:45:00+08:00"},
				Price:           GarudaPrice{Amount: 2100000, Currency: "IDR"},
				Aircraft:        "Airbus A330-300",
				Amenities:       []string{"wifi"},
				FareClass:       "economy",
				DurationMinutes: 285,
				Stops:           1,
				AvailableSeats:  9,
			},
			departureCity:      "Jakarta",
			arrivalCity:        "UNKNOWN",
			departureTimestamp: 1765782000,
			arrivalTimestamp:   1765795500,
			durationMinutes:    285,
			durationFormatted:  "4h 45m",
			stops:              1,
			amenities:          "wifi",
			cabinClass:         "economy",
			aircraft:           "Airbus A330-300",
		},
		{
			name: "datetime that does not parse",
			flight: GarudaFlight{
				FlightID:        "GA999",
				Airline:         "Garuda Indonesia",
				AirlineCode:     "GA",
				Departure:       GarudaEndpoint{Airport: "CGK", Time: "15-12-2025 14:00"},
				Arrival:         GarudaEndpoint{Airport: "DPS", Time: "15-12-2025 16:00"},
				DurationMinutes: 120,
			},
			departureCity:     "Jakarta",
			arrivalCity:       "Denpasar",
			durationMinutes:   120,
			durationFormatted: "2h 0m",
		},
	}

	provider, err := NewGaruda()
	if err != nil {
		t.Fatalf("NewGaruda() error = %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flights := provider.Normalize(&GarudaResponse{Status: "success", Flights: []GarudaFlight{tt.flight}})
			if len(flights) != 1 {
				t.Fatalf("len(Normalize()) = %d, want 1", len(flights))
			}

			got := flights[0]

			check := func(name string, got, want any) {
				t.Helper()

				if got != want {
					t.Errorf("%s = %v, want %v", name, got, want)
				}
			}

			check("Id", got.Id, tt.flight.FlightID+"_Garuda Indonesia")
			check("Provider", got.Provider, "Garuda Indonesia")
			check("Airline.Name", got.Airline.Name, tt.flight.Airline)
			check("Airline.Code", got.Airline.Code, tt.flight.AirlineCode)
			check("FlightNumber", got.FlightNumber, tt.flight.FlightID)
			check("Departure.Airport", got.Departure.Airport, tt.flight.Departure.Airport)
			check("Departure.City", got.Departure.City, tt.departureCity)
			check("Departure.Datetime", got.Departure.Datetime, tt.flight.Departure.Time)
			check("Departure.Timestamp", got.Departure.Timestamp, tt.departureTimestamp)
			check("Arrival.Airport", got.Arrival.Airport, tt.flight.Arrival.Airport)
			check("Arrival.City", got.Arrival.City, tt.arrivalCity)
			check("Arrival.Datetime", got.Arrival.Datetime, tt.flight.Arrival.Time)
			check("Arrival.Timestamp", got.Arrival.Timestamp, tt.arrivalTimestamp)
			check("Duration.TotalMinutes", got.Duration.TotalMinutes, tt.durationMinutes)
			check("Duration.Formatted", got.Duration.Formatted, tt.durationFormatted)
			check("Stops", got.Stops, tt.stops)
			check("Price.Amount", got.Price.Amount, tt.flight.Price.Amount)
			check("Price.Currency", got.Price.Currency, tt.flight.Price.Currency)
			check("AvailableSeats", got.AvailableSeats, tt.flight.AvailableSeats)
			check("CabinClass", got.CabinClass, tt.cabinClass)
			check("Amenities", strings.Join(got.Amenities, ", "), tt.amenities)
			check("Baggage", got.Baggage, tt.baggage)

			if got.Aircraft == nil {
				t.Fatalf("Aircraft = nil, want %q", tt.aircraft)
			}
			check("Aircraft", *got.Aircraft, tt.aircraft)
		})
	}
}
