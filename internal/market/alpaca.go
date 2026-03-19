package market

import (
	"fmt"
	"os"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"

	"brandon-bot/internal/strategy"
)

// Client wraps the Alpaca market data SDK.
type Client struct {
	md *marketdata.Client
}

// NewClient creates an Alpaca market data client using ALPACA_API_KEY and ALPACA_SECRET from env.
func NewClient() *Client {
	return &Client{
		md: marketdata.NewClient(marketdata.ClientOpts{
			APIKey:    os.Getenv("ALPACA_API_KEY"),
			APISecret: os.Getenv("ALPACA_SECRET"),
		}),
	}
}

// FetchBars retrieves historical OHLCV bars for a single symbol and returns them as Ticks.
// start/end are in UTC. tf is the bar timeframe (e.g. marketdata.OneMin).
// feed selects the data source ("iex" for free tier, "sip" for paid); empty string uses the Alpaca default.
func (c *Client) FetchBars(symbol string, start, end time.Time, tf marketdata.TimeFrame, feed marketdata.Feed) ([]strategy.Tick, error) {
	bars, err := c.md.GetBars(symbol, marketdata.GetBarsRequest{
		TimeFrame: tf,
		Start:     start,
		End:       end,
		Feed:      feed,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching bars for %s: %w", symbol, err)
	}

	ticks := make([]strategy.Tick, len(bars))
	for i, bar := range bars {
		ticks[i] = strategy.Tick{
			Symbol:    symbol,
			Timestamp: bar.Timestamp,
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			Volume:    int64(bar.Volume),
		}
	}
	return ticks, nil
}

// FetchBarsForSymbols fetches historical bars for multiple symbols concurrently
// and returns them merged and sorted by timestamp, ready for backtesting replay.
func (c *Client) FetchBarsForSymbols(symbols []string, start, end time.Time, tf marketdata.TimeFrame, feed marketdata.Feed) ([]strategy.Tick, error) {
	type result struct {
		ticks []strategy.Tick
		err   error
	}

	ch := make(chan result, len(symbols))
	for _, sym := range symbols {
		go func(s string) {
			ticks, err := c.FetchBars(s, start, end, tf, feed)
			ch <- result{ticks, err}
		}(sym)
	}

	var all []strategy.Tick
	for range symbols {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		all = append(all, r.ticks...)
	}

	SortTicks(all)
	return all, nil
}

// ParseTimeFrame converts CLI strings like "1m", "5m", "1h", "1d" to an Alpaca TimeFrame.
func ParseTimeFrame(s string) (marketdata.TimeFrame, error) {
	switch s {
	case "1m":
		return marketdata.NewTimeFrame(1, marketdata.Min), nil
	case "5m":
		return marketdata.NewTimeFrame(5, marketdata.Min), nil
	case "15m":
		return marketdata.NewTimeFrame(15, marketdata.Min), nil
	case "1h":
		return marketdata.NewTimeFrame(1, marketdata.Hour), nil
	case "1d":
		return marketdata.NewTimeFrame(1, marketdata.Day), nil
	default:
		return marketdata.TimeFrame{}, fmt.Errorf("unsupported timeframe %q — use 1m, 5m, 15m, 1h, or 1d", s)
	}
}
