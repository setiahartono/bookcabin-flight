package scoring

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"testing"

	"bookcabin-flight/internal/provider"
)

// newFlight builds the flight a test needs. The provider package keeps the
// nested types of FlightData unexported, so its JSON form is how another
// package gets a flight with a price, a travel time and stops.
func newFlight(t *testing.T, id string, amount, minutes, stops int) provider.FlightData {
	t.Helper()

	body := fmt.Sprintf(
		`{"id":%q,"price":{"amount":%d,"currency":"IDR"},"duration":{"total_minutes":%d},"stops":%d}`,
		id, amount, minutes, stops,
	)

	var flight provider.FlightData
	if err := json.Unmarshal([]byte(body), &flight); err != nil {
		t.Fatalf("decode flight %q: %v", id, err)
	}

	return flight
}

func almost(t *testing.T, name string, got, want float64) {
	t.Helper()

	if math.IsNaN(got) || math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func ids(ranked []provider.FlightData) []string {
	got := make([]string, 0, len(ranked))

	for _, flight := range ranked {
		got = append(got, flight.Id)
	}

	return got
}

func TestRankOrdersFlightsByBestValue(t *testing.T) {
	// The cheapest flight is direct, so it wins even though it is the slowest.
	cheap := newFlight(t, "QZ520_AirAsia", 650000, 130, 0)
	// Faster but 300000 more, second best value.
	mid := newFlight(t, "JT740_Lion Air", 950000, 105, 0)
	// The quickest, but the most expensive and with a stop.
	pricey := newFlight(t, "GA403_Garuda", 1200000, 100, 1)

	got := Rank([]provider.FlightData{mid, pricey, cheap}, SortByValue)

	want := []string{"QZ520_AirAsia", "JT740_Lion Air", "GA403_Garuda"}
	if !slices.Equal(ids(got), want) {
		t.Fatalf("Rank() ids = %v, want %v", ids(got), want)
	}

	tests := []struct {
		index       int
		value       float64
		price       float64
		convenience float64
	}{
		{index: 0, value: 0.958, price: 1, convenience: 0.861},
		{index: 1, value: 0.77, price: 0.684, convenience: 0.971},
		{index: 2, value: 0.619, price: 0.542, convenience: 0.8},
	}

	for _, tt := range tests {
		t.Run(got[tt.index].Id, func(t *testing.T) {
			almost(t, "Score.Value", got[tt.index].Score.Value, tt.value)
			almost(t, "Score.Price", got[tt.index].Score.Price, tt.price)
			almost(t, "Score.Convenience", got[tt.index].Score.Convenience, tt.convenience)
		})
	}
}

func TestRankSingleFlightIsTheBestValue(t *testing.T) {
	got := Rank([]provider.FlightData{newFlight(t, "QZ520_AirAsia", 650000, 100, 0)}, SortByValue)

	if len(got) != 1 {
		t.Fatalf("len(Rank()) = %d, want 1", len(got))
	}

	almost(t, "Score.Value", got[0].Score.Value, 1)
	almost(t, "Score.Price", got[0].Score.Price, 1)
	almost(t, "Score.Convenience", got[0].Score.Convenience, 1)
}

func TestRankGivesNoCreditForUnknownPriceAndTravelTime(t *testing.T) {
	known := newFlight(t, "QZ520_AirAsia", 650000, 100, 0)
	// What a provider normalizes to when it cannot map a fare or a travel time.
	unknown := newFlight(t, "ID9999_Batik Air", 0, 0, 0)

	got := Rank([]provider.FlightData{unknown, known}, SortByValue)

	want := []string{"QZ520_AirAsia", "ID9999_Batik Air"}
	if !slices.Equal(ids(got), want) {
		t.Fatalf("Rank() ids = %v, want %v", ids(got), want)
	}

	almost(t, "known Score.Value", got[0].Score.Value, 1)
	almost(t, "unknown Score.Price", got[1].Score.Price, 0)
	// The stops of that flight are known, so only the fare and the travel time
	// take the score down to the share of an itinerary without stops.
	almost(t, "unknown Score.Convenience", got[1].Score.Convenience, 0.4)
	almost(t, "unknown Score.Value", got[1].Score.Value, 0.12)
}

func TestRankPrefersTheFewerStops(t *testing.T) {
	direct := newFlight(t, "GA403_Garuda", 1000000, 100, 0)
	oneStop := newFlight(t, "ID7042_Batik Air", 1000000, 100, 1)
	twoStops := newFlight(t, "QG900_Citilink", 1000000, 100, 2)

	got := Rank([]provider.FlightData{twoStops, oneStop, direct}, SortByValue)

	want := []string{"GA403_Garuda", "ID7042_Batik Air", "QG900_Citilink"}
	if !slices.Equal(ids(got), want) {
		t.Fatalf("Rank() ids = %v, want %v", ids(got), want)
	}

	tests := []struct {
		index       int
		value       float64
		convenience float64
	}{
		{index: 0, value: 1, convenience: 1},
		{index: 1, value: 0.94, convenience: 0.8},
		{index: 2, value: 0.92, convenience: 0.733},
	}

	for _, tt := range tests {
		t.Run(got[tt.index].Id, func(t *testing.T) {
			almost(t, "Score.Value", got[tt.index].Score.Value, tt.value)
			almost(t, "Score.Convenience", got[tt.index].Score.Convenience, tt.convenience)
			almost(t, "Score.Price", got[tt.index].Score.Price, 1)
		})
	}
}

func TestRankKeepsTheOrderOfFlightsThatScoreTheSame(t *testing.T) {
	first := newFlight(t, "QZ520_AirAsia", 800000, 120, 0)
	second := newFlight(t, "JT740_Lion Air", 800000, 120, 0)
	worst := newFlight(t, "GA403_Garuda", 1600000, 240, 1)

	got := Rank([]provider.FlightData{first, second, worst}, SortByValue)

	want := []string{"QZ520_AirAsia", "JT740_Lion Air", "GA403_Garuda"}
	if !slices.Equal(ids(got), want) {
		t.Fatalf("Rank() ids = %v, want %v", ids(got), want)
	}

	almost(t, "first Score.Value", got[0].Score.Value, 1)
	almost(t, "second Score.Value", got[1].Score.Value, 1)
	almost(t, "worst Score.Value", got[2].Score.Value, 0.5)
	almost(t, "worst Score.Price", got[2].Score.Price, 0.5)
	almost(t, "worst Score.Convenience", got[2].Score.Convenience, 0.5)
}

func TestRankWithoutFlights(t *testing.T) {
	got := Rank(nil, SortByValue)

	if got == nil {
		t.Fatal("Rank(nil) = nil, want an empty slice so a response marshals to []")
	}
	if len(got) != 0 {
		t.Errorf("len(Rank(nil)) = %d, want 0", len(got))
	}
}

func TestRankScoresStayBetweenZeroAndOne(t *testing.T) {
	got := Rank([]provider.FlightData{
		newFlight(t, "QZ520_AirAsia", 650000, 130, 0),
		newFlight(t, "ID6514_Batik Air", 1100000, 185, 1),
		newFlight(t, "GA403_Garuda", 1200000, 100, 3),
	}, SortByValue)

	if len(got) != 3 {
		t.Fatalf("len(Rank()) = %d, want 3", len(got))
	}

	for _, flight := range got {
		for name, value := range map[string]float64{
			"Score.Value":       flight.Score.Value,
			"Score.Price":       flight.Score.Price,
			"Score.Convenience": flight.Score.Convenience,
		} {
			if value < 0 || value > 1 {
				t.Errorf("%s %s = %v, want between 0 and 1", flight.Id, name, value)
			}
		}
	}
}

func TestRankPutsTheScoreOnTheFlight(t *testing.T) {
	got := Rank([]provider.FlightData{newFlight(t, "QZ520_AirAsia", 650000, 100, 0)}, SortByValue)

	body, err := json.Marshal(got[0])
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded struct {
		Id    string         `json:"id"`
		Score provider.Score `json:"score"`
	}

	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", body, err)
	}

	if decoded.Id != "QZ520_AirAsia" {
		t.Errorf("id = %q, want the flight fields beside the score", decoded.Id)
	}
	almost(t, "Score.Value", decoded.Score.Value, 1)
}

