// Package scoring rates the flights of an already filtered search result by
// best value: how much of the cheapest fare is paid for how convenient the
// itinerary is.
package scoring

import (
	"math"
	"sort"
	"strings"

	"bookcabin-flight/internal/provider"
)

// Weights of the best value score. The fare dominates, convenience decides
// between the flights that cost about the same.
// PriceWeight+ConvenienceWeight and DurationWeight+StopsWeight each add up to 1.
const (
	PriceWeight       = 0.7
	ConvenienceWeight = 0.3

	DurationWeight = 0.6
	StopsWeight    = 0.4
)

// SortKey names the score component a search result is ordered by. Every flight
// is scored on all three components whatever the key, so a response always
// reports them and can be reordered by a client itself.
type SortKey string

const (
	// SortByValue lists the best value first, which is the order of a search that
	// asked for none.
	SortByValue SortKey = "value"
	// SortByPrice lists the cheapest fare first.
	SortByPrice SortKey = "price"
	// SortByConvenience lists the quickest itinerary with the fewest stops first.
	SortByConvenience SortKey = "convenience"
)

// ParseSortKey reads the sortBy a search asked for, written in any case and with
// spaces around it. A missing key is the default, and a key that names no score
// component is reported, so the search can be refused instead of quietly ordered
// the way it was not asked for.
func ParseSortKey(value string) (SortKey, bool) {
	switch SortKey(strings.ToLower(strings.TrimSpace(value))) {
	case "":
		return SortByValue, true
	case SortByValue:
		return SortByValue, true
	case SortByPrice:
		return SortByPrice, true
	case SortByConvenience:
		return SortByConvenience, true
	default:
		return "", false
	}
}

// Rank scores the flights of the subset and returns them ordered by the score
// component the key names, best value first by default: the first flight of a
// search result is the one to recommend.
//
// Every flight is compared to the best the subset offers: the cheapest fare, and
// the quickest itinerary with the fewest stops. The cheapest flight scores 1 on
// price and a flight costing twice as much scores 0.5; travel time is scored the
// same way, while every stop takes a share off the stops component. A price or a
// travel time a provider could not normalize (zero) earns no credit for that
// component, while the stops of the itinerary always count.
// Flights that score the same keep the order they were given in.
func Rank(flights []provider.FlightData, by SortKey) []provider.FlightData {
	cheapest, quickest := reference(flights)

	ranked := make([]provider.FlightData, 0, len(flights))
	for _, flight := range flights {
		flight.Score = score(flight, cheapest, quickest)

		ranked = append(ranked, flight)
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return component(ranked[i], by) > component(ranked[j], by)
	})

	return ranked
}

// component is the score the flights of a result are ordered by.
func component(flight provider.FlightData, by SortKey) float64 {
	switch by {
	case SortByPrice:
		return flight.Score.Price
	case SortByConvenience:
		return flight.Score.Convenience
	default:
		return flight.Score.Value
	}
}

// reference returns the cheapest fare and the shortest travel time of the
// subset, ignoring the flights whose price or travel time is unknown.
func reference(flights []provider.FlightData) (cheapest, quickest int) {
	for _, flight := range flights {
		cheapest = smallestPositive(cheapest, flight.Price.Amount)
		quickest = smallestPositive(quickest, flight.Duration.TotalMinutes)
	}

	return cheapest, quickest
}

// smallestPositive returns the smaller of the two values, where zero means
// unknown and never wins over a known value.
func smallestPositive(best, value int) int {
	if value <= 0 || (best > 0 && best <= value) {
		return best
	}

	return value
}

func score(flight provider.FlightData, cheapest, quickest int) provider.Score {
	price := ratio(flight.Price.Amount, cheapest)
	convenience := round(DurationWeight*ratio(flight.Duration.TotalMinutes, quickest) + StopsWeight*stops(flight.Stops))

	return provider.Score{
		Value:       round(PriceWeight*price + ConvenienceWeight*convenience),
		Price:       price,
		Convenience: convenience,
	}
}

// ratio compares a value to the best of the subset: the best scores 1 and twice
// the best scores 0.5. An unknown value scores 0.
func ratio(value, best int) float64 {
	if value <= 0 || best <= 0 {
		return 0
	}

	return round(float64(best) / float64(value))
}

// stops takes the component down for every stop on the itinerary.
func stops(count int) float64 {
	if count < 0 {
		count = 0
	}

	return 1 / float64(1+count)
}

// round keeps the scores at three decimals so they stay readable in a response.
func round(value float64) float64 {
	return math.Round(value*1000) / 1000
}
