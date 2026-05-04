package calc

import (
	"fmt"
	"math"
	"time"
)

type PricePoint struct {
	Month  string
	Median int
}

type AptResult struct {
	Name  string
	Y1    PricePoint
	Y3    PricePoint
	Y1Pct float64
	Y3Pct float64
	Y1OK  bool
	Y3OK  bool
}

type Gangnam3Result struct {
	Apts     []AptResult
	AvgY1Pct float64
	AvgY3Pct float64
}

// AnalyzeWithCurrent computes change rates and returns current price.
// now is injected for testability; pass time.Now() in production.
func AnalyzeWithCurrent(aptName string, prices map[string]int, now time.Time) (PricePoint, AptResult, error) {
	current, ok := findCurrent(prices, now)
	if !ok {
		return PricePoint{}, AptResult{}, fmt.Errorf("%s: 현재가 산출 불가", aptName)
	}

	r := AptResult{Name: aptName}
	r.Y1, r.Y1OK = findBase(prices, now, 12)
	r.Y3, r.Y3OK = findBase(prices, now, 36)

	if r.Y1OK {
		r.Y1Pct = changePct(current.Median, r.Y1.Median)
	}
	if r.Y3OK {
		r.Y3Pct = changePct(current.Median, r.Y3.Median)
	}

	return current, r, nil
}

// Gangnam3Avg computes the average Y1/Y3 change rate across multiple apartments.
// Apartments with unavailable data are excluded from the average.
func Gangnam3Avg(apts []AptResult) Gangnam3Result {
	var sumY1, sumY3 float64
	var cntY1, cntY3 int

	for _, a := range apts {
		if a.Y1OK {
			sumY1 += a.Y1Pct
			cntY1++
		}
		if a.Y3OK {
			sumY3 += a.Y3Pct
			cntY3++
		}
	}

	r := Gangnam3Result{Apts: apts}
	if cntY1 > 0 {
		r.AvgY1Pct = sumY1 / float64(cntY1)
	}
	if cntY3 > 0 {
		r.AvgY3Pct = sumY3 / float64(cntY3)
	}
	return r
}

// FollowRate computes the follow rate of target vs benchmark.
// 100% = same movement as gangnam3, >100% = outperformed.
func FollowRate(targetPct, avgPct float64) float64 {
	if avgPct == 0 {
		return 0
	}
	return targetPct / avgPct * 100
}

// findCurrent finds the most recent price within the last 9 months.
func findCurrent(prices map[string]int, now time.Time) (PricePoint, bool) {
	for i := range 9 {
		month := now.AddDate(0, -i, 0).Format("200601")
		if median, ok := prices[month]; ok {
			return PricePoint{Month: month, Median: median}, true
		}
	}
	return PricePoint{}, false
}

// findBase finds the closest price to targetMonths ago, expanding tolerance ±3 → ±6 → ±9.
func findBase(prices map[string]int, now time.Time, targetMonths int) (PricePoint, bool) {
	target := now.AddDate(0, -targetMonths, 0)

	for _, tolerance := range []int{3, 6, 9} {
		best := PricePoint{}
		bestDiff := math.MaxInt32

		for offset := -tolerance; offset <= tolerance; offset++ {
			month := target.AddDate(0, offset, 0).Format("200601")
			if median, ok := prices[month]; ok {
				diff := abs(offset)
				if diff < bestDiff {
					bestDiff = diff
					best = PricePoint{Month: month, Median: median}
				}
			}
		}

		if bestDiff < math.MaxInt32 {
			return best, true
		}
	}

	return PricePoint{}, false
}

func changePct(current, base int) float64 {
	if base == 0 {
		return 0
	}
	return float64(current-base) / float64(base) * 100
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