// TestRankOrdersByTheKeyItWasGiven uses a subset whose cheapest fare, best value
// and most convenient flight are three different flights, so every key puts
// another flight first.
func TestRankOrdersByTheKeyItWasGiven(t *testing.T) {
	// The cheapest, but the slowest and with two stops.
	cheapest := newFlight(t, "QZ520_AirAsia", 650000, 200, 2)
	// The best value: nearly as cheap as the cheapest, direct and the quickest.
	best := newFlight(t, "JT740_Lion Air", 700000, 100, 0)
	// Direct and quick as well, but the most expensive.
	pricey := newFlight(t, "GA403_Garuda", 900000, 110, 0)

	flights := []provider.FlightData{pricey, cheapest, best}

	tests := []struct {
		name string
		key  SortKey
		want []string
	}{
		{name: "best value first", key: SortByValue, want: []string{"JT740_Lion Air", "QZ520_AirAsia", "GA403_Garuda"}},
		{name: "cheapest fare first", key: SortByPrice, want: []string{"QZ520_AirAsia", "JT740_Lion Air", "GA403_Garuda"}},
		{name: "most convenient first", key: SortByConvenience, want: []string{"JT740_Lion Air", "GA403_Garuda", "QZ520_AirAsia"}},
	}

	scores := make(map[string]provider.Score)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Rank(flights, tt.key)

			if !slices.Equal(ids(got), tt.want) {
				t.Fatalf("Rank() ids = %v, want %v", ids(got), tt.want)
			}

			// The key orders the result, it does not change how a flight scores.
			for _, flight := range got {
				known, seen := scores[flight.Id]
				if !seen {
					scores[flight.Id] = flight.Score

					continue
				}

				if flight.Score != known {
					t.Errorf("%s Score = %+v, want %+v", flight.Id, flight.Score, known)
				}
			}
		})
	}
}

func TestRankKeepsTheOrderOfFlightsThatShareTheKey(t *testing.T) {
	first := newFlight(t, "QZ520_AirAsia", 800000, 120, 0)
	second := newFlight(t, "JT740_Lion Air", 800000, 120, 0)

	got := Rank([]provider.FlightData{first, second}, SortByPrice)

	want := []string{"QZ520_AirAsia", "JT740_Lion Air"}
	if !slices.Equal(ids(got), want) {
		t.Errorf("Rank() ids = %v, want %v", ids(got), want)
	}
}

func TestParseSortKey(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  SortKey
		ok    bool
	}{
		{name: "left out", value: "", want: SortByValue, ok: true},
		{name: "the default", value: "value", want: SortByValue, ok: true},
		{name: "price", value: "price", want: SortByPrice, ok: true},
		{name: "convenience", value: "convenience", want: SortByConvenience, ok: true},
		{name: "written in another case", value: "PRICE", want: SortByPrice, ok: true},
		{name: "with spaces around it", value: " convenience ", want: SortByConvenience, ok: true},
		{name: "an order no flight scores", value: "cheapest", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseSortKey(tt.value)

			if got != tt.want || ok != tt.ok {
				t.Errorf("ParseSortKey(%q) = %q, %v, want %q, %v", tt.value, got, ok, tt.want, tt.ok)
			}
		})
	}
}
