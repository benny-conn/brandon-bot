package script

import (
	"math"
	"sort"

	"github.com/dop251/goja"
)

// registerTA injects the `ta` global object into the Goja VM with stateless
// technical analysis helper functions. All functions are pure — they take
// slices of numbers and return computed values without internal state.
func registerTA(vm *goja.Runtime) {
	ta := vm.NewObject()

	// -----------------------------------------------------------------------
	// Moving Averages
	// -----------------------------------------------------------------------

	// ta.sma(values, period) → number
	ta.Set("sma", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		sum := 0.0
		for _, v := range vals[len(vals)-period:] {
			sum += v
		}
		return vm.ToValue(sum / float64(period))
	})

	// ta.ema(values, period) → number
	ta.Set("ema", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		k := 2.0 / float64(period+1)
		// Seed with SMA of first `period` values.
		ema := 0.0
		for i := 0; i < period; i++ {
			ema += vals[i]
		}
		ema /= float64(period)
		for i := period; i < len(vals); i++ {
			ema = vals[i]*k + ema*(1-k)
		}
		return vm.ToValue(ema)
	})

	// ta.wma(values, period) → number
	ta.Set("wma", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		window := vals[len(vals)-period:]
		num, den := 0.0, 0.0
		for i, v := range window {
			w := float64(i + 1)
			num += v * w
			den += w
		}
		return vm.ToValue(num / den)
	})

	// ta.vwap(highs, lows, closes, volumes) → number
	ta.Set("vwap", func(call goja.FunctionCall) goja.Value {
		highs := toFloat64Slice(call.Argument(0).Export())
		lows := toFloat64Slice(call.Argument(1).Export())
		closes := toFloat64Slice(call.Argument(2).Export())
		volumes := toFloat64Slice(call.Argument(3).Export())
		n := minLen(len(highs), len(lows), len(closes), len(volumes))
		if n == 0 {
			return vm.ToValue(math.NaN())
		}
		cumTPV, cumVol := 0.0, 0.0
		for i := 0; i < n; i++ {
			tp := (highs[i] + lows[i] + closes[i]) / 3.0
			cumTPV += tp * volumes[i]
			cumVol += volumes[i]
		}
		if cumVol == 0 {
			return vm.ToValue(math.NaN())
		}
		return vm.ToValue(cumTPV / cumVol)
	})

	// -----------------------------------------------------------------------
	// Momentum / Oscillators
	// -----------------------------------------------------------------------

	// ta.rsi(closes, period) → number (0-100)
	ta.Set("rsi", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period+1 {
			return vm.ToValue(math.NaN())
		}
		// Calculate initial average gain/loss over first `period` changes.
		avgGain, avgLoss := 0.0, 0.0
		for i := 1; i <= period; i++ {
			diff := vals[i] - vals[i-1]
			if diff > 0 {
				avgGain += diff
			} else {
				avgLoss -= diff
			}
		}
		avgGain /= float64(period)
		avgLoss /= float64(period)
		// Smoothed RSI (Wilder's method).
		for i := period + 1; i < len(vals); i++ {
			diff := vals[i] - vals[i-1]
			gain, loss := 0.0, 0.0
			if diff > 0 {
				gain = diff
			} else {
				loss = -diff
			}
			avgGain = (avgGain*float64(period-1) + gain) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		}
		if avgLoss == 0 {
			return vm.ToValue(100.0)
		}
		rs := avgGain / avgLoss
		return vm.ToValue(100.0 - 100.0/(1.0+rs))
	})

	// ta.macd(closes, fast, slow, signal) → { macd, signal, histogram }
	ta.Set("macd", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		fast := toInt(call.Argument(1).Export())
		slow := toInt(call.Argument(2).Export())
		sig := toInt(call.Argument(3).Export())
		if slow <= 0 || fast <= 0 || sig <= 0 || len(vals) < slow {
			return vm.ToValue(math.NaN())
		}
		emaFast := calcEMASeries(vals, fast)
		emaSlow := calcEMASeries(vals, slow)
		n := minLen(len(emaFast), len(emaSlow))
		if n == 0 {
			return vm.ToValue(math.NaN())
		}
		// MACD line = fast EMA - slow EMA (aligned from the end)
		macdLine := make([]float64, n)
		fOff := len(emaFast) - n
		sOff := len(emaSlow) - n
		for i := 0; i < n; i++ {
			macdLine[i] = emaFast[fOff+i] - emaSlow[sOff+i]
		}
		if len(macdLine) < sig {
			return vm.ToValue(math.NaN())
		}
		sigLine := calcEMASeries(macdLine, sig)
		macdVal := macdLine[len(macdLine)-1]
		sigVal := sigLine[len(sigLine)-1]
		result := vm.NewObject()
		result.Set("macd", macdVal)
		result.Set("signal", sigVal)
		result.Set("histogram", macdVal-sigVal)
		return result
	})

	// ta.stochastic(highs, lows, closes, kPeriod, dPeriod) → { k, d }
	ta.Set("stochastic", func(call goja.FunctionCall) goja.Value {
		highs := toFloat64Slice(call.Argument(0).Export())
		lows := toFloat64Slice(call.Argument(1).Export())
		closes := toFloat64Slice(call.Argument(2).Export())
		kPeriod := toInt(call.Argument(3).Export())
		dPeriod := toInt(call.Argument(4).Export())
		n := minLen(len(highs), len(lows), len(closes))
		if kPeriod <= 0 || dPeriod <= 0 || n < kPeriod {
			return vm.ToValue(math.NaN())
		}
		// Compute %K series
		kSeries := make([]float64, 0, n-kPeriod+1)
		for i := kPeriod - 1; i < n; i++ {
			hh := highs[i-kPeriod+1]
			ll := lows[i-kPeriod+1]
			for j := i - kPeriod + 2; j <= i; j++ {
				if highs[j] > hh {
					hh = highs[j]
				}
				if lows[j] < ll {
					ll = lows[j]
				}
			}
			if hh == ll {
				kSeries = append(kSeries, 50.0)
			} else {
				kSeries = append(kSeries, 100.0*(closes[i]-ll)/(hh-ll))
			}
		}
		if len(kSeries) < dPeriod {
			return vm.ToValue(math.NaN())
		}
		// %D = SMA of %K
		dVal := 0.0
		for _, v := range kSeries[len(kSeries)-dPeriod:] {
			dVal += v
		}
		dVal /= float64(dPeriod)
		result := vm.NewObject()
		result.Set("k", kSeries[len(kSeries)-1])
		result.Set("d", dVal)
		return result
	})

	// ta.cci(highs, lows, closes, period) → number
	ta.Set("cci", func(call goja.FunctionCall) goja.Value {
		highs := toFloat64Slice(call.Argument(0).Export())
		lows := toFloat64Slice(call.Argument(1).Export())
		closes := toFloat64Slice(call.Argument(2).Export())
		period := toInt(call.Argument(3).Export())
		n := minLen(len(highs), len(lows), len(closes))
		if period <= 0 || n < period {
			return vm.ToValue(math.NaN())
		}
		// Typical prices for last `period` bars
		tps := make([]float64, period)
		off := n - period
		for i := 0; i < period; i++ {
			tps[i] = (highs[off+i] + lows[off+i] + closes[off+i]) / 3.0
		}
		mean := 0.0
		for _, v := range tps {
			mean += v
		}
		mean /= float64(period)
		// Mean deviation
		md := 0.0
		for _, v := range tps {
			md += math.Abs(v - mean)
		}
		md /= float64(period)
		if md == 0 {
			return vm.ToValue(0.0)
		}
		return vm.ToValue((tps[period-1] - mean) / (0.015 * md))
	})

	// ta.williamsR(highs, lows, closes, period) → number
	ta.Set("williamsR", func(call goja.FunctionCall) goja.Value {
		highs := toFloat64Slice(call.Argument(0).Export())
		lows := toFloat64Slice(call.Argument(1).Export())
		closes := toFloat64Slice(call.Argument(2).Export())
		period := toInt(call.Argument(3).Export())
		n := minLen(len(highs), len(lows), len(closes))
		if period <= 0 || n < period {
			return vm.ToValue(math.NaN())
		}
		off := n - period
		hh, ll := highs[off], lows[off]
		for i := 1; i < period; i++ {
			if highs[off+i] > hh {
				hh = highs[off+i]
			}
			if lows[off+i] < ll {
				ll = lows[off+i]
			}
		}
		if hh == ll {
			return vm.ToValue(-50.0)
		}
		return vm.ToValue(-100.0 * (hh - closes[n-1]) / (hh - ll))
	})

	// ta.roc(closes, period) → number
	ta.Set("roc", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) <= period {
			return vm.ToValue(math.NaN())
		}
		prev := vals[len(vals)-1-period]
		if prev == 0 {
			return vm.ToValue(math.NaN())
		}
		return vm.ToValue((vals[len(vals)-1] - prev) / prev * 100.0)
	})

	// -----------------------------------------------------------------------
	// Volatility
	// -----------------------------------------------------------------------

	// ta.atr(highs, lows, closes, period) → number
	ta.Set("atr", func(call goja.FunctionCall) goja.Value {
		highs := toFloat64Slice(call.Argument(0).Export())
		lows := toFloat64Slice(call.Argument(1).Export())
		closes := toFloat64Slice(call.Argument(2).Export())
		period := toInt(call.Argument(3).Export())
		n := minLen(len(highs), len(lows), len(closes))
		if period <= 0 || n < period+1 {
			return vm.ToValue(math.NaN())
		}
		// True ranges starting from index 1
		trs := make([]float64, n-1)
		for i := 1; i < n; i++ {
			hl := highs[i] - lows[i]
			hc := math.Abs(highs[i] - closes[i-1])
			lc := math.Abs(lows[i] - closes[i-1])
			trs[i-1] = math.Max(hl, math.Max(hc, lc))
		}
		if len(trs) < period {
			return vm.ToValue(math.NaN())
		}
		// Initial ATR = SMA of first `period` true ranges
		atr := 0.0
		for i := 0; i < period; i++ {
			atr += trs[i]
		}
		atr /= float64(period)
		// Wilder smoothing
		for i := period; i < len(trs); i++ {
			atr = (atr*float64(period-1) + trs[i]) / float64(period)
		}
		return vm.ToValue(atr)
	})

	// ta.bollingerBands(closes, period, stddev) → { upper, middle, lower }
	ta.Set("bollingerBands", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		mult := toFloat(call.Argument(2).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		window := vals[len(vals)-period:]
		mean := 0.0
		for _, v := range window {
			mean += v
		}
		mean /= float64(period)
		variance := 0.0
		for _, v := range window {
			d := v - mean
			variance += d * d
		}
		sd := math.Sqrt(variance / float64(period))
		result := vm.NewObject()
		result.Set("upper", mean+mult*sd)
		result.Set("middle", mean)
		result.Set("lower", mean-mult*sd)
		return result
	})

	// ta.stddev(values, period) → number
	ta.Set("stddev", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		window := vals[len(vals)-period:]
		mean := 0.0
		for _, v := range window {
			mean += v
		}
		mean /= float64(period)
		variance := 0.0
		for _, v := range window {
			d := v - mean
			variance += d * d
		}
		return vm.ToValue(math.Sqrt(variance / float64(period)))
	})

	// ta.variance(values, period) → number
	ta.Set("variance", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		window := vals[len(vals)-period:]
		mean := 0.0
		for _, v := range window {
			mean += v
		}
		mean /= float64(period)
		variance := 0.0
		for _, v := range window {
			d := v - mean
			variance += d * d
		}
		return vm.ToValue(variance / float64(period))
	})

	// ta.annualizedVol(closes, period) → number
	ta.Set("annualizedVol", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period+1 {
			return vm.ToValue(math.NaN())
		}
		// Daily returns for last `period` days
		rets := make([]float64, period)
		off := len(vals) - period - 1
		for i := 0; i < period; i++ {
			if vals[off+i] == 0 {
				return vm.ToValue(math.NaN())
			}
			rets[i] = (vals[off+i+1] - vals[off+i]) / vals[off+i]
		}
		mean := 0.0
		for _, r := range rets {
			mean += r
		}
		mean /= float64(period)
		variance := 0.0
		for _, r := range rets {
			d := r - mean
			variance += d * d
		}
		sd := math.Sqrt(variance / float64(period))
		return vm.ToValue(sd * math.Sqrt(252))
	})

	// -----------------------------------------------------------------------
	// Trend
	// -----------------------------------------------------------------------

	// ta.adx(highs, lows, closes, period) → number
	ta.Set("adx", func(call goja.FunctionCall) goja.Value {
		highs := toFloat64Slice(call.Argument(0).Export())
		lows := toFloat64Slice(call.Argument(1).Export())
		closes := toFloat64Slice(call.Argument(2).Export())
		period := toInt(call.Argument(3).Export())
		n := minLen(len(highs), len(lows), len(closes))
		if period <= 0 || n < 2*period+1 {
			return vm.ToValue(math.NaN())
		}

		// Compute +DM, -DM, TR series
		pDM := make([]float64, n-1)
		nDM := make([]float64, n-1)
		tr := make([]float64, n-1)
		for i := 1; i < n; i++ {
			upMove := highs[i] - highs[i-1]
			downMove := lows[i-1] - lows[i]
			if upMove > downMove && upMove > 0 {
				pDM[i-1] = upMove
			}
			if downMove > upMove && downMove > 0 {
				nDM[i-1] = downMove
			}
			hl := highs[i] - lows[i]
			hc := math.Abs(highs[i] - closes[i-1])
			lc := math.Abs(lows[i] - closes[i-1])
			tr[i-1] = math.Max(hl, math.Max(hc, lc))
		}

		// Wilder smoothing for +DM, -DM, TR
		smoothPDM := wilderSmooth(pDM, period)
		smoothNDM := wilderSmooth(nDM, period)
		smoothTR := wilderSmooth(tr, period)

		// Compute DX series
		m := minLen(len(smoothPDM), len(smoothNDM), len(smoothTR))
		dx := make([]float64, m)
		for i := 0; i < m; i++ {
			if smoothTR[i] == 0 {
				dx[i] = 0
				continue
			}
			pDI := 100.0 * smoothPDM[i] / smoothTR[i]
			nDI := 100.0 * smoothNDM[i] / smoothTR[i]
			sum := pDI + nDI
			if sum == 0 {
				dx[i] = 0
			} else {
				dx[i] = 100.0 * math.Abs(pDI-nDI) / sum
			}
		}

		if len(dx) < period {
			return vm.ToValue(math.NaN())
		}

		// ADX = Wilder-smoothed DX
		adx := 0.0
		for i := 0; i < period; i++ {
			adx += dx[i]
		}
		adx /= float64(period)
		for i := period; i < len(dx); i++ {
			adx = (adx*float64(period-1) + dx[i]) / float64(period)
		}
		return vm.ToValue(adx)
	})

	// ta.supertrend(highs, lows, closes, period, multiplier) → { value, direction }
	ta.Set("supertrend", func(call goja.FunctionCall) goja.Value {
		highs := toFloat64Slice(call.Argument(0).Export())
		lows := toFloat64Slice(call.Argument(1).Export())
		closes := toFloat64Slice(call.Argument(2).Export())
		period := toInt(call.Argument(3).Export())
		mult := toFloat(call.Argument(4).Export())
		n := minLen(len(highs), len(lows), len(closes))
		if period <= 0 || n < period+1 {
			return vm.ToValue(math.NaN())
		}

		// Compute ATR series using Wilder smoothing
		trs := make([]float64, n-1)
		for i := 1; i < n; i++ {
			hl := highs[i] - lows[i]
			hc := math.Abs(highs[i] - closes[i-1])
			lc := math.Abs(lows[i] - closes[i-1])
			trs[i-1] = math.Max(hl, math.Max(hc, lc))
		}
		if len(trs) < period {
			return vm.ToValue(math.NaN())
		}
		atrVal := 0.0
		for i := 0; i < period; i++ {
			atrVal += trs[i]
		}
		atrVal /= float64(period)

		// Compute supertrend from period onwards
		// direction: 1 = up (bullish), -1 = down (bearish)
		direction := 1.0
		var stVal float64
		upperBand := (highs[period]+lows[period])/2.0 + mult*atrVal
		lowerBand := (highs[period]+lows[period])/2.0 - mult*atrVal
		if closes[period] <= upperBand {
			direction = -1
			stVal = upperBand
		} else {
			direction = 1
			stVal = lowerBand
		}
		_ = stVal

		prevUpper := upperBand
		prevLower := lowerBand

		for i := period + 1; i < n; i++ {
			atrVal = (atrVal*float64(period-1) + trs[i-1]) / float64(period)
			basicUpper := (highs[i]+lows[i])/2.0 + mult*atrVal
			basicLower := (highs[i]+lows[i])/2.0 - mult*atrVal

			if basicUpper < prevUpper || closes[i-1] > prevUpper {
				upperBand = basicUpper
			} else {
				upperBand = prevUpper
			}
			if basicLower > prevLower || closes[i-1] < prevLower {
				lowerBand = basicLower
			} else {
				lowerBand = prevLower
			}

			if direction == 1 {
				if closes[i] < lowerBand {
					direction = -1
					stVal = upperBand
				} else {
					stVal = lowerBand
				}
			} else {
				if closes[i] > upperBand {
					direction = 1
					stVal = lowerBand
				} else {
					stVal = upperBand
				}
			}
			prevUpper = upperBand
			prevLower = lowerBand
		}

		result := vm.NewObject()
		result.Set("value", stVal)
		result.Set("direction", direction)
		return result
	})

	// -----------------------------------------------------------------------
	// Statistics
	// -----------------------------------------------------------------------

	// ta.percentile(values, pct) → number
	ta.Set("percentile", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		pct := toFloat(call.Argument(1).Export())
		if len(vals) == 0 {
			return vm.ToValue(math.NaN())
		}
		sorted := make([]float64, len(vals))
		copy(sorted, vals)
		sort.Float64s(sorted)
		idx := pct / 100.0 * float64(len(sorted)-1)
		lower := int(math.Floor(idx))
		upper := int(math.Ceil(idx))
		if lower == upper || upper >= len(sorted) {
			return vm.ToValue(sorted[lower])
		}
		frac := idx - float64(lower)
		return vm.ToValue(sorted[lower]*(1-frac) + sorted[upper]*frac)
	})

	// ta.percentileRank(values, value) → number (0-100)
	ta.Set("percentileRank", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		val := toFloat(call.Argument(1).Export())
		if len(vals) == 0 {
			return vm.ToValue(math.NaN())
		}
		count := 0
		for _, v := range vals {
			if v <= val {
				count++
			}
		}
		return vm.ToValue(float64(count) / float64(len(vals)) * 100.0)
	})

	// ta.zscore(values, value) → number
	ta.Set("zscore", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		val := toFloat(call.Argument(1).Export())
		if len(vals) == 0 {
			return vm.ToValue(math.NaN())
		}
		mean := 0.0
		for _, v := range vals {
			mean += v
		}
		mean /= float64(len(vals))
		variance := 0.0
		for _, v := range vals {
			d := v - mean
			variance += d * d
		}
		sd := math.Sqrt(variance / float64(len(vals)))
		if sd == 0 {
			return vm.ToValue(0.0)
		}
		return vm.ToValue((val - mean) / sd)
	})

	// ta.correlation(xs, ys) → number
	ta.Set("correlation", func(call goja.FunctionCall) goja.Value {
		xs := toFloat64Slice(call.Argument(0).Export())
		ys := toFloat64Slice(call.Argument(1).Export())
		n := minLen(len(xs), len(ys))
		if n < 2 {
			return vm.ToValue(math.NaN())
		}
		mx, my := 0.0, 0.0
		for i := 0; i < n; i++ {
			mx += xs[i]
			my += ys[i]
		}
		mx /= float64(n)
		my /= float64(n)
		cov, vx, vy := 0.0, 0.0, 0.0
		for i := 0; i < n; i++ {
			dx := xs[i] - mx
			dy := ys[i] - my
			cov += dx * dy
			vx += dx * dx
			vy += dy * dy
		}
		denom := math.Sqrt(vx * vy)
		if denom == 0 {
			return vm.ToValue(0.0)
		}
		return vm.ToValue(cov / denom)
	})

	// ta.linearRegression(xs, ys) → { slope, intercept, r2 }
	ta.Set("linearRegression", func(call goja.FunctionCall) goja.Value {
		xs := toFloat64Slice(call.Argument(0).Export())
		ys := toFloat64Slice(call.Argument(1).Export())
		n := minLen(len(xs), len(ys))
		if n < 2 {
			return vm.ToValue(math.NaN())
		}
		slope, intercept, r2 := linearReg(xs[:n], ys[:n])
		result := vm.NewObject()
		result.Set("slope", slope)
		result.Set("intercept", intercept)
		result.Set("r2", r2)
		return result
	})

	// ta.returns(closes) → number[]
	ta.Set("returns", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		if len(vals) < 2 {
			return vm.ToValue([]float64{})
		}
		rets := make([]float64, len(vals)-1)
		for i := 1; i < len(vals); i++ {
			if vals[i-1] == 0 {
				rets[i-1] = 0
			} else {
				rets[i-1] = (vals[i] - vals[i-1]) / vals[i-1]
			}
		}
		return vm.ToValue(rets)
	})

	// ta.cumReturns(closes) → number[]
	ta.Set("cumReturns", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		if len(vals) < 2 || vals[0] == 0 {
			return vm.ToValue([]float64{})
		}
		cum := make([]float64, len(vals))
		cum[0] = 0
		for i := 1; i < len(vals); i++ {
			cum[i] = (vals[i] - vals[0]) / vals[0]
		}
		return vm.ToValue(cum)
	})

	// ta.drawdown(equityCurve) → number (current drawdown from peak, as positive fraction)
	ta.Set("drawdown", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		if len(vals) == 0 {
			return vm.ToValue(0.0)
		}
		peak := vals[0]
		for _, v := range vals {
			if v > peak {
				peak = v
			}
		}
		if peak == 0 {
			return vm.ToValue(0.0)
		}
		return vm.ToValue((peak - vals[len(vals)-1]) / peak)
	})

	// ta.maxDrawdown(equityCurve) → number (max historical drawdown, positive fraction)
	ta.Set("maxDrawdown", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		if len(vals) == 0 {
			return vm.ToValue(0.0)
		}
		peak := vals[0]
		maxDD := 0.0
		for _, v := range vals {
			if v > peak {
				peak = v
			}
			if peak > 0 {
				dd := (peak - v) / peak
				if dd > maxDD {
					maxDD = dd
				}
			}
		}
		return vm.ToValue(maxDD)
	})

	// ta.sharpe(returns, riskFreeRate) → number
	ta.Set("sharpe", func(call goja.FunctionCall) goja.Value {
		rets := toFloat64Slice(call.Argument(0).Export())
		rf := toFloat(call.Argument(1).Export())
		if len(rets) < 2 {
			return vm.ToValue(math.NaN())
		}
		mean := 0.0
		for _, r := range rets {
			mean += r
		}
		mean /= float64(len(rets))
		variance := 0.0
		for _, r := range rets {
			d := r - mean
			variance += d * d
		}
		sd := math.Sqrt(variance / float64(len(rets)))
		if sd == 0 {
			return vm.ToValue(0.0)
		}
		// Annualize: assume daily returns
		dailyRf := rf / 252.0
		return vm.ToValue((mean - dailyRf) / sd * math.Sqrt(252))
	})

	// ta.sortino(returns, riskFreeRate) → number
	ta.Set("sortino", func(call goja.FunctionCall) goja.Value {
		rets := toFloat64Slice(call.Argument(0).Export())
		rf := toFloat(call.Argument(1).Export())
		if len(rets) < 2 {
			return vm.ToValue(math.NaN())
		}
		dailyRf := rf / 252.0
		mean := 0.0
		for _, r := range rets {
			mean += r
		}
		mean /= float64(len(rets))
		// Downside deviation
		downVar := 0.0
		downCount := 0
		for _, r := range rets {
			excess := r - dailyRf
			if excess < 0 {
				downVar += excess * excess
				downCount++
			}
		}
		if downCount == 0 {
			return vm.ToValue(math.Inf(1))
		}
		downDev := math.Sqrt(downVar / float64(len(rets)))
		if downDev == 0 {
			return vm.ToValue(math.Inf(1))
		}
		return vm.ToValue((mean - dailyRf) / downDev * math.Sqrt(252))
	})

	// -----------------------------------------------------------------------
	// Utility
	// -----------------------------------------------------------------------

	// ta.crossover(fast, slow) → boolean
	ta.Set("crossover", func(call goja.FunctionCall) goja.Value {
		fast := toFloat64Slice(call.Argument(0).Export())
		slow := toFloat64Slice(call.Argument(1).Export())
		n := minLen(len(fast), len(slow))
		if n < 2 {
			return vm.ToValue(false)
		}
		return vm.ToValue(fast[n-2] <= slow[n-2] && fast[n-1] > slow[n-1])
	})

	// ta.crossunder(fast, slow) → boolean
	ta.Set("crossunder", func(call goja.FunctionCall) goja.Value {
		fast := toFloat64Slice(call.Argument(0).Export())
		slow := toFloat64Slice(call.Argument(1).Export())
		n := minLen(len(fast), len(slow))
		if n < 2 {
			return vm.ToValue(false)
		}
		return vm.ToValue(fast[n-2] >= slow[n-2] && fast[n-1] < slow[n-1])
	})

	// ta.highest(values, period) → number
	ta.Set("highest", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		max := vals[len(vals)-period]
		for _, v := range vals[len(vals)-period+1:] {
			if v > max {
				max = v
			}
		}
		return vm.ToValue(max)
	})

	// ta.lowest(values, period) → number
	ta.Set("lowest", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		min := vals[len(vals)-period]
		for _, v := range vals[len(vals)-period+1:] {
			if v < min {
				min = v
			}
		}
		return vm.ToValue(min)
	})

	// ta.sum(values, period) → number
	ta.Set("sum", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) < period {
			return vm.ToValue(math.NaN())
		}
		s := 0.0
		for _, v := range vals[len(vals)-period:] {
			s += v
		}
		return vm.ToValue(s)
	})

	// ta.change(values, period) → number
	ta.Set("change", func(call goja.FunctionCall) goja.Value {
		vals := toFloat64Slice(call.Argument(0).Export())
		period := toInt(call.Argument(1).Export())
		if period <= 0 || len(vals) <= period {
			return vm.ToValue(math.NaN())
		}
		return vm.ToValue(vals[len(vals)-1] - vals[len(vals)-1-period])
	})

	vm.Set("ta", ta)
}

