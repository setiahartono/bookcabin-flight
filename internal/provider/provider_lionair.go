package provider

import (
	"context"
	"fmt"
	"time"

	"bookcabin-flight/internal/utils"
)

const lionAirFixture = "lion_air_search_response.json"

const (
	lionAirMinDelay = 100 * time.Millisecond
	lionAirMaxDelay = 200 * time.Millisecond
)

type LionResponse struct {
	Success bool     `json:"success"`
	Data    LionData `json:"data"`
}

type LionData struct {
	AvailableFlights []LionFlight `json:"available_flights"`
}

type LionFlight struct {
	ID         string        `json:"id"`
	Carrier    LionCarrier   `json:"carrier"`
	Route      LionRoute     `json:"route"`
	Schedule   LionSchedule  `json:"schedule"`
	FlightTime int           `json:"flight_time"`
	IsDirect   bool          `json:"is_direct"`
	StopCount  int           `json:"stop_count"`
	Layovers   []LionLayover `json:"layovers"`
	Pricing    LionPricing   `json:"pricing"`
	SeatsLeft  int           `json:"seats_left"`
	PlaneType  string        `json:"plane_type"`
	Services   LionServices  `json:"services"`
}

type LionCarrier struct {
	Name string `json:"name"`
	IATA string `json:"iata"`
}

type LionRoute struct {
	From LionEndpoint `json:"from"`
	To   LionEndpoint `json:"to"`
}

type LionEndpoint struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

type LionSchedule struct {
	Departure         string `json:"departure"`
	DepartureTimezone string `json:"departure_timezone"`
	Arrival           string `json:"arrival"`
	ArrivalTimezone   string `json:"arrival_timezone"`
}

type LionLayover struct {
	Airport         string `json:"airport"`
	DurationMinutes int    `json:"duration_minutes"`
}

type LionPricing struct {
	Total    int    `json:"total"`
	Currency string `json:"currency"`
	FareType string `json:"fare_type"`
}

type LionServices struct {
	WifiAvailable    bool                 `json:"wifi_available"`
	MealsIncluded    bool                 `json:"meals_included"`
	BaggageAllowance LionBaggageAllowance `json:"baggage_allowance"`
}

type LionBaggageAllowance struct {
	Cabin string `json:"cabin"`
	Hold  string `json:"hold"`
}

type LionAir struct {
	providerService
}

func NewLionAir() (*LionAir, error) {
	svc, err := newProviderService("Lion Air", lionAirFixture)
	if err != nil {
		return nil, err
	}

	return &LionAir{providerService: *svc}, nil
}

func (l *LionAir) Normalize(resp *LionResponse) []FlightData {
	flights := make([]FlightData, 0, len(resp.Data.AvailableFlights))

	for _, f := range resp.Data.AvailableFlights {
		departureCity, ok := utils.AirportCity(f.Route.From.Code)
		if ok != true {
			departureCity = "UNKNOWN"
		}
		arrivalCity, ok := utils.AirportCity(f.Route.To.Code)
		if ok != true {
			arrivalCity = "UNKNOWN"
		}

		flights = append(flights, FlightData{
			Id:       fmt.Sprintf("%s_%s", f.ID, l.name),
			Provider: l.name,
			Airline: airline{
				Name: f.Carrier.Name,
				Code: f.Carrier.IATA,
			},
			FlightNumber: f.ID,
			Departure: schedule{
				Airport:   f.Route.From.Code,
				City:      departureCity,
				Datetime:  f.Schedule.Departure,
				Timestamp: lionTimestamp(f.Schedule.Departure, f.Schedule.DepartureTimezone),
			},
			Arrival: schedule{
				Airport:   f.Route.To.Code,
				City:      arrivalCity,
				Datetime:  f.Schedule.Arrival,
				Timestamp: lionTimestamp(f.Schedule.Arrival, f.Schedule.ArrivalTimezone),
			},
			Duration: duration{
				TotalMinutes: f.FlightTime,
				Formatted:    utils.FormatDuration(f.FlightTime),
			},
			Stops: lionStops(f),
			Price: price{
				Amount:   f.Pricing.Total,
				Currency: f.Pricing.Currency,
			},
			AvailableSeats: f.SeatsLeft,
			CabinClass:     f.Pricing.FareType,
			Aircraft:       &f.PlaneType,
			Amenities:      lionAmenities(f.Services),
			Baggage:        lionBaggage(f.Services.BaggageAllowance),
		})
	}

	return flights
}

func lionStops(f LionFlight) int {
	if f.StopCount > 0 {
		return f.StopCount
	}

	return len(f.Layovers)
}

func lionAmenities(services LionServices) []string {
	amenities := make([]string, 0, 2)

	if services.WifiAvailable {
		amenities = append(amenities, "WiFi")
	}

	if services.MealsIncluded {
		amenities = append(amenities, "Meals")
	}

	return amenities
}

func lionBaggage(allowance LionBaggageAllowance) baggage {
	return utils.AcquireBaggageInfo(fmt.Sprintf("%s, %s", allowance.Cabin, allowance.Hold))
}

func lionTimestamp(datetime, timezone string) int64 {
	if location, err := time.LoadLocation(timezone); err == nil {
		if parsed, err := time.ParseInLocation("2006-01-02T15:04:05", datetime, location); err == nil {
			return parsed.Unix()
		}
	}

	if timestamp, err := utils.DatetimeToTimestamp(datetime); err == nil {
		return timestamp
	}

	return 0
}

func (l *LionAir) Search(ctx context.Context, _ SearchRequest) ([]FlightData, error) {
	if err := l.wait(ctx, lionAirMinDelay, lionAirMaxDelay); err != nil {
		return nil, err
	}

	var resp LionResponse
	if err := l.decode(&resp); err != nil {
		return nil, err
	}

	flightData := l.Normalize(&resp)

	return flightData, nil
}
