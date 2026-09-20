package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"bookcabin-flight/internal/utils"
)

func TestAirAsiaSearchReturnsEveryRecordedFlight(t *testing.T) {
	tests := []struct {
		index        int
		flightCode   string
		fromAirport  string
		toAirport    string
		directFlight bool
		priceIDR     int
		seats        int
		stops        []AirAsiaStop
	}{
		{
			index:        0,
			flightCode:   "QZ520",
			fromAirport:  "CGK",
			toAirport:    "DPS",
			directFlight: true,
			priceIDR:     650000,
			seats:        67,
		},
		{
			index:        1,
			flightCode:   "QZ524",
			fromAirport:  "CGK",
			toAirport:    "DPS",
			directFlight: true,
			priceIDR:     720000,
			seats:        54,
		},
		{
			index:        2,
			flightCode:   "QZ532",
			fromAirport:  "CGK",
			toAirport:    "DPS",
			directFlight: true,
			priceIDR:     595000,
			seats:        72,
		},
		{
			index:       3,
			flightCode:  "QZ7250",
			fromAirport: "CGK",
			toAirport:   "DPS",
			priceIDR:    485000,
			seats:       88,
			stops:       []AirAsiaStop{{Airport: "SOC", WaitTimeMinutes: 95}},
		},
	}

	provider, err := NewAirAsia()
	if err != nil {
		t.Fatalf("NewAirAsia() error = %v", err)
	}

	var flights []FlightData

	for range 10 {
		flights, err = provider.Search(context.Background(), SearchRequest{})
		if err == nil {
			break
		}
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Search() error = %v, want nil or ErrUnavailable", err)
		}
	}
	if err != nil {
		t.Fatalf("Search() error = %v after 10 attempts", err)
	}

	if got, want := len(flights), len(tests); got != want {
		t.Fatalf("len(Search()) = %d, want %d", got, want)
	}

	for _, tt := range tests {
		t.Run(tt.flightCode, func(t *testing.T) {
			flight := flights[tt.index]

			if got := flight.Id; got != tt.flightCode+"_AirAsia" {
				t.Errorf("Id = %q, want %q", got, tt.flightCode+"_AirAsia")
			}
			if got := flight.FlightNumber; got != tt.flightCode {
				t.Errorf("FlightNumber = %q, want %q", got, tt.flightCode)
			}
			if got := flight.Departure.Airport; got != tt.fromAirport {
				t.Errorf("Departure.Airport = %q, want %q", got, tt.fromAirport)
			}
			if got := flight.Arrival.Airport; got != tt.toAirport {
				t.Errorf("Arrival.Airport = %q, want %q", got, tt.toAirport)
			}
			if got := flight.Price.Amount; got != tt.priceIDR {
				t.Errorf("Price.Amount = %d, want %d", got, tt.priceIDR)
			}
			if got := flight.AvailableSeats; got != tt.seats {
				t.Errorf("AvailableSeats = %d, want %d", got, tt.seats)
			}
			if got := flight.Stops; got != len(tt.stops) {
				t.Errorf("Stops = %d, want %d", got, len(tt.stops))
			}
			if tt.directFlight && flight.Stops != 0 {
				t.Errorf("Stops = %d, want 0 for a direct flight", flight.Stops)
			}
		})
	}
}

