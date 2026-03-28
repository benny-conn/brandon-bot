package orderutil

import (
	"testing"

	"github.com/benny-conn/runbook/strategy"
)

func TestValidateOrders_DropsEmptySymbol(t *testing.T) {
	orders := []strategy.Order{
		{Symbol: "", Side: "buy", Qty: 1},
	}
	var reasons []string
	valid := ValidateOrders(orders, 0, func(r string) { reasons = append(reasons, r) })
	if len(valid) != 0 {
		t.Errorf("expected 0 valid orders, got %d", len(valid))
	}
	if len(reasons) != 1 {
		t.Errorf("expected 1 rejection, got %d", len(reasons))
	}
}

func TestValidateOrders_DropsZeroQty(t *testing.T) {
	orders := []strategy.Order{
		{Symbol: "AAPL", Side: "buy", Qty: 0},
		{Symbol: "AAPL", Side: "buy", Qty: -5},
	}
	valid := ValidateOrders(orders, 0, nil)
	if len(valid) != 0 {
		t.Errorf("expected 0 valid orders, got %d", len(valid))
	}
}

func TestValidateOrders_CapsQty(t *testing.T) {
	orders := []strategy.Order{
		{Symbol: "AAPL", Side: "buy", Qty: 100},
	}
	valid := ValidateOrders(orders, 10, nil)
	if len(valid) != 1 {
		t.Fatalf("expected 1 valid order, got %d", len(valid))
	}
	if valid[0].Qty != 10 {
		t.Errorf("Qty = %v, want 10 (capped)", valid[0].Qty)
	}
}

func TestValidateOrders_PassesValid(t *testing.T) {
	orders := []strategy.Order{
		{Symbol: "AAPL", Side: "buy", Qty: 5},
		{Symbol: "GOOG", Side: "sell", Qty: 3},
	}
	valid := ValidateOrders(orders, 0, nil)
	if len(valid) != 2 {
		t.Errorf("expected 2 valid orders, got %d", len(valid))
	}
}

func TestValidateOrders_NilRejectFunc(t *testing.T) {
	orders := []strategy.Order{
		{Symbol: "", Side: "buy", Qty: 1},
	}
	// Should not panic with nil onReject.
	valid := ValidateOrders(orders, 0, nil)
	if len(valid) != 0 {
		t.Errorf("expected 0 valid orders, got %d", len(valid))
	}
}

func TestBuildFlattenOrders_Long(t *testing.T) {
	positions := []strategy.Position{
		{Symbol: "AAPL", Qty: 10},
	}
	orders := BuildFlattenOrders(positions, "test")
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].Side != "sell" || orders[0].Qty != 10 {
		t.Errorf("got side=%q qty=%v, want sell/10", orders[0].Side, orders[0].Qty)
	}
	if orders[0].OrderType != "market" {
		t.Errorf("OrderType = %q, want market", orders[0].OrderType)
	}
}

func TestBuildFlattenOrders_Short(t *testing.T) {
	positions := []strategy.Position{
		{Symbol: "AAPL", Qty: -5},
	}
	orders := BuildFlattenOrders(positions, "test")
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].Side != "buy" || orders[0].Qty != 5 {
		t.Errorf("got side=%q qty=%v, want buy/5", orders[0].Side, orders[0].Qty)
	}
}

func TestBuildFlattenOrders_SkipsFlat(t *testing.T) {
	positions := []strategy.Position{
		{Symbol: "AAPL", Qty: 0},
	}
	orders := BuildFlattenOrders(positions, "test")
	if len(orders) != 0 {
		t.Errorf("expected 0 orders for flat position, got %d", len(orders))
	}
}

func TestBuildFlattenOrders_Multiple(t *testing.T) {
	positions := []strategy.Position{
		{Symbol: "AAPL", Qty: 10},
		{Symbol: "GOOG", Qty: -3},
		{Symbol: "MSFT", Qty: 0},
	}
	orders := BuildFlattenOrders(positions, "test")
	if len(orders) != 2 {
		t.Errorf("expected 2 orders (AAPL sell + GOOG buy), got %d", len(orders))
	}
}

func TestBuildFlattenOrders_Empty(t *testing.T) {
	orders := BuildFlattenOrders(nil, "test")
	if len(orders) != 0 {
		t.Errorf("expected 0 orders, got %d", len(orders))
	}
}
