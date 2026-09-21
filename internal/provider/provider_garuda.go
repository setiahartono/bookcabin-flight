package provider

import (
	"context"
	"fmt"
	"time"

	"bookcabin-flight/internal/utils"
)

const garudaFixture = "garuda_indonesia_search_response.json"

const (
	garudaMinDelay = 50 * time.Millisecond
	garudaMaxDelay = 100 * time.Millisecond
)

type GarudaResponse struct {
	Status  string         `json:"status"`
	Flights []GarudaFlight `json:"flights"`
}

type GarudaFlight struct {
	FlightID        string          `json:"flight_id"`
	Airline         string          `json:"airline"`
	AirlineCode     string          `json:"airline_code"`
	Departure       GarudaEndpoint  `json:"departure"`
	Arrival         GarudaEndpoint  `json:"arrival"`
	DurationMinutes int             `json:"duration_minutes"`
	Stops           int             `json:"stops"`
	Aircraft        string          `json:"aircraft"`
	Price           GarudaPrice     `json:"price"`
	Segments        []GarudaSegment `json:"segments"`
	AvailableSeats  int             `json:"available_seats"`
	FareClass       string          `json:"fare_class"`
	Baggage         GarudaBaggage   `json:"baggage"`
	Amenities       []string        `json:"amenities"`
}

type GarudaEndpoint struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type GarudaSegment struct {
	FlightNumber    string                `json:"flight_number"`
	Departure       GarudaSegmentEndpoint `json:"departure"`
	Arrival         GarudaSegmentEndpoint `json:"arrival"`
	DurationMinutes int                   `json:"duration_minutes"`
	LayoverMinutes  int                   `json:"layover_minutes"`
}

type GarudaSegmentEndpoint struct {
	Airport string `json:"airport"`
	Time    string `json:"time"`
}

type GarudaPrice struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type GarudaBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}

type Garuda struct {
	providerService
}

func NewGaruda() (*Garuda, error) {
	svc, err := newProviderService("Garuda Indonesia", garudaFixture)
	if err != nil {
		return nil, err
	}

	return &Garuda{providerService: *svc}, nil
}

// Normalize maps a Garuda Indonesia response onto the flight shape every
// provider shares.
func (g *Garuda) Normalize(resp *GarudaResponse) []FlightData {
	flights := make([]FlightData, 0, len(resp.Flights))

	for _, f := range resp.Flights {
		departureCity, ok := utils.AirportCity(f.Departure.Airport)
		if ok != true {
			departureCity = "UNKNOWN"
		}
		arrivalCity, ok := utils.AirportCity(f.Arrival.Airport)
		if ok != true {
			arrivalCity = "UNKNOWN"
		}
		departureTimestamp, err := utils.DatetimeToTimestamp(f.Departure.Time)
		if err != nil {
			departureTimestamp = 0
		}
		arrivalTimestamp, err := utils.DatetimeToTimestamp(f.Arrival.Time)
		if err != nil {
			arrivalTimestamp = 0
		}

		flights = append(flights, FlightData{
			Id:       fmt.Sprintf("%s_%s", f.FlightID, g.name),
			Provider: g.name,
			Airline: airline{
				Name: f.Airline,
				Code: f.AirlineCode,
			},
			FlightNumber: f.FlightID,
			Departure: schedule{
				Airport:   f.Departure.Airport,
				City:      departureCity,
				Datetime:  f.Departure.Time,
				Timestamp: departureTimestamp,
			},
			Arrival: schedule{
				Airport:   f.Arrival.Airport,
				City:      arrivalCity,
				Datetime:  f.Arrival.Time,
				Timestamp: arrivalTimestamp,
			},
			Duration: duration{
				TotalMinutes: f.DurationMinutes,
				Formatted:    utils.FormatDuration(f.DurationMinutes),
			},
			Stops: f.Stops,
			Price: price{
				Amount:   f.Price.Amount,
				Currency: f.Price.Currency,
			},
			AvailableSeats: f.AvailableSeats,
			CabinClass:     f.FareClass,
			Aircraft:       &f.Aircraft,
			Amenities:      f.Amenities,
			Baggage:        garudaBaggage(f.Baggage),
		})
	}

	return flights
}

// garudaBaggage renders the piece counts Garuda sends as the baggage text the
// shared flight shape carries.
func garudaBaggage(allowance GarudaBaggage) baggage {
	return baggage{
		CarryOn: pieceCount(allowance.CarryOn),
		Checked: pieceCount(allowance.Checked),
	}
}

// pieceCount names a baggage allowance counted in pieces.
func pieceCount(pieces int) string {
	if pieces <= 0 {
		return ""
	}

	if pieces == 1 {
		return "1 piece"
	}

	return fmt.Sprintf("%d pieces", pieces)
}

func (g *Garuda) Search(ctx context.Context, req SearchRequest) ([]FlightData, bool, error) {
	return g.search(req, func() ([]FlightData, error) {
		if err := g.wait(ctx, garudaMinDelay, garudaMaxDelay); err != nil {
			return nil, err
		}

		var resp GarudaResponse
		if err := g.decode(&resp); err != nil {
			return nil, err
		}

		return g.Normalize(&resp), nil
	})
}
