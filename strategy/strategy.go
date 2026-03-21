package strategy

import "time"

// Tick represents a single OHLCV bar for a symbol.
type Tick struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
}

// Order is a trade instruction returned by a strategy.
type Order struct {
	Symbol     string
	Side       string  // "buy" or "sell"
	Qty        float64
	OrderType  string  // "market", "limit", "stop", or "stop_limit"
	LimitPrice float64 // limit price for limit and stop-limit orders
	StopPrice  float64 // trigger price for stop and stop-limit orders
	StopLoss   float64 // broker-native bracket stop loss price (0 = disabled)
	TakeProfit float64 // broker-native bracket take profit price (0 = disabled)
	Reason     string  // for logging/debugging
}

// Fill is the result of an executed order.
type Fill struct {
	Symbol    string
	Side      string
	Qty       float64
	Price     float64
	Timestamp time.Time
}

// Position represents an open holding in the portfolio.
type Position struct {
	Symbol       string
	Qty          float64
	AvgCost      float64
	MarketValue  float64
	UnrealizedPL float64
}

// Portfolio is a read-only view of the current account state passed into OnTick.
type Portfolio interface {
	Cash() float64
	Equity() float64
	Position(symbol string) *Position
	Positions() []Position
	TotalPL() float64
}

// Trade represents a single real-time trade print from the exchange.
type Trade struct {
	Symbol     string
	Timestamp  time.Time
	Price      float64
	Size       uint32
	Exchange   string
	Conditions []string
}

// Configurable is an optional interface a strategy can implement to accept
// a JSON config file passed via --config on the CLI. Configure is called once
// after the strategy is constructed and before the first OnTick, so it can
// override any defaults set in the constructor.
// Partial configs are fine — only fields present in the JSON are updated;
// missing fields keep their constructor defaults.
type Configurable interface {
	Configure(data []byte) error
}

// Strategy is implemented by any trading algorithm.
// The engine calls OnTick on every price update and executes any returned orders.
// All strategy state must live inside the Strategy implementation — the engine is stateless w.r.t. strategy internals.
type Strategy interface {
	Name() string
	OnTick(tick Tick, portfolio Portfolio) []Order
	OnFill(fill Fill)
}

// TradeSubscriber is an optional interface a strategy can implement to receive
// individual trade prints instead of (or in addition to) completed bars.
// If the strategy implements this, the paper engine will also subscribe to
// the trade stream for the requested symbols.
// OnTick is still called for bar events — implement it as a no-op if not needed.
type TradeSubscriber interface {
	OnTrade(trade Trade, portfolio Portfolio) []Order
}

// Quote represents a real-time bid/ask update from the exchange.
type Quote struct {
	Symbol    string
	Timestamp time.Time
	BidPrice  float64
	BidSize   float64
	AskPrice  float64
	AskSize   float64
}

// QuoteSubscriber is an optional interface a strategy can implement to receive
// real-time bid/ask quote updates. If implemented, the engine subscribes to the
// quote stream for the requested symbols in addition to bars.
// Useful for spread-aware entry logic and limit order placement.
type QuoteSubscriber interface {
	OnQuote(quote Quote, portfolio Portfolio) []Order
}
