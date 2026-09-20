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
