package provider

import "bookcabin-flight/internal/utils"

type FlightData struct {
	Id             string   `json:"id"`
	Provider       string   `json:"provider"`
	Airline        airline  `json:"airline"`
	FlightNumber   string   `json:"flight_number"`
	Departure      schedule `json:"departure"`
	Arrival        schedule `json:"arrival"`
	Duration       duration `json:"duration"`
	Stops          int      `json:"stops"`
	Price          price    `json:"price"`
	AvailableSeats int      `json:"available_seats"`
	CabinClass     string   `json:"cabin_class"`
	Aircraft       *string  `json:"aircraft"`
	Amenities      []string `json:"amenities"`
	Baggage        baggage  `json:"baggage"`
	Score          Score    `json:"score"`
}

// Score is the best value of a flight inside the subset it was scored in,
// between 0 (worst) and 1 (the best value that subset offers).
// Value is the weighted sum of the Price and Convenience components, both
// rounded to three decimals, so Value can be recomputed from them.
type Score struct {
	Value       float64 `json:"value"`
	Price       float64 `json:"price"`
	Convenience float64 `json:"convenience"`
}

type airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type schedule struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

type duration struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type price struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type baggage = utils.Baggage
