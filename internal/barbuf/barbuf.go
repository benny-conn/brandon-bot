// Package barbuf provides a per-symbol circular bar buffer for the engine.
package barbuf

import (
	"sync"

	"github.com/benny-conn/brandon-bot/strategy"
)

// Ensure DailyTracker implements strategy.DailyLevelProvider.
var _ strategy.DailyLevelProvider = (*DailyTracker)(nil)

const defaultCapacity = 500

// Buffer stores the last N bars per symbol in a ring buffer.
type Buffer struct {
	mu   sync.RWMutex
	data map[string]*ring
}

type ring struct {
	bars []strategy.Tick
	head int // next write index
	len  int // number of valid entries
	cap  int
}

// New creates a new bar buffer.
func New() *Buffer {
	return &Buffer{data: make(map[string]*ring)}
}

// Push adds a bar for a symbol.
func (b *Buffer) Push(tick strategy.Tick) {
	b.mu.Lock()
	defer b.mu.Unlock()

	r, ok := b.data[tick.Symbol]
	if !ok {
		r = &ring{bars: make([]strategy.Tick, defaultCapacity), cap: defaultCapacity}
		b.data[tick.Symbol] = r
	}
	r.bars[r.head] = tick
	r.head = (r.head + 1) % r.cap
	if r.len < r.cap {
		r.len++
	}
}

// Last returns the last N bars for a symbol, oldest first.
// Returns fewer than N if not enough bars have been received.
func (b *Buffer) Last(symbol string, n int) []strategy.Tick {
	b.mu.RLock()
	defer b.mu.RUnlock()

	r, ok := b.data[symbol]
	if !ok || r.len == 0 {
		return nil
	}
	if n > r.len {
		n = r.len
	}

	result := make([]strategy.Tick, n)
	start := (r.head - n + r.cap) % r.cap
	for i := 0; i < n; i++ {
		result[i] = r.bars[(start+i)%r.cap]
	}
	return result
}

// DailyLevels is an alias for strategy.DailyLevels.
type DailyLevels = strategy.DailyLevels

// DailyTracker tracks daily high/low/open/close per symbol.
type DailyTracker struct {
	mu   sync.RWMutex
	data map[string]*dailyState
}

type dailyState struct {
	date      string
	todayHigh float64
	todayLow  float64
	todayOpen float64
	prevHigh  float64
	prevLow   float64
	prevClose float64
	lastClose float64
}

// NewDailyTracker creates a new daily level tracker.
func NewDailyTracker() *DailyTracker {
	return &DailyTracker{data: make(map[string]*dailyState)}
}

// Update processes a new bar and updates daily levels.
// date should be a "YYYY-MM-DD" string in the market's timezone.
func (d *DailyTracker) Update(symbol, date string, open, high, low, close float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	s, ok := d.data[symbol]
	if !ok {
		s = &dailyState{}
		d.data[symbol] = s
	}

	if date != s.date {
		// New day — roll previous day's levels.
		if s.date != "" {
			s.prevHigh = s.todayHigh
			s.prevLow = s.todayLow
			s.prevClose = s.lastClose
		}
		s.date = date
		s.todayHigh = high
		s.todayLow = low
		s.todayOpen = open
	} else {
		if high > s.todayHigh {
			s.todayHigh = high
		}
		if low < s.todayLow {
			s.todayLow = low
		}
	}
	s.lastClose = close
}

// Levels returns the current daily levels for a symbol.
func (d *DailyTracker) Levels(symbol string) DailyLevels {
	d.mu.RLock()
	defer d.mu.RUnlock()

	s, ok := d.data[symbol]
	if !ok {
		return DailyLevels{}
	}
	return DailyLevels{
		PrevHigh:  s.prevHigh,
		PrevLow:   s.prevLow,
		PrevClose: s.prevClose,
		TodayHigh: s.todayHigh,
		TodayLow:  s.todayLow,
		TodayOpen: s.todayOpen,
	}
}
