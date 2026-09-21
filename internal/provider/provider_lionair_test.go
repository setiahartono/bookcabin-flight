package provider

import (
	"context"
	"strings"
	"testing"
)

func TestLionAirSearchReturnsEveryRecordedFlight(t *testing.T) {
	tests := []struct {
		id                 string
		origin             string
		destination        string
		durationFormatted  string
		stops              int
		priceIDR           int
		seats              int
		departureTimestamp int64
		arrivalTimestamp   int64
	}{
		{
			id:                 "JT740",
			origin:             "CGK",
			destination:        "DPS",
			durationFormatted:  "1h 45m",
			stops:              0,
			priceIDR:           950000,
			seats:              45,
			departureTimestamp: 1765751400,
			arrivalTimestamp:   1765757700,
		},
		{
			id:                 "JT742",
			origin:             "CGK",
			destination:        "DPS",
			durationFormatted:  "1h 50m",
			stops:              0,
			priceIDR:           890000,
			seats:              38,
			departureTimestamp: 1765773900,
			arrivalTimestamp:   1765780500,
		},
		{
			id:                 "JT650",
			origin:             "CGK",
			destination:        "DPS",
			durationFormatted:  "3h 50m",
			stops:              1,
			priceIDR:           780000,
			seats:              52,
			departureTimestamp: 1765790400,
			arrivalTimestamp:   1765804200,
		},
		{
			id:                 "JT780",
			origin:             "CGK",
			destination:        "UPG",
			durationFormatted:  "2h 15m",
			stops:              0,
			priceIDR:           760000,
			seats:              51,
			departureTimestamp: 1765761600,
			arrivalTimestamp:   1765769700,
		},
		{
			id:                 "JT784",
			origin:             "CGK",
			destination:        "UPG",
			durationFormatted:  "4h 45m",
			stops:              1,
			priceIDR:           640000,
			seats:              38,
			departureTimestamp: 1765780800,
			arrivalTimestamp:   1765797900,
		},
		{
			id:                 "JT920",
			origin:             "CGK",
			destination:        "DJJ",
			durationFormatted:  "4h 55m",
			stops:              0,
			priceIDR:           1420000,
			seats:              29,
			departureTimestamp: 1765754100,
			arrivalTimestamp:   1765771800,
		},
		{
			id:                 "JT924",
			origin:             "CGK",
			destination:        "DJJ",
			durationFormatted:  "5h 55m",
			stops:              1,
			priceIDR:           1280000,
			seats:              34,
			departureTimestamp: 1765787400,
			arrivalTimestamp:   1765808700,
		},
		{
			id:                 "JT741",
			origin:             "DPS",
			destination:        "CGK",
			durationFormatted:  "1h 55m",
			stops:              0,
			priceIDR:           600000,
			seats:              56,
			departureTimestamp: 1766194800,
			arrivalTimestamp:   1766201700,
		},
		{
			id:                 "JT745",
			origin:             "DPS",
			destination:        "CGK",
			durationFormatted:  "4h 25m",
			stops:              1,
			priceIDR:           560000,
			seats:              43,
			departureTimestamp: 1766218800,
			arrivalTimestamp:   1766234700,
		},
		{
			id:                 "JT749",
			origin:             "DPS",
			destination:        "CGK",
			durationFormatted:  "1h 55m",
			stops:              0,
			priceIDR:           615000,
			seats:              49,
			departureTimestamp: 1766358300,
			arrivalTimestamp:   1766365200,
		},
		{
			id:                 "JT781",
			origin:             "UPG",
			destination:        "CGK",
			durationFormatted:  "2h 15m",
			stops:              0,
			priceIDR:           690000,
			seats:              47,
			departureTimestamp: 1766204700,
			arrivalTimestamp:   1766212800,
		},
		{
			id:                 "JT785",
			origin:             "UPG",
			destination:        "CGK",
			durationFormatted:  "4h 25m",
			stops:              1,
			priceIDR:           650000,
			seats:              40,
			departureTimestamp: 1766227800,
			arrivalTimestamp:   1766243700,
		},
		{
			id:                 "JT789",
			origin:             "UPG",
			destination:        "CGK",
			durationFormatted:  "2h 15m",
			stops:              0,
			priceIDR:           705000,
			seats:              44,
			departureTimestamp: 1766369400,
			arrivalTimestamp:   1766377500,
		},
	}

	provider, err := NewLionAir()
	if err != nil {
		t.Fatalf("NewLionAir() error = %v", err)
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

			if got := flight.Id; got != tt.id+"_Lion Air" {
				t.Errorf("Id = %q, want %q", got, tt.id+"_Lion Air")
			}
			if got := flight.Provider; got != "Lion Air" {
				t.Errorf("Provider = %q, want %q", got, "Lion Air")
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
			if got := flight.Departure.Timestamp; got != tt.departureTimestamp {
				t.Errorf("Departure.Timestamp = %d, want %d", got, tt.departureTimestamp)
			}
			if got := flight.Arrival.Timestamp; got != tt.arrivalTimestamp {
				t.Errorf("Arrival.Timestamp = %d, want %d", got, tt.arrivalTimestamp)
			}
		})
	}
}

