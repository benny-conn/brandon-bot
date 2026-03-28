package barbuf

import (
	"testing"
	"time"

	"github.com/benny-conn/runbook/strategy"
)

func TestBuffer_PushAndLast(t *testing.T) {
	b := New()
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		b.Push(strategy.Tick{
			Symbol:    "AAPL",
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			Close:     float64(100 + i),
		})
	}

	bars := b.Last("AAPL", 3)
	if len(bars) != 3 {
		t.Fatalf("expected 3 bars, got %d", len(bars))
	}
	// Should be oldest first: 102, 103, 104
	if bars[0].Close != 102 {
		t.Errorf("bars[0].Close = %v, want 102", bars[0].Close)
	}
	if bars[2].Close != 104 {
		t.Errorf("bars[2].Close = %v, want 104", bars[2].Close)
	}
}

func TestBuffer_LastMoreThanAvailable(t *testing.T) {
	b := New()
	b.Push(strategy.Tick{Symbol: "AAPL", Close: 100})
	b.Push(strategy.Tick{Symbol: "AAPL", Close: 101})

	bars := b.Last("AAPL", 10)
	if len(bars) != 2 {
		t.Errorf("expected 2 bars (all available), got %d", len(bars))
	}
}

func TestBuffer_LastUnknownSymbol(t *testing.T) {
	b := New()
	bars := b.Last("AAPL", 5)
	if bars != nil {
		t.Errorf("expected nil for unknown symbol, got %v", bars)
	}
}

func TestBuffer_MultipleSymbols(t *testing.T) {
	b := New()
	b.Push(strategy.Tick{Symbol: "AAPL", Close: 100})
	b.Push(strategy.Tick{Symbol: "GOOG", Close: 200})
	b.Push(strategy.Tick{Symbol: "AAPL", Close: 101})

	aapl := b.Last("AAPL", 10)
	goog := b.Last("GOOG", 10)
	if len(aapl) != 2 {
		t.Errorf("AAPL: expected 2 bars, got %d", len(aapl))
	}
	if len(goog) != 1 {
		t.Errorf("GOOG: expected 1 bar, got %d", len(goog))
	}
}

func TestBuffer_RingWraparound(t *testing.T) {
	b := New()
	// Push more than defaultCapacity (500) bars.
	for i := 0; i < 600; i++ {
		b.Push(strategy.Tick{Symbol: "X", Close: float64(i)})
	}

	bars := b.Last("X", 500)
	if len(bars) != 500 {
		t.Fatalf("expected 500 bars (capacity), got %d", len(bars))
	}
	// Oldest should be bar 100 (first 100 were overwritten).
	if bars[0].Close != 100 {
		t.Errorf("oldest bar = %v, want 100", bars[0].Close)
	}
	// Newest should be bar 599.
	if bars[499].Close != 599 {
		t.Errorf("newest bar = %v, want 599", bars[499].Close)
	}
}

// --- DailyTracker tests ---

func TestDailyTracker_SingleDay(t *testing.T) {
	d := NewDailyTracker()
	d.Update("AAPL", "2025-01-02", 100, 110, 95, 105)
	d.Update("AAPL", "2025-01-02", 106, 112, 98, 108)

	levels := d.Levels("AAPL")
	if levels.TodayOpen != 100 {
		t.Errorf("TodayOpen = %v, want 100", levels.TodayOpen)
	}
	if levels.TodayHigh != 112 {
		t.Errorf("TodayHigh = %v, want 112", levels.TodayHigh)
	}
	if levels.TodayLow != 95 {
		t.Errorf("TodayLow = %v, want 95", levels.TodayLow)
	}
}

func TestDailyTracker_DayRollover(t *testing.T) {
	d := NewDailyTracker()
	// Day 1
	d.Update("AAPL", "2025-01-02", 100, 110, 95, 105)
	d.Update("AAPL", "2025-01-02", 106, 108, 98, 107)

	// Day 2
	d.Update("AAPL", "2025-01-03", 108, 115, 102, 112)

	levels := d.Levels("AAPL")
	// Previous day levels should be from day 1.
	if levels.PrevHigh != 110 {
		t.Errorf("PrevHigh = %v, want 110", levels.PrevHigh)
	}
	if levels.PrevLow != 95 {
		t.Errorf("PrevLow = %v, want 95", levels.PrevLow)
	}
	if levels.PrevClose != 107 {
		t.Errorf("PrevClose = %v, want 107 (last close of day 1)", levels.PrevClose)
	}
	// Today's levels should be from day 2.
	if levels.TodayOpen != 108 {
		t.Errorf("TodayOpen = %v, want 108", levels.TodayOpen)
	}
	if levels.TodayHigh != 115 {
		t.Errorf("TodayHigh = %v, want 115", levels.TodayHigh)
	}
}

func TestDailyTracker_UnknownSymbol(t *testing.T) {
	d := NewDailyTracker()
	levels := d.Levels("UNKNOWN")
	if levels.PrevHigh != 0 || levels.TodayOpen != 0 {
		t.Errorf("expected zero levels for unknown symbol")
	}
}

func TestDailyTracker_MultipleSymbols(t *testing.T) {
	d := NewDailyTracker()
	d.Update("AAPL", "2025-01-02", 100, 110, 95, 105)
	d.Update("GOOG", "2025-01-02", 200, 220, 190, 210)

	aapl := d.Levels("AAPL")
	goog := d.Levels("GOOG")
	if aapl.TodayOpen != 100 {
		t.Errorf("AAPL TodayOpen = %v, want 100", aapl.TodayOpen)
	}
	if goog.TodayOpen != 200 {
		t.Errorf("GOOG TodayOpen = %v, want 200", goog.TodayOpen)
	}
}
