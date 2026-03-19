package market

import (
	"sort"

	"brandon-bot/internal/strategy"
)

// SortTicks sorts a slice of ticks by timestamp ascending, in place.
func SortTicks(ticks []strategy.Tick) {
	sort.Slice(ticks, func(i, j int) bool {
		return ticks[i].Timestamp.Before(ticks[j].Timestamp)
	})
}
