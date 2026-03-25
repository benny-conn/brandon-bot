# Bug: Recovery Seeds Short Positions as Buys

## Problem

`engine/recovery.go` lines 44-48 apply all broker positions as `side: "buy"` regardless of whether they're long or short:

```go
for _, pos := range positions {
    e.portfolio.ApplyFill(strategy.Fill{
        Symbol: pos.Symbol,
        Side:   "buy",
        Qty:    pos.Qty,
        Price:  pos.AvgEntryPrice,
    })
}
```

For short positions (negative qty on the broker), this creates a **long** position in the simulated portfolio instead of a short. This causes:

1. **Cash double-counted** — `ApplyFill` for a buy deducts `qty * price` from cash. But the broker's `account.Cash` already reflects the short sale proceeds. So cash is reduced by a position that was never actually bought.
2. **Wrong position direction** — The portfolio thinks it's long when the broker is short. Any subsequent sell fill to close the short would create a second short instead of flattening.
3. **Equity mismatch** — With the position direction wrong, `UpdateMarketPrice` computes unrealized P&L in the wrong direction.

## Real-World Impact

Observed on a live Alpaca strategy (`c5d7c750`):
- Alpaca account: ~$100,220 equity
- App shows: equity $144,701, cash $174,677
- Strategy capital configured at $50k, but portfolio seeded from full broker cash ($100k)
- Cash inflated to $174k from short sale proceeds accumulating on top of already-correct broker cash

## Fix

Check `pos.Qty` sign (or use a side field if available from the provider) and apply the correct fill side:

```go
for _, pos := range positions {
    side := "buy"
    qty := pos.Qty
    if qty < 0 {
        side = "sell"
        qty = -qty
    }
    e.portfolio.ApplyFill(strategy.Fill{
        Symbol: pos.Symbol,
        Side:   side,
        Qty:    qty,
        Price:  pos.AvgEntryPrice,
    })
}
```

Also check what the Alpaca provider returns for `pos.Qty` on short positions — it should be negative. If the provider normalizes it to positive with a separate side field, use that instead.

## Secondary Issue: Capital vs Account Size

The recovery replaces the configured capital with the real broker cash (line 41):

```go
e.portfolio = portfolio.NewSimulatedPortfolio(account.Cash)
```

This is intentional (the portfolio needs to track real positions), but it means the "capital" setting on a strategy is only used for:
- The `capital` JS global (position sizing in the script)
- Backtest initial capital

The live portfolio always reflects the full broker account, not the strategy's capital allocation. This is confusing when a user sets capital to $50k on a $100k account — the strategy page shows $100k equity instead of $50k.

This isn't necessarily a bug (tracking real positions requires real cash), but the frontend should clarify that equity/cash reflect the broker account, not the strategy allocation.

## Files

- `engine/recovery.go` — short position recovery bug (lines 44-48)
- `internal/portfolio/portfolio.go` — `ApplyFill` cash handling (correct, but affected by wrong inputs)
- `provider/provider.go` — check `Position` struct for qty sign convention
