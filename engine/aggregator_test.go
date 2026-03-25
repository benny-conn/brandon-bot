package engine

import (
	"testing"
	"time"

	"github.com/benny-conn/brandon-bot/strategy"
)

func TestParseTimeframe(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		err   bool
	}{
		{"1s", time.Second, false},
		{"1m", time.Minute, false},
		{"5m", 5 * time.Minute, false},
		{"15m", 15 * time.Minute, false},
		{"1h", time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"3m", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseTimeframe(tt.input)
		if tt.err && err == nil {
			t.Errorf("ParseTimeframe(%q): expected error", tt.input)
		}
		if !tt.err && got != tt.want {
			t.Errorf("ParseTimeframe(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSortTimeframes(t *testing.T) {
	sorted, err := SortTimeframes([]string{"1h", "1m", "5m"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1m", "5m", "1h"}
	for i, tf := range sorted {
		if tf != want[i] {
			t.Errorf("index %d: got %q, want %q", i, tf, want[i])
		}
	}
}

func TestBarAggregator_5mFrom1m(t *testing.T) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	var emitted []strategy.Tick
	agg := NewBarAggregator("5m", 5*time.Minute, func(_ string, tick strategy.Tick) {
		emitted = append(emitted, tick)
	})

	// Feed 6 one-minute bars: 10:00, 10:01, 10:02, 10:03, 10:04, 10:05.
	// The 10:05 bar crosses into a new 5m window, so the 10:00-10:04 bar should emit.
	for i := 0; i < 6; i++ {
		agg.Update(strategy.Tick{
			Symbol:    "AAPL",
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			Open:      float64(100 + i),
			High:      float64(105 + i),
			Low:       float64(95 + i),
			Close:     float64(101 + i),
			Volume:    int64(1000 + i),
		})
	}

	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted bar, got %d", len(emitted))
	}

	bar := emitted[0]
	if bar.Symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", bar.Symbol)
	}
	if bar.Timestamp != base {
		t.Errorf("timestamp = %v, want %v", bar.Timestamp, base)
	}
	// Open should be from first bar (10:00): 100
	if bar.Open != 100 {
		t.Errorf("open = %v, want 100", bar.Open)
	}
	// High should be max of all highs: 109 (105+4)
	if bar.High != 109 {
		t.Errorf("high = %v, want 109", bar.High)
	}
	// Low should be min of all lows: 95 (95+0)
	if bar.Low != 95 {
		t.Errorf("low = %v, want 95", bar.Low)
	}
	// Close should be from last bar in window (10:04): 105
	if bar.Close != 105 {
		t.Errorf("close = %v, want 105", bar.Close)
	}
	// Volume should be sum: 1000+1001+1002+1003+1004 = 5010
	if bar.Volume != 5010 {
		t.Errorf("volume = %v, want 5010", bar.Volume)
	}
}

func TestBarAggregator_MultipleSymbols(t *testing.T) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	var emitted []strategy.Tick
	agg := NewBarAggregator("5m", 5*time.Minute, func(_ string, tick strategy.Tick) {
		emitted = append(emitted, tick)
	})

	// Interleave AAPL and MSFT bars.
	for i := 0; i < 6; i++ {
		ts := base.Add(time.Duration(i) * time.Minute)
		agg.Update(strategy.Tick{Symbol: "AAPL", Timestamp: ts, Open: 100, High: 110, Low: 90, Close: 100, Volume: 100})
		agg.Update(strategy.Tick{Symbol: "MSFT", Timestamp: ts, Open: 200, High: 210, Low: 190, Close: 200, Volume: 200})
	}

	// Both symbols should emit a 5m bar when the window changes at 10:05.
	if len(emitted) != 2 {
		t.Fatalf("expected 2 emitted bars, got %d", len(emitted))
	}

	symbols := map[string]bool{}
	for _, bar := range emitted {
		symbols[bar.Symbol] = true
	}
	if !symbols["AAPL"] || !symbols["MSFT"] {
		t.Errorf("expected bars for both AAPL and MSFT, got %v", symbols)
	}
}

func TestBarAggregator_Flush(t *testing.T) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	var emitted []strategy.Tick
	agg := NewBarAggregator("5m", 5*time.Minute, func(_ string, tick strategy.Tick) {
		emitted = append(emitted, tick)
	})

	// Feed 3 bars — not enough to trigger a window change.
	for i := 0; i < 3; i++ {
		agg.Update(strategy.Tick{
			Symbol:    "AAPL",
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			Open:      100, High: 110, Low: 90, Close: 100, Volume: 100,
		})
	}

	if len(emitted) != 0 {
		t.Fatalf("expected no emitted bars before flush, got %d", len(emitted))
	}

	agg.Flush()

	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted bar after flush, got %d", len(emitted))
	}
}
