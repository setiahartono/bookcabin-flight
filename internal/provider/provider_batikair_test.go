package provider

import (
	"context"
	"errors"
	"testing"
)

func TestBatikAirNormalizeMapsRecordedFlights(t *testing.T) {
	tests := []struct {
		index             int
		id                string
		flightNumber      string
		departureCity     string
		arrivalCity       string
		departureEpoch    int64
		arrivalEpoch      int64
		durationMinutes   int
		durationFormatted string
		stops             int
		totalPrice        int
		seats             int
		aircraft          string
		amenities         int
	}{
		{
			index:             0,
			id:                "ID6514_Batik Air",
			flightNumber:      "ID6514",
			departureCity:     "Jakarta",
			arrivalCity:       "Denpasar",
			departureEpoch:    1765757700,
			arrivalEpoch:      1765764000,
			durationMinutes:   105,
			durationFormatted: "1h 45m",
			totalPrice:        1100000,
			seats:             32,
			aircraft:          "Airbus A320",
			amenities:         2,
		},
		{
			index:             1,
			id:                "ID6520_Batik Air",
			flightNumber:      "ID6520",
			departureCity:     "Jakarta",
			arrivalCity:       "Denpasar",
			departureEpoch:    1765780200,
			arrivalEpoch:      1765786800,
			durationMinutes:   110,
			durationFormatted: "1h 50m",
			totalPrice:        1180000,
			seats:             18,
			aircraft:          "Boeing 737-800",
			amenities:         3,
		},
		{
			index:             2,
			id:                "ID7042_Batik Air",
			flightNumber:      "ID7042",
			departureCity:     "Jakarta",
			arrivalCity:       "Denpasar",
			departureEpoch:    1765799100,
			arrivalEpoch:      1765813800,
			durationMinutes:   185,
			durationFormatted: "3h 5m",
			stops:             1,
			totalPrice:        950000,
			seats:             41,
			aircraft:          "Airbus A320",
			amenities:         1,
		},
	}

	b, err := NewBatikAir()
	if err != nil {
		t.Fatalf("NewBatikAir() error = %v", err)
	}

	flights, err := b.Search(context.Background(), SearchRequest{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got, want := len(flights), len(tests); got != want {
		t.Fatalf("len(Search()) = %d, want %d", got, want)
	}

	for _, tt := range tests {
		t.Run(tt.flightNumber, func(t *testing.T) {
			flight := flights[tt.index]

			check := func(name string, got, want any) {
				t.Helper()

				if got != want {
					t.Errorf("%s = %v, want %v", name, got, want)
				}
			}

			check("Id", flight.Id, tt.id)
			check("Provider", flight.Provider, "Batik Air")
			check("Airline.Name", flight.Airline.Name, "Batik Air")
			check("Airline.Code", flight.Airline.Code, "ID")
			check("FlightNumber", flight.FlightNumber, tt.flightNumber)
			check("Departure.Airport", flight.Departure.Airport, "CGK")
			check("Departure.City", flight.Departure.City, tt.departureCity)
			check("Departure.Timestamp", flight.Departure.Timestamp, tt.departureEpoch)
			check("Arrival.Airport", flight.Arrival.Airport, "DPS")
			check("Arrival.City", flight.Arrival.City, tt.arrivalCity)
			check("Arrival.Timestamp", flight.Arrival.Timestamp, tt.arrivalEpoch)
			check("Duration.TotalMinutes", flight.Duration.TotalMinutes, tt.durationMinutes)
			check("Duration.Formatted", flight.Duration.Formatted, tt.durationFormatted)
			check("Stops", flight.Stops, tt.stops)
			check("Price.Amount", flight.Price.Amount, tt.totalPrice)
			check("Price.Currency", flight.Price.Currency, "IDR")
			check("AvailableSeats", flight.AvailableSeats, tt.seats)
			check("CabinClass", flight.CabinClass, "Y")
			check("len(Amenities)", len(flight.Amenities), tt.amenities)
			check("Baggage.CarryOn", flight.Baggage.CarryOn, "7kg cabin")
			check("Baggage.Checked", flight.Baggage.Checked, "20kg checked")

			if flight.Aircraft == nil {
				t.Error("Aircraft = nil, want the model")
			} else {
				check("Aircraft", *flight.Aircraft, tt.aircraft)
			}
		})
	}
}

func TestBatikAirNormalizeFallsBackOnUnmappableData(t *testing.T) {
	b, err := NewBatikAir()
	if err != nil {
		t.Fatalf("NewBatikAir() error = %v", err)
	}

	resp := &BatikResponse{
		Code:    200,
		Message: "OK",
		Results: []BatikFlight{
			{
				FlightNumber:      "ID9999",
				AirlineIATA:       "ID",
				Origin:            "XXX",
				Destination:       "YYY",
				DepartureDateTime: "not a datetime",
				ArrivalDateTime:   "15-12-2025 07:15",
				TravelTime:        "soon",
				Fare:              BatikFare{TotalPrice: 100, CurrencyCode: "IDR", Class: "Y"},
			},
		},
	}

	flights := b.Normalize(resp)

	if got, want := len(flights), 1; got != want {
		t.Fatalf("len(Normalize()) = %d, want %d", got, want)
	}

	flight := flights[0]

	if got, want := flight.Id, "ID9999_Batik Air"; got != want {
		t.Errorf("Id = %q, want %q", got, want)
	}
	if got, want := flight.Departure.City, "UNKNOWN"; got != want {
		t.Errorf("Departure.City = %q, want %q", got, want)
	}
	if got, want := flight.Arrival.City, "UNKNOWN"; got != want {
		t.Errorf("Arrival.City = %q, want %q", got, want)
	}
	if got, want := flight.Departure.Timestamp, int64(0); got != want {
		t.Errorf("Departure.Timestamp = %d, want %d", got, want)
	}
	if got, want := flight.Arrival.Timestamp, int64(0); got != want {
		t.Errorf("Arrival.Timestamp = %d, want %d", got, want)
	}
	if got, want := flight.Duration.TotalMinutes, 0; got != want {
		t.Errorf("Duration.TotalMinutes = %d, want %d", got, want)
	}
	if got, want := flight.Duration.Formatted, "0h 0m"; got != want {
		t.Errorf("Duration.Formatted = %q, want %q", got, want)
	}
	if flight.Aircraft == nil {
		t.Error("Aircraft = nil, want the empty model")
	} else if got, want := *flight.Aircraft, ""; got != want {
		t.Errorf("Aircraft = %q, want %q", got, want)
	}
	if got, want := len(flight.Amenities), 0; got != want {
		t.Errorf("len(Amenities) = %d, want %d", got, want)
	}
	if got, want := flight.Price.Amount, 100; got != want {
		t.Errorf("Price.Amount = %d, want %d", got, want)
	}
}

func TestBatikAirNormalizeWithoutResults(t *testing.T) {
	b, err := NewBatikAir()
	if err != nil {
		t.Fatalf("NewBatikAir() error = %v", err)
	}

	if got := len(b.Normalize(&BatikResponse{Code: 200, Message: "OK"})); got != 0 {
		t.Errorf("len(Normalize()) = %d, want 0", got)
	}
}

func TestBatikAirSearchHonoursContext(t *testing.T) {
	b, err := NewBatikAir()
	if err != nil {
		t.Fatalf("NewBatikAir() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	flights, err := b.Search(ctx, SearchRequest{})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Search() error = %v, want context.Canceled", err)
	}
	if len(flights) != 0 {
		t.Errorf("len(Search()) = %d, want 0", len(flights))
	}
}