func TestAirAsiaNormalizeMapsRecordedFlight(t *testing.T) {
	tests := []struct {
		name              string
		flight            AirAsiaFlight
		departureCity     string
		arrivalCity       string
		durationMinutes   int
		durationFormatted string
		stops             int
	}{
		{
			name: "QZ520",
			flight: AirAsiaFlight{
				FlightCode:    "QZ520",
				Airline:       "AirAsia",
				FromAirport:   "CGK",
				ToAirport:     "DPS",
				DepartTime:    "2025-12-15T04:45:00+07:00",
				ArriveTime:    "2025-12-15T07:25:00+08:00",
				DurationHours: 1.67,
				DirectFlight:  true,
				PriceIDR:      650000,
				Seats:         67,
				CabinClass:    "economy",
				BaggageNote:   "Cabin baggage only, checked bags additional fee",
			},
			departureCity:     "Jakarta",
			arrivalCity:       "Denpasar",
			durationMinutes:   100,
			durationFormatted: "1h 40m",
		},
		{
			name: "QZ524",
			flight: AirAsiaFlight{
				FlightCode:    "QZ524",
				Airline:       "AirAsia",
				FromAirport:   "CGK",
				ToAirport:     "DPS",
				DepartTime:    "2025-12-15T10:00:00+07:00",
				ArriveTime:    "2025-12-15T12:45:00+08:00",
				DurationHours: 1.75,
				DirectFlight:  true,
				PriceIDR:      720000,
				Seats:         54,
				CabinClass:    "economy",
				BaggageNote:   "Cabin baggage only, checked bags additional fee",
			},
			departureCity:     "Jakarta",
			arrivalCity:       "Denpasar",
			durationMinutes:   105,
			durationFormatted: "1h 45m",
		},
		{
			name: "QZ532",
			flight: AirAsiaFlight{
				FlightCode:    "QZ532",
				Airline:       "AirAsia",
				FromAirport:   "CGK",
				ToAirport:     "DPS",
				DepartTime:    "2025-12-15T19:30:00+07:00",
				ArriveTime:    "2025-12-15T22:10:00+08:00",
				DurationHours: 1.67,
				DirectFlight:  true,
				PriceIDR:      595000,
				Seats:         72,
				CabinClass:    "economy",
				BaggageNote:   "Cabin baggage only, checked bags additional fee",
			},
			departureCity:     "Jakarta",
			arrivalCity:       "Denpasar",
			durationMinutes:   100,
			durationFormatted: "1h 40m",
		},
		{
			name: "QZ7250",
			flight: AirAsiaFlight{
				FlightCode:    "QZ7250",
				Airline:       "AirAsia",
				FromAirport:   "CGK",
				ToAirport:     "DPS",
				DepartTime:    "2025-12-15T15:15:00+07:00",
				ArriveTime:    "2025-12-15T20:35:00+08:00",
				DurationHours: 4.33,
				Stops:         []AirAsiaStop{{Airport: "SOC", WaitTimeMinutes: 95}},
				PriceIDR:      485000,
				Seats:         88,
				CabinClass:    "economy",
				BaggageNote:   "Cabin baggage only, checked bags additional fee",
			},
			departureCity:     "Jakarta",
			arrivalCity:       "Denpasar",
			durationMinutes:   260,
			durationFormatted: "4h 20m",
			stops:             1,
		},
	}

	provider, err := NewAirAsia()
	if err != nil {
		t.Fatalf("NewAirAsia() error = %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flights := provider.Normalize(&AirAsiaResponse{Status: "ok", Flights: []AirAsiaFlight{tt.flight}})
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

			check("Id", got.Id, tt.flight.FlightCode+"_AirAsia")
			check("Provider", got.Provider, "AirAsia")
			check("Airline.Name", got.Airline.Name, tt.flight.Airline)
			check("Airline.Code", got.Airline.Code, "QZ")
			check("FlightNumber", got.FlightNumber, tt.flight.FlightCode)
			check("Departure.Airport", got.Departure.Airport, tt.flight.FromAirport)
			check("Departure.City", got.Departure.City, tt.departureCity)
			check("Departure.Datetime", got.Departure.Datetime, tt.flight.DepartTime)
			check("Arrival.Airport", got.Arrival.Airport, tt.flight.ToAirport)
			check("Arrival.City", got.Arrival.City, tt.arrivalCity)
			check("Arrival.Datetime", got.Arrival.Datetime, tt.flight.ArriveTime)
			check("Duration.TotalMinutes", got.Duration.TotalMinutes, tt.durationMinutes)
			check("Duration.Formatted", got.Duration.Formatted, tt.durationFormatted)
			check("Stops", got.Stops, tt.stops)
			check("Price.Amount", got.Price.Amount, tt.flight.PriceIDR)
			check("Price.Currency", got.Price.Currency, "IDR")
			check("AvailableSeats", got.AvailableSeats, tt.flight.Seats)
			check("CabinClass", got.CabinClass, tt.flight.CabinClass)
			check("len(Amenities)", len(got.Amenities), 0)
			check("Baggage.CarryOn", got.Baggage.CarryOn, "Cabin baggage only")
			check("Baggage.Checked", got.Baggage.Checked, "additional fee")

			departure, err := time.Parse(time.RFC3339, got.Departure.Datetime)
			if err != nil {
				t.Fatalf("Departure.Datetime %q: %v", got.Departure.Datetime, err)
			}
			check("Departure.Timestamp", got.Departure.Timestamp, departure.Unix())

			arrival, err := time.Parse(time.RFC3339, got.Arrival.Datetime)
			if err != nil {
				t.Fatalf("Arrival.Datetime %q: %v", got.Arrival.Datetime, err)
			}
			check("Arrival.Timestamp", got.Arrival.Timestamp, arrival.Unix())
		})
	}
}

func TestAcquireBaggageInfo(t *testing.T) {
	tests := []struct {
		name        string
		baggageNote string
		want        baggage
	}{
		{
			name:        "carry on and checked allowance",
			baggageNote: "Cabin baggage only, checked bags additional fee",
			want:        baggage{CarryOn: "Cabin baggage only", Checked: "additional fee"},
		},
		{
			name:        "carry on only",
			baggageNote: "Cabin baggage only",
			want:        baggage{CarryOn: "Cabin baggage only"},
		},
		{
			name:        "empty note",
			baggageNote: "",
			want:        baggage{},
		},
		{
			name:        "checked part without the prefix",
			baggageNote: "7kg cabin, 20kg hold",
			want:        baggage{CarryOn: "7kg cabin", Checked: "20kg hold"},
		},
		{
			name:        "extra whitespace",
			baggageNote: "  7kg cabin , checked bags   20kg  ",
			want:        baggage{CarryOn: "7kg cabin", Checked: "20kg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.AcquireBaggageInfo(tt.baggageNote)

			if got != tt.want {
				t.Errorf("acquireBaggageInfo(%q) = %+v, want %+v", tt.baggageNote, got, tt.want)
			}
		})
	}
}