// -----------------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------------

// toFloat64Slice converts a JS array export to []float64.
func toFloat64Slice(v interface{}) []float64 {
	switch arr := v.(type) {
	case []interface{}:
		out := make([]float64, 0, len(arr))
		for _, item := range arr {
			out = append(out, toFloat(item))
		}
		return out
	case []float64:
		return arr
	}
	return nil
}

// toFloat converts a JS number export to float64.
func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}

// toInt converts a JS number export to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// minLen returns the minimum of the given lengths.
func minLen(lengths ...int) int {
	if len(lengths) == 0 {
		return 0
	}
	m := lengths[0]
	for _, l := range lengths[1:] {
		if l < m {
			m = l
		}
	}
	return m
}

// calcEMASeries returns the full EMA series for the given values and period.
func calcEMASeries(vals []float64, period int) []float64 {
	if period <= 0 || len(vals) < period {
		return nil
	}
	k := 2.0 / float64(period+1)
	// Seed with SMA.
	ema := 0.0
	for i := 0; i < period; i++ {
		ema += vals[i]
	}
	ema /= float64(period)
	out := make([]float64, 0, len(vals)-period+1)
	out = append(out, ema)
	for i := period; i < len(vals); i++ {
		ema = vals[i]*k + ema*(1-k)
		out = append(out, ema)
	}
	return out
}

