package engine

import (
	"fmt"
	"sort"
	"time"

	"github.com/benny-conn/runbook/strategy"
)

// ParseTimeframe converts a canonical timeframe string to a time.Duration.
func ParseTimeframe(tf string) (time.Duration, error) {
	switch tf {
	case "1s":
		return time.Second, nil
	case "15s":
		return 15 * time.Second, nil
	case "30s":
		return 30 * time.Second, nil
	case "1m":
		return time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "30m":
		return 30 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported timeframe %q", tf)
	}
}

// SortTimeframes sorts timeframe strings by duration ascending (finest first)
// and returns the sorted list. Returns an error if any timeframe is invalid.
func SortTimeframes(tfs []string) ([]string, error) {
	type tfDur struct {
		tf  string
		dur time.Duration
	}
	items := make([]tfDur, len(tfs))
	for i, tf := range tfs {
		dur, err := ParseTimeframe(tf)
		if err != nil {
			return nil, err
		}
		items[i] = tfDur{tf, dur}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].dur < items[j].dur
	})
	sorted := make([]string, len(items))
	for i, item := range items {
		sorted[i] = item.tf
	}
	return sorted, nil
}

// barAccumulator tracks in-progress OHLCV state for one symbol in one timeframe.
type barAccumulator struct {
	windowStart time.Time
	open        float64
	high        float64
	low         float64
	close       float64
	volume      int64
	hasData     bool
}

func (a *barAccumulator) update(tick strategy.Tick) {
	if !a.hasData {
		a.open = tick.Open
		a.high = tick.High
		a.low = tick.Low
		a.hasData = true
	}
	if tick.High > a.high {
		a.high = tick.High
	}
	if tick.Low < a.low {
		a.low = tick.Low
	}
	a.close = tick.Close
	a.volume += tick.Volume
}

func (a *barAccumulator) flush(symbol string) strategy.Tick {
	return strategy.Tick{
		Symbol:    symbol,
		Timestamp: a.windowStart,
		Open:      a.open,
		High:      a.high,
		Low:       a.low,
		Close:     a.close,
		Volume:    a.volume,
	}
}

func (a *barAccumulator) reset(windowStart time.Time) {
	*a = barAccumulator{windowStart: windowStart}
}

// BarAggregator accumulates base-timeframe ticks into a higher-timeframe bar.
// It maintains per-symbol state and emits a completed bar via the handler
// whenever a new time window begins (meaning the previous window is complete).
type BarAggregator struct {
	timeframe    string
	duration     time.Duration
	accumulators map[string]*barAccumulator
	handler      func(timeframe string, tick strategy.Tick)
}

// NewBarAggregator creates an aggregator for a single target timeframe.
// The handler is called synchronously each time a completed bar is ready.
func NewBarAggregator(timeframe string, dur time.Duration, handler func(string, strategy.Tick)) *BarAggregator {
	return &BarAggregator{
		timeframe:    timeframe,
		duration:     dur,
		accumulators: make(map[string]*barAccumulator),
		handler:      handler,
	}
}

// Update feeds a base-timeframe tick into the aggregator. If the tick crosses
// into a new time window, the previously accumulated bar is emitted via handler
// before the new window starts accumulating.
func (ba *BarAggregator) Update(tick strategy.Tick) {
	windowStart := tick.Timestamp.Truncate(ba.duration)

	acc, ok := ba.accumulators[tick.Symbol]
	if !ok {
		acc = &barAccumulator{}
		acc.reset(windowStart)
		ba.accumulators[tick.Symbol] = acc
	}

	if windowStart != acc.windowStart && acc.hasData {
		ba.handler(ba.timeframe, acc.flush(tick.Symbol))
		acc.reset(windowStart)
	}

	acc.update(tick)
}

// Flush emits any in-progress bars (e.g. at shutdown).
func (ba *BarAggregator) Flush() {
	for symbol, acc := range ba.accumulators {
		if acc.hasData {
			ba.handler(ba.timeframe, acc.flush(symbol))
		}
	}
}