func TestLionAirNormalizeMapsRecordedFlight(t *testing.T) {
	tests := []struct {
		name               string
		flight             LionFlight
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
			name: "JT740 direct flight",
			flight: LionFlight{
				ID:      "JT740",
				Carrier: LionCarrier{Name: "Lion Air", IATA: "JT"},
				Route: LionRoute{
					From: LionEndpoint{Code: "CGK", City: "Jakarta"},
					To:   LionEndpoint{Code: "DPS", City: "Denpasar"},
				},
				Schedule: LionSchedule{
					Departure:         "2025-12-15T05:30:00",
					DepartureTimezone: "Asia/Jakarta",
					Arrival:           "2025-12-15T08:15:00",
					ArrivalTimezone:   "Asia/Makassar",
				},
				FlightTime: 105,
				IsDirect:   true,
				Pricing:    LionPricing{Total: 950000, Currency: "IDR", FareType: "ECONOMY"},
				SeatsLeft:  45,
				PlaneType:  "Boeing 737-900ER",
				Services: LionServices{
					BaggageAllowance: LionBaggageAllowance{Cabin: "7 kg", Hold: "20 kg"},
				},
			},
			departureCity:      "Jakarta",
			arrivalCity:        "Denpasar",
			departureTimestamp: 1765751400,
			arrivalTimestamp:   1765757700,
			durationMinutes:    105,
			durationFormatted:  "1h 45m",
			stops:              0,
			baggage:            baggage{CarryOn: "7 kg", Checked: "20 kg"},
			cabinClass:         "ECONOMY",
			aircraft:           "Boeing 737-900ER",
		},
		{
			name: "JT650 stop count wins over the layover list",
			flight: LionFlight{
				ID:      "JT650",
				Carrier: LionCarrier{Name: "Lion Air", IATA: "JT"},
				Route: LionRoute{
					From: LionEndpoint{Code: "CGK"},
					To:   LionEndpoint{Code: "DPS"},
				},
				Schedule: LionSchedule{
					Departure:         "2025-12-15T16:20:00",
					DepartureTimezone: "Asia/Jakarta",
					Arrival:           "2025-12-15T21:10:00",
					ArrivalTimezone:   "Asia/Makassar",
				},
				FlightTime: 230,
				IsDirect:   false,
				StopCount:  1,
				Layovers:   []LionLayover{{Airport: "SUB", DurationMinutes: 75}},
				Pricing:    LionPricing{Total: 780000, Currency: "IDR", FareType: "ECONOMY"},
				SeatsLeft:  52,
				PlaneType:  "Boeing 737-800",
				Services: LionServices{
					BaggageAllowance: LionBaggageAllowance{Cabin: "7 kg", Hold: "20 kg"},
				},
			},
			departureCity:      "Jakarta",
			arrivalCity:        "Denpasar",
			departureTimestamp: 1765790400,
			arrivalTimestamp:   1765804200,
			durationMinutes:    230,
			durationFormatted:  "3h 50m",
			stops:              1,
			baggage:            baggage{CarryOn: "7 kg", Checked: "20 kg"},
			cabinClass:         "ECONOMY",
			aircraft:           "Boeing 737-800",
		},

		{
			name: "layover list without a stop count",
			flight: LionFlight{
				ID:      "JT652",
				Carrier: LionCarrier{Name: "Lion Air", IATA: "JT"},
				Route: LionRoute{
					From: LionEndpoint{Code: "CGK"},
					To:   LionEndpoint{Code: "UPG"},
				},
				Schedule: LionSchedule{
					Departure:         "2025-12-15T05:30:00",
					DepartureTimezone: "Asia/Jakarta",
					Arrival:           "2025-12-15T08:15:00",
					ArrivalTimezone:   "Asia/Makassar",
				},
				FlightTime: 105,
				Layovers:   []LionLayover{{Airport: "SUB"}, {Airport: "DPS"}},
				Pricing:    LionPricing{Total: 950000, Currency: "IDR", FareType: "ECONOMY"},
				Services: LionServices{
					BaggageAllowance: LionBaggageAllowance{Cabin: "7 kg", Hold: "20 kg"},
				},
			},
			departureCity:      "Jakarta",
			arrivalCity:        "Makassar",
			departureTimestamp: 1765751400,
			arrivalTimestamp:   1765757700,
			durationMinutes:    105,
			durationFormatted:  "1h 45m",
			stops:              2,
			baggage:            baggage{CarryOn: "7 kg", Checked: "20 kg"},
			cabinClass:         "ECONOMY",
		},
		{
			name: "unknown airport and unknown timezone",
			flight: LionFlight{
				ID:      "JT999",
				Carrier: LionCarrier{Name: "Lion Air", IATA: "JT"},
				Route: LionRoute{
					From: LionEndpoint{Code: "XXX"},
					To:   LionEndpoint{Code: "DPS"},
				},
				Schedule: LionSchedule{
					Departure:         "2025-12-15T05:30:00",
					DepartureTimezone: "Mars/Olympus",
					Arrival:           "2025-12-15T08:15:00",
					ArrivalTimezone:   "Mars/Olympus",
				},
				FlightTime: 105,
				Pricing:    LionPricing{Total: 950000, Currency: "IDR", FareType: "ECONOMY"},
				PlaneType:  "Boeing 737-800",
				Services: LionServices{
					WifiAvailable:    true,
					MealsIncluded:    true,
					BaggageAllowance: LionBaggageAllowance{Cabin: "Cabin baggage only", Hold: "additional fee"},
				},
			},
			departureCity:     "UNKNOWN",
			arrivalCity:       "Denpasar",
			durationMinutes:   105,
			durationFormatted: "1h 45m",
			amenities:         "WiFi, Meals",
			baggage:           baggage{CarryOn: "Cabin baggage only", Checked: "additional fee"},
			cabinClass:        "ECONOMY",
			aircraft:          "Boeing 737-800",
		},
		{
			name: "datetime that already carries its offset",
			flight: LionFlight{
				ID:      "JT741",
				Carrier: LionCarrier{Name: "Lion Air", IATA: "JT"},
				Route: LionRoute{
					From: LionEndpoint{Code: "CGK"},
					To:   LionEndpoint{Code: "DPS"},
				},
				Schedule: LionSchedule{
					Departure: "2025-12-15T05:30:00+07:00",
					Arrival:   "2025-12-15T08:15:00+08:00",
				},
				FlightTime: 105,
				Pricing:    LionPricing{Total: 950000, Currency: "IDR", FareType: "ECONOMY"},
				Services: LionServices{
					BaggageAllowance: LionBaggageAllowance{Cabin: "7 kg", Hold: "20 kg"},
				},
			},
			departureCity:      "Jakarta",
			arrivalCity:        "Denpasar",
			departureTimestamp: 1765751400,
			arrivalTimestamp:   1765757700,
			durationMinutes:    105,
			durationFormatted:  "1h 45m",
			baggage:            baggage{CarryOn: "7 kg", Checked: "20 kg"},
			cabinClass:         "ECONOMY",
		},
	}

	provider, err := NewLionAir()
	if err != nil {
		t.Fatalf("NewLionAir() error = %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flights := provider.Normalize(&LionResponse{Success: true, Data: LionData{AvailableFlights: []LionFlight{tt.flight}}})
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

			check("Id", got.Id, tt.flight.ID+"_Lion Air")
			check("Provider", got.Provider, "Lion Air")
			check("Airline.Name", got.Airline.Name, tt.flight.Carrier.Name)
			check("Airline.Code", got.Airline.Code, tt.flight.Carrier.IATA)
			check("FlightNumber", got.FlightNumber, tt.flight.ID)
			check("Departure.Airport", got.Departure.Airport, tt.flight.Route.From.Code)
			check("Departure.City", got.Departure.City, tt.departureCity)
			check("Departure.Datetime", got.Departure.Datetime, tt.flight.Schedule.Departure)
			check("Departure.Timestamp", got.Departure.Timestamp, tt.departureTimestamp)
			check("Arrival.Airport", got.Arrival.Airport, tt.flight.Route.To.Code)
			check("Arrival.City", got.Arrival.City, tt.arrivalCity)
			check("Arrival.Datetime", got.Arrival.Datetime, tt.flight.Schedule.Arrival)
			check("Arrival.Timestamp", got.Arrival.Timestamp, tt.arrivalTimestamp)
			check("Duration.TotalMinutes", got.Duration.TotalMinutes, tt.durationMinutes)
			check("Duration.Formatted", got.Duration.Formatted, tt.durationFormatted)
			check("Stops", got.Stops, tt.stops)
			check("Price.Amount", got.Price.Amount, tt.flight.Pricing.Total)
			check("Price.Currency", got.Price.Currency, tt.flight.Pricing.Currency)
			check("AvailableSeats", got.AvailableSeats, tt.flight.SeatsLeft)
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
