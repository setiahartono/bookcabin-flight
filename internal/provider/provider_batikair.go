package provider

import (
	"context"
	"fmt"
	"time"

	"bookcabin-flight/internal/utils"
)

const batikAirFixture = "batik_air_search_response.json"

const (
	batikAirMinDelay = 200 * time.Millisecond
	batikAirMaxDelay = 400 * time.Millisecond
)

type BatikResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Results []BatikFlight `json:"results"`
}

type BatikFlight struct {
	FlightNumber      string            `json:"flightNumber"`
	AirlineName       string            `json:"airlineName"`
	AirlineIATA       string            `json:"airlineIATA"`
	Origin            string            `json:"origin"`
	Destination       string            `json:"destination"`
	DepartureDateTime string            `json:"departureDateTime"`
	ArrivalDateTime   string            `json:"arrivalDateTime"`
	TravelTime        string            `json:"travelTime"`
	NumberOfStops     int               `json:"numberOfStops"`
	Connections       []BatikConnection `json:"connections"`
	Fare              BatikFare         `json:"fare"`
	SeatsAvailable    int               `json:"seatsAvailable"`
	AircraftModel     string            `json:"aircraftModel"`
	BaggageInfo       string            `json:"baggageInfo"`
	OnboardServices   []string          `json:"onboardServices"`
}

type BatikConnection struct {
	StopAirport  string `json:"stopAirport"`
	StopDuration string `json:"stopDuration"`
}

type BatikFare struct {
	BasePrice    int    `json:"basePrice"`
	Taxes        int    `json:"taxes"`
	TotalPrice   int    `json:"totalPrice"`
	CurrencyCode string `json:"currencyCode"`
	Class        string `json:"class"`
}

type BatikAir struct {
	providerService
}

func NewBatikAir() (*BatikAir, error) {
	svc, err := newProviderService("Batik Air", batikAirFixture)
	if err != nil {
		return nil, err
	}

	return &BatikAir{providerService: *svc}, nil
}

func (b *BatikAir) Normalize(resp *BatikResponse) []FlightData {
	flights := make([]FlightData, 0, len(resp.Results))

	for _, f := range resp.Results {
		departureCity, ok := utils.AirportCity(f.Origin)
		if ok != true {
			departureCity = "UNKNOWN"
		}
		arrivalCity, ok := utils.AirportCity(f.Destination)
		if ok != true {
			arrivalCity = "UNKNOWN"
		}
		departureTimestamp, err := utils.DatetimeToTimestamp(f.DepartureDateTime)
		if err != nil {
			departureTimestamp = 0
		}
		arrivalTimestamp, err := utils.DatetimeToTimestamp(f.ArrivalDateTime)
		if err != nil {
			arrivalTimestamp = 0
		}
		totalDurationInMinutes := travelTimeInMinutes(f.TravelTime)

		flights = append(flights, FlightData{
			Id:       fmt.Sprintf("%s_%s", f.FlightNumber, b.name),
			Provider: b.name,
			Airline: airline{
				Name: b.name,
				Code: f.AirlineIATA,
			},
			FlightNumber: f.FlightNumber,
			Departure: schedule{
				Airport:   f.Origin,
				City:      departureCity,
				Datetime:  f.DepartureDateTime,
				Timestamp: departureTimestamp,
			},
			Arrival: schedule{
				Airport:   f.Destination,
				City:      arrivalCity,
				Datetime:  f.ArrivalDateTime,
				Timestamp: arrivalTimestamp,
			},
			Duration: duration{
				TotalMinutes: totalDurationInMinutes,
				Formatted:    utils.FormatDuration(totalDurationInMinutes),
			},
			Stops: f.NumberOfStops,
			Price: price{
				Amount:   f.Fare.TotalPrice,
				Currency: f.Fare.CurrencyCode,
			},
			AvailableSeats: f.SeatsAvailable,
			CabinClass:     f.Fare.Class,
			Aircraft:       &f.AircraftModel,
			Amenities:      f.OnboardServices,
			Baggage:        utils.AcquireBaggageInfo(f.BaggageInfo),
		})
	}

	return flights
}

// travelTimeInMinutes reads a travel time such as "1h 45m".
func travelTimeInMinutes(travelTime string) int {
	var hours, minutes int

	if _, err := fmt.Sscanf(travelTime, "%dh %dm", &hours, &minutes); err != nil {
		return 0
	}

	return hours*60 + minutes
}

func (b *BatikAir) Search(ctx context.Context, req SearchRequest) ([]FlightData, bool, error) {
	return b.search(ctx, req, func() ([]FlightData, error) {
		if err := b.wait(ctx, batikAirMinDelay, batikAirMaxDelay); err != nil {
			return nil, err
		}

		var resp BatikResponse
		if err := b.decode(&resp); err != nil {
			return nil, err
		}

		return b.Normalize(&resp), nil
	})
}