// wilderSmooth applies Wilder's smoothing to a series.
func wilderSmooth(vals []float64, period int) []float64 {
	if len(vals) < period {
		return nil
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += vals[i]
	}
	out := make([]float64, 0, len(vals)-period+1)
	s := sum / float64(period)
	out = append(out, s)
	for i := period; i < len(vals); i++ {
		s = (s*float64(period-1) + vals[i]) / float64(period)
		out = append(out, s)
	}
	return out
}

// linearReg computes simple linear regression returning slope, intercept, r².
func linearReg(xs, ys []float64) (slope, intercept, r2 float64) {
	n := float64(len(xs))
	mx, my := 0.0, 0.0
	for i := range xs {
		mx += xs[i]
		my += ys[i]
	}
	mx /= n
	my /= n
	ssXX, ssXY, ssYY := 0.0, 0.0, 0.0
	for i := range xs {
		dx := xs[i] - mx
		dy := ys[i] - my
		ssXX += dx * dx
		ssXY += dx * dy
		ssYY += dy * dy
	}
	if ssXX == 0 {
		return 0, my, 0
	}
	slope = ssXY / ssXX
	intercept = my - slope*mx
	if ssYY == 0 {
		r2 = 1
	} else {
		r2 = (ssXY * ssXY) / (ssXX * ssYY)
	}
	return
}
