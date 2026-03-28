package bracket

import (
	"testing"

	"github.com/benny-conn/runbook/strategy"
)

func TestNewFromOrder_NoDistances(t *testing.T) {
	order := strategy.Order{Symbol: "AAPL", Side: "buy", Qty: 10}
	if NewFromOrder(order, 100) != nil {
		t.Error("expected nil when no TP/SL distances")
	}
}

func TestNewFromOrder_BuyEntry(t *testing.T) {
	order := strategy.Order{Symbol: "AAPL", Side: "buy", Qty: 10, TPDistance: 5, SLDistance: 3}
	b := NewFromOrder(order, 100)
	if b == nil {
		t.Fatal("expected bracket")
	}
	if b.TakeProfit != 105 {
		t.Errorf("TP = %v, want 105", b.TakeProfit)
	}
	if b.StopLoss != 97 {
		t.Errorf("SL = %v, want 97", b.StopLoss)
	}
	if b.Side != "buy" {
		t.Errorf("Side = %q, want buy", b.Side)
	}
}

func TestNewFromOrder_SellEntry(t *testing.T) {
	order := strategy.Order{Symbol: "AAPL", Side: "sell", Qty: 5, TPDistance: 4, SLDistance: 2}
	b := NewFromOrder(order, 100)
	if b == nil {
		t.Fatal("expected bracket")
	}
	// Short entry: TP is below, SL is above.
	if b.TakeProfit != 96 {
		t.Errorf("TP = %v, want 96", b.TakeProfit)
	}
	if b.StopLoss != 102 {
		t.Errorf("SL = %v, want 102", b.StopLoss)
	}
}

func TestNewFromOrder_TPOnly(t *testing.T) {
	order := strategy.Order{Symbol: "X", Side: "buy", Qty: 1, TPDistance: 10}
	b := NewFromOrder(order, 50)
	if b == nil {
		t.Fatal("expected bracket")
	}
	if b.TakeProfit != 60 {
		t.Errorf("TP = %v, want 60", b.TakeProfit)
	}
	if b.StopLoss != 0 {
		t.Errorf("SL = %v, want 0", b.StopLoss)
	}
}

func TestNewFromOrder_SLOnly(t *testing.T) {
	order := strategy.Order{Symbol: "X", Side: "sell", Qty: 1, SLDistance: 5}
	b := NewFromOrder(order, 200)
	if b == nil {
		t.Fatal("expected bracket")
	}
	if b.TakeProfit != 0 {
		t.Errorf("TP = %v, want 0", b.TakeProfit)
	}
	if b.StopLoss != 205 {
		t.Errorf("SL = %v, want 205", b.StopLoss)
	}
}

func TestCheck_LongTP(t *testing.T) {
	b := &Pending{Symbol: "X", Side: "buy", Qty: 1, TakeProfit: 110, StopLoss: 90}
	// High touches TP.
	tr := b.Check(110, 95)
	if tr == nil {
		t.Fatal("expected trigger")
	}
	if tr.Side != "sell" || tr.Price != 110 {
		t.Errorf("got side=%q price=%v, want sell @ 110", tr.Side, tr.Price)
	}
}

func TestCheck_LongSL(t *testing.T) {
	b := &Pending{Symbol: "X", Side: "buy", Qty: 1, TakeProfit: 110, StopLoss: 90}
	// Low touches SL.
	tr := b.Check(105, 90)
	if tr == nil {
		t.Fatal("expected trigger")
	}
	if tr.Side != "sell" || tr.Price != 90 {
		t.Errorf("got side=%q price=%v, want sell @ 90", tr.Side, tr.Price)
	}
}

func TestCheck_ShortTP(t *testing.T) {
	b := &Pending{Symbol: "X", Side: "sell", Qty: 2, TakeProfit: 90, StopLoss: 110}
	// Low touches TP.
	tr := b.Check(105, 90)
	if tr == nil {
		t.Fatal("expected trigger")
	}
	if tr.Side != "buy" || tr.Price != 90 {
		t.Errorf("got side=%q price=%v, want buy @ 90", tr.Side, tr.Price)
	}
}

func TestCheck_ShortSL(t *testing.T) {
	b := &Pending{Symbol: "X", Side: "sell", Qty: 2, TakeProfit: 90, StopLoss: 110}
	// High touches SL.
	tr := b.Check(110, 95)
	if tr == nil {
		t.Fatal("expected trigger")
	}
	if tr.Side != "buy" || tr.Price != 110 {
		t.Errorf("got side=%q price=%v, want buy @ 110", tr.Side, tr.Price)
	}
}

func TestCheck_NoTrigger(t *testing.T) {
	b := &Pending{Symbol: "X", Side: "buy", Qty: 1, TakeProfit: 110, StopLoss: 90}
	// Price stays in range.
	if tr := b.Check(105, 95); tr != nil {
		t.Errorf("expected no trigger, got %+v", tr)
	}
}

func TestCheck_TPPriorityOverSL(t *testing.T) {
	// Both TP and SL touched in the same bar — TP wins (checked first).
	b := &Pending{Symbol: "X", Side: "buy", Qty: 1, TakeProfit: 105, StopLoss: 95}
	tr := b.Check(110, 90)
	if tr == nil {
		t.Fatal("expected trigger")
	}
	// TP is checked first for long entries.
	if tr.Price != 105 {
		t.Errorf("price = %v, want 105 (TP priority)", tr.Price)
	}
}

func TestCheckAll_MultipleSymbols(t *testing.T) {
	brackets := []Pending{
		{Symbol: "A", Side: "buy", Qty: 1, TakeProfit: 110, StopLoss: 90},
		{Symbol: "B", Side: "sell", Qty: 2, TakeProfit: 45, StopLoss: 55},
	}

	// Only symbol A triggers.
	remaining, fills := CheckAll(brackets, "A", 115, 95)
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(remaining))
	}
	if remaining[0].Symbol != "B" {
		t.Errorf("remaining symbol = %q, want B", remaining[0].Symbol)
	}
	if len(fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(fills))
	}
	if fills[0].Symbol != "A" || fills[0].Side != "sell" {
		t.Errorf("fill = %+v, want A/sell", fills[0])
	}
}

func TestCheckAll_NoTriggers(t *testing.T) {
	brackets := []Pending{
		{Symbol: "A", Side: "buy", Qty: 1, TakeProfit: 200, StopLoss: 50},
	}
	remaining, fills := CheckAll(brackets, "A", 150, 100)
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining, got %d", len(remaining))
	}
	if len(fills) != 0 {
		t.Errorf("expected 0 fills, got %d", len(fills))
	}
}

func TestCheckAll_Empty(t *testing.T) {
	remaining, fills := CheckAll(nil, "A", 100, 90)
	if len(remaining) != 0 || len(fills) != 0 {
		t.Errorf("expected empty results for nil input")
	}
}
