# brandon-bot

A paper trading backtester and live simulator in Go. Test a day trading strategy against historical data or run it live against a paper account without risking real money.

Supports three providers: **Alpaca** (stocks/ETFs), **Interactive Brokers** (stocks, futures, and more), and **Tradovate** (futures). Swap between them with a single flag.

---

## Prerequisites

- Go 1.22+
- An [Alpaca Markets](https://alpaca.markets) account (free) — for backtesting and Alpaca paper trading
- An [Interactive Brokers](https://www.interactivebrokers.com) account with paper trading enabled — for IBKR live paper trading
- A [Tradovate](https://www.tradovate.com) account with API credentials — for Tradovate live paper trading
- Environment variables set in your shell (see below)

## Environment Variables

**Alpaca** (required for backtesting and `--provider=alpaca`):
```bash
ALPACA_API_KEY=your_key_here
ALPACA_SECRET=your_secret_here
ALPACA_BASE_URL=https://paper-api.alpaca.markets/v2
```

**IBKR** (required for `--provider=ibkr`):
```bash
IBKR_ACCOUNT_ID=DU1234567                  # your paper account ID
IBKR_GATEWAY_URL=https://localhost:5055    # optional, this is the default
```

IBKR also requires **IB Gateway** running locally before starting the bot — see [IBKR Setup](#ibkr-setup) below.

**Tradovate** (required for `--provider=tradovate`):
```bash
TRADOVATE_USERNAME=your_username
TRADOVATE_PASSWORD=your_password
TRADOVATE_APP_ID=your_app_name         # registered in the Tradovate developer portal
TRADOVATE_CID=your_client_id           # from API credentials
TRADOVATE_SEC=your_api_secret          # from API credentials
TRADOVATE_DEVICE_ID=some-stable-uuid   # generate once: uuidgen
TRADOVATE_DEMO=true                    # "false" for live; defaults to demo for safety
# TRADOVATE_APP_VERSION=1.0            # optional, defaults to 1.0
```

---

## Running a Backtest

Fetches historical bars from Alpaca, replays them through the strategy, and prints a full performance report.

```bash
go run cmd/backtest/main.go \
  --strategy=ma_crossover \
  --symbols=AAPL,TSLA \
  --from=2024-01-01 \
  --to=2024-12-31 \
  --timeframe=1d \
  --capital=10000
```

**Flags:**

| Flag          | Default        | Description                              |
| ------------- | -------------- | ---------------------------------------- |
| `--strategy`  | `ma_crossover` | Strategy to run                          |
| `--symbols`   | `AAPL`         | Comma-separated ticker list              |
| `--from`      | required       | Start date (YYYY-MM-DD)                  |
| `--to`        | required       | End date (YYYY-MM-DD)                    |
| `--timeframe` | `1d`           | Bar size: `1m`, `5m`, `15m`, `1h`, `1d`  |
| `--capital`   | `10000`        | Starting capital in USD                  |
| `--feed`      | `iex`          | Alpaca feed: `iex` (free) or `sip` (paid)|

**Output:**

```
Fetching 1d bars for AAPL from 2024-01-01 to 2024-12-31...
Loaded 252 bars

=== Backtest Results ===
Initial capital:  $10000.00
Final equity:     $10054.73
Total return:     0.55%
Max drawdown:     0.62%
Sharpe ratio:     0.0405 (per-bar, not annualized)
Total trades:     3
Win rate:         0.0% (0 W / 3 L)

--- Trade Log ---
[2024-05-03 04:00] BUY  AAPL  qty=5.78  price=$186.65
...

Run saved to database (id=1)
```

Results and fills are saved to SQLite for later review.

---

## Running Paper Trading (Live)

Connects to a broker via WebSocket, streams real-time bars, and places orders against your paper account. This is a long-running process — stays alive during market hours and shuts down cleanly on `Ctrl+C`.

### Alpaca

```bash
go run cmd/paper/main.go \
  --provider=alpaca \
  --strategy=ma_crossover \
  --symbols=AAPL,TSLA \
  --timeframe=1m \
  --capital=10000 \
  --feed=iex
```

### IBKR

```bash
go run cmd/paper/main.go \
  --provider=ibkr \
  --strategy=ma_crossover \
  --symbols=AAPL \
  --timeframe=1m \
  --capital=10000
```

### Tradovate

```bash
go run cmd/paper/main.go \
  --provider=tradovate \
  --strategy=ma_crossover \
  --symbols=ESZ4 \
  --timeframe=1m \
  --capital=50000
```

Note: Tradovate uses specific contract symbols (e.g. `ESZ4` for E-Mini S&P December 2024, `NQZ4` for E-Mini Nasdaq). Month codes: F=Jan G=Feb H=Mar J=Apr K=May M=Jun N=Jul Q=Aug U=Sep V=Oct X=Nov Z=Dec.

**Flags:**

| Flag           | Default        | Description                                                                               |
| -------------- | -------------- | ----------------------------------------------------------------------------------------- |
| `--provider`   | `alpaca`       | Data + execution provider: `alpaca` or `ibkr`                                             |
| `--strategy`   | `ma_crossover` | Strategy to run                                                                           |
| `--symbols`    | `AAPL`         | Comma-separated ticker list                                                               |
| `--timeframe`  | `1m`           | Bar size: `1s`, `1m`, `5m`, `15m`, `1h`, `1d`                                            |
| `--capital`    | `10000`        | Starting capital (used only on first run — subsequent runs seed from real account state)  |
| `--feed`       | `iex`          | Alpaca feed: `iex` (free) or `sip` (paid) — ignored for IBKR                             |

**On startup**, the engine automatically:

1. Fetches your real account balance and open positions from the broker
2. Replays recent historical bars to warm up strategy indicators (EMAs, etc.)
3. Injects any existing positions into the strategy so it knows what it's holding
4. Then begins the live WebSocket stream

This means **restarting the bot mid-session is safe** — it picks up from the correct state rather than thinking it has no positions.

---

## IBKR Setup

IBKR paper trading requires **IB Gateway** running locally. IB Gateway is a lightweight headless process (no charts UI) that the bot connects to via the Client Portal API.

1. Download **IB Gateway** from [ibkr.com](https://www.interactivebrokers.com/en/trading/ibgateway.php) (use the stable/latest channel)
2. Log in with your **paper account** credentials
3. IB Gateway listens on `https://localhost:5055` — leave it running while the bot is active
4. Set `IBKR_ACCOUNT_ID` to your paper account ID (format `DU1234567`, visible after login)
5. Run the bot with `--provider=ibkr`

The bot sends a keep-alive ping to IB Gateway every 55 seconds to maintain the session. You do not need to do anything else to keep it connected.

**Getting a paper account:**
1. Sign up for a live IBKR account at ibkr.com and wait for approval (~1–3 business days)
2. After approval, log into the Client Portal → **Settings → Paper Trading Account** to create one
3. The paper account gets its own separate login credentials

---

## Tradovate Setup

Tradovate requires API credentials from their developer portal. Unlike IBKR, no local gateway process is needed — the bot connects directly to Tradovate's cloud API.

1. Sign up at [tradovate.com](https://www.tradovate.com) and open a **Sim (demo) account**
2. Go to **Account → API Access** in the Tradovate platform to request API credentials
3. You'll receive a `cid` (client ID) and `sec` (API secret) — set these as `TRADOVATE_CID` and `TRADOVATE_SEC`
4. Generate a stable device ID once and store it: `uuidgen` (macOS/Linux)
5. Set `TRADOVATE_DEMO=true` (or omit it — demo is the default)
6. Run the bot with `--provider=tradovate`

Tokens expire every 90 minutes; the bot renews them automatically every 85 minutes.

**Tradovate vs IBKR for the target strategy:**
- Tradovate is purpose-built for futures (ES, NQ, CL, etc.) with a clean modern API
- WebSocket fills arrive natively via `user/syncrequest` — no polling needed
- For sub-second bar data, use `--timeframe=1s` which maps to tick bars; or use `SubscribeTrades` for individual quote updates

---

## Project Structure

```
brandon-bot/
├── cmd/
│   ├── backtest/main.go          # CLI: run a backtest
│   └── paper/main.go             # CLI: run paper trading live
├── internal/
│   ├── provider/
│   │   ├── provider.go           # MarketData + Execution interfaces and shared types
│   │   ├── alpaca/
│   │   │   └── alpaca.go         # Alpaca implementation (stocks/ETFs, iex/sip feeds)
│   │   ├── ibkr/
│   │   │   ├── client.go         # IB Gateway HTTP + WebSocket client
│   │   │   └── ibkr.go           # IBKR implementation (stocks, futures)
│   │   └── tradovate/
│   │       ├── auth.go           # Token acquisition and 85-minute renewal
│   │       ├── ws.go             # Tradovate WebSocket framing + heartbeat
│   │       └── tradovate.go      # Tradovate implementation (futures)
│   ├── strategy/
│   │   ├── strategy.go           # Core interfaces: Strategy, Portfolio, Tick, Order, Fill
│   │   ├── ma_crossover.go       # Example: 9/21 EMA crossover strategy
│   │   └── rsi_pullback.go       # Example: RSI pullback with 200-SMA trend filter
│   ├── portfolio/
│   │   └── portfolio.go          # Tracks cash, positions, P&L
│   ├── backtest/
│   │   └── engine.go             # Backtesting engine + performance metrics
│   ├── paper/
│   │   ├── engine.go             # Live paper trading engine (WebSocket event loop)
│   │   └── recovery.go           # Startup state recovery from broker
│   ├── risk/
│   │   └── risk.go               # Position sizing helpers
│   └── db/
│       └── db.go                 # SQLite logging (runs, fills, snapshots)
├── go.mod
└── go.sum
```

---

## Adding a Custom Strategy

1. Create `internal/strategy/my_strategy.go`
2. Implement the `Strategy` interface:

```go
type MyStrategy struct {
    // your internal state (indicators, position tracking, etc.)
}

func (s *MyStrategy) Name() string { return "my_strategy" }

// Called on every bar. Return any orders to place — the engine handles execution.
// All state lives inside the strategy; the engine only sees what orders come back.
func (s *MyStrategy) OnTick(tick strategy.Tick, portfolio strategy.Portfolio) []strategy.Order {
    // your logic here
    return nil
}

// Called when an order is filled. Update your internal position tracking.
func (s *MyStrategy) OnFill(fill strategy.Fill) {}

// Optional: subscribe to individual trade prints instead of (or in addition to) bars.
// If this method exists, the paper engine automatically subscribes to the trade stream.
// OnTick is still called for bar events — implement it as a no-op if you don't need it.
func (s *MyStrategy) OnTrade(trade strategy.Trade, portfolio strategy.Portfolio) []strategy.Order {
    // react to every individual trade print (price, size, exchange, conditions)
    return nil
}

// Optional: support safe restarts mid-position.
// Called on startup if the bot already holds a position when it comes back up.
func (s *MyStrategy) SeedPosition(symbol string, qty, avgCost float64) {}
```

3. Register it in both `cmd/backtest/main.go` and `cmd/paper/main.go`:

```go
func resolveStrategy(name string) (strategy.Strategy, error) {
    switch name {
    case "ma_crossover":
        return strategy.NewMACrossover(), nil
    case "my_strategy":           // add this
        return strategy.NewMyStrategy(), nil
    default:
        return nil, fmt.Errorf("available strategies: ma_crossover, my_strategy")
    }
}
```

4. Run it:

```bash
go run cmd/backtest/main.go --strategy=my_strategy --symbols=AAPL --from=2024-01-01 --to=2024-12-31
```

---

## Built-in Strategies

### MA Crossover

A simple 9/21 exponential moving average crossover, included as a working example to validate the engine. **Not intended for real use.**

- **Buy signal**: 9-period EMA crosses above 21-period EMA → buy 10% of available cash
- **Sell signal**: 9-period EMA crosses below 21-period EMA → sell full position
- **Stop loss**: price drops 2% below entry → sell immediately

### RSI Pullback

RSI-based pullback strategy with a 200-period SMA trend filter.

---

## Database

Results are logged to SQLite at `DATABASE_PATH` (default: `./trading_bot.db`).

| Table              | Contents                                  |
| ------------------ | ----------------------------------------- |
| `backtest_runs`    | Summary metrics for each backtest run     |
| `backtest_fills`   | Individual fills from each run            |
| `paper_orders`     | Orders submitted during paper trading     |
| `paper_fills`      | Fill confirmations from the broker        |
| `paper_snapshots`  | Portfolio snapshots taken after each fill |

---

## Live Trading (Future)

When ready to trade with real money, the architecture is identical — just swap credentials:

**Alpaca live:**
```bash
ALPACA_BASE_URL=https://api.alpaca.markets/v2
```

**IBKR live:** point IB Gateway at your live account instead of paper.

A `cmd/live/main.go` will be added with additional safeguards (position limits, kill switch, etc.) before this is used.
