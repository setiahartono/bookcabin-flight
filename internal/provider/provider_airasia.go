package provider

import (
	"bookcabin-flight/internal/utils"
	"context"
	"fmt"
	"math"
	"time"
)

const providerName = "AirAsia"
const flightPrefix = "QZ"
const airAsiaFixture = "airasia_search_response.json"

const (
	airAsiaMinDelay    = 50 * time.Millisecond
	airAsiaMaxDelay    = 150 * time.Millisecond
	airAsiaSuccessRate = 0.9
)

type AirAsiaResponse struct {
	Status  string          `json:"status"`
	Flights []AirAsiaFlight `json:"flights"`
}

type AirAsiaFlight struct {
	FlightCode    string        `json:"flight_code"`
	Airline       string        `json:"airline"`
	FromAirport   string        `json:"from_airport"`
	ToAirport     string        `json:"to_airport"`
	DepartTime    string        `json:"depart_time"`
	ArriveTime    string        `json:"arrive_time"`
	DurationHours float64       `json:"duration_hours"`
	DirectFlight  bool          `json:"direct_flight"`
	Stops         []AirAsiaStop `json:"stops"`
	PriceIDR      int           `json:"price_idr"`
	Seats         int           `json:"seats"`
	CabinClass    string        `json:"cabin_class"`
	BaggageNote   string        `json:"baggage_note"`
}

type AirAsiaStop struct {
	Airport         string `json:"airport"`
	WaitTimeMinutes int    `json:"wait_time_minutes"`
}

type AirAsia struct {
	providerService
}

func NewAirAsia() (*AirAsia, error) {
	svc, err := newProviderService(providerName, airAsiaFixture)
	if err != nil {
		return nil, err
	}

	return &AirAsia{providerService: *svc}, nil
}

func (a *AirAsia) Normalize(resp *AirAsiaResponse) []FlightData {
	flights := make([]FlightData, 0, len(resp.Flights))

	for _, f := range resp.Flights {
		departureCity, ok := utils.AirportCity(f.FromAirport)
		if ok != true {
			departureCity = "UNKNOWN"
		}
		arrivalCity, ok := utils.AirportCity(f.ToAirport)
		if ok != true {
			arrivalCity = "UNKNOWN"
		}
		departureTimestamp, err := utils.DatetimeToTimestamp(f.DepartTime)
		if err != nil {
			departureTimestamp = 0
		}
		arrivalTimestamp, err := utils.DatetimeToTimestamp(f.ArriveTime)
		if err != nil {
			arrivalTimestamp = 0
		}
		totalDurationInMinutes := int(math.Round(f.DurationHours * 60))

		flights = append(flights, FlightData{
			Id:       fmt.Sprintf("%s_%s", f.FlightCode, providerName),
			Provider: providerName,
			Airline: airline{
				Name: providerName,
				Code: flightPrefix,
			},
			FlightNumber: f.FlightCode,
			Departure: schedule{
				Airport:   f.FromAirport,
				City:      departureCity,
				Datetime:  f.DepartTime,
				Timestamp: departureTimestamp,
			},
			Arrival: schedule{
				Airport:   f.ToAirport,
				City:      arrivalCity,
				Datetime:  f.ArriveTime,
				Timestamp: arrivalTimestamp,
			},
			Duration: duration{
				TotalMinutes: totalDurationInMinutes,
				Formatted:    utils.FormatDuration(totalDurationInMinutes),
			},
			Stops: len(f.Stops),
			Price: price{
				Amount:   f.PriceIDR,
				Currency: "IDR", // hardcoded, no currency returned from the API
			},
			AvailableSeats: f.Seats,
			CabinClass:     f.CabinClass,
			Aircraft:       nil,
			Amenities:      []string{},
			Baggage:        utils.AcquireBaggageInfo(f.BaggageNote),
		})
	}
	return flights
}

func (a *AirAsia) Search(ctx context.Context, req SearchRequest) ([]FlightData, bool, error) {
	return a.search(req, func() ([]FlightData, error) {
		if err := a.wait(ctx, airAsiaMinDelay, airAsiaMaxDelay); err != nil {
			return nil, err
		}

		if err := a.maybeFail(airAsiaSuccessRate); err != nil {
			return nil, err
		}

		var resp AirAsiaResponse
		if err := a.decode(&resp); err != nil {
			return nil, err
		}

		return a.Normalize(&resp), nil
	})
}
