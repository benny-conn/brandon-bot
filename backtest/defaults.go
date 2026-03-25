package backtest

import "time"

// DefaultDuration returns a sensible backtest duration for the given bar
// timeframe. Longer timeframes need more calendar days to produce enough
// bars for warmup + meaningful trading.
func DefaultDuration(timeframe string) time.Duration {
	const day = 24 * time.Hour
	switch timeframe {
	case "1s", "15s", "30s":
		return 7 * day
	case "1m", "5m":
		return 30 * day
	case "15m", "30m", "1h":
		return 60 * day
	case "1d":
		return 365 * day
	default:
		return 30 * day
	}
}
