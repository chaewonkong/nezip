package report

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"nezip/internal/calc"
	"strings"
)

type Output struct {
	Target       TargetJSON    `json:"target"`
	Gangnam3     Gangnam3JSON  `json:"gangnam3"`
	FollowRateY1 float64       `json:"follow_rate_y1"`
	FollowRateY3 float64       `json:"follow_rate_y3"`
	CacheUsed    bool          `json:"cache_used"`
	Warnings     []string      `json:"warnings"`
}

type TargetJSON struct {
	Name         string  `json:"name"`
	Area         float64 `json:"area"`
	Current      int     `json:"current"`
	CurrentMonth string  `json:"current_month"`
	Y1           int     `json:"y1"`
	Y1Month      string  `json:"y1_month"`
	Y1Pct        float64 `json:"y1_pct"`
	Y3           int     `json:"y3"`
	Y3Month      string  `json:"y3_month"`
	Y3Pct        float64 `json:"y3_pct"`
}

type Gangnam3JSON struct {
	AvgY1Pct  float64         `json:"avg_y1_pct"`
	AvgY3Pct  float64         `json:"avg_y3_pct"`
	Breakdown []AptBreakdown  `json:"breakdown"`
}

type AptBreakdown struct {
	Name  string  `json:"name"`
	Y1Pct float64 `json:"y1_pct"`
	Y3Pct float64 `json:"y3_pct"`
	Y1OK  bool    `json:"y1_ok"`
	Y3OK  bool    `json:"y3_ok"`
}

func WriteJSON(w io.Writer, out Output) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func WriteHuman(w io.Writer, aptName string, area float64, current calc.PricePoint, apt calc.AptResult, g3 calc.Gangnam3Result, followY1, followY3 float64, warnings []string) {
	sep := strings.Repeat("━", 45)

	fmt.Fprintf(w, "%s\n", sep)
	fmt.Fprintf(w, "🏠 아파트 실거래가 변동율 분석 리포트\n")
	fmt.Fprintf(w, "%s\n\n", sep)

	fmt.Fprintf(w, "[대상 아파트]\n")
	fmt.Fprintf(w, "  아파트명: %s\n", aptName)
	fmt.Fprintf(w, "  전용면적: %.2f㎡ (약 %.1f평)\n", area, area/3.3058)

	fmt.Fprintf(w, "\n[실거래가 현황]\n")
	fmt.Fprintf(w, "  현재가  (%s): %s만원\n", current.Month, fmtWon(current.Median))

	if apt.Y1OK {
		fmt.Fprintf(w, "  1년 전  (%s): %s만원\n", apt.Y1.Month, fmtWon(apt.Y1.Median))
	} else {
		fmt.Fprintf(w, "  1년 전: 데이터 없음\n")
	}
	if apt.Y3OK {
		fmt.Fprintf(w, "  3년 전  (%s): %s만원\n", apt.Y3.Month, fmtWon(apt.Y3.Median))
	} else {
		fmt.Fprintf(w, "  3년 전: 데이터 없음\n")
	}

	fmt.Fprintf(w, "\n[변동율]\n")
	if apt.Y1OK {
		fmt.Fprintf(w, "  최근 1년: %+.1f%%\n", apt.Y1Pct)
	} else {
		fmt.Fprintf(w, "  최근 1년: 계산 불가\n")
	}
	if apt.Y3OK {
		fmt.Fprintf(w, "  최근 3년: %+.1f%%\n", apt.Y3Pct)
	} else {
		fmt.Fprintf(w, "  최근 3년: 계산 불가\n")
	}

	fmt.Fprintf(w, "\n%s\n\n", sep)
	fmt.Fprintf(w, "[강남3구 기준 아파트 변동율]\n")
	fmt.Fprintf(w, "  %-20s  %7s  %7s\n", "아파트명", "1년", "3년")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 38))

	for _, a := range g3.Apts {
		y1s := "N/A"
		y3s := "N/A"
		if a.Y1OK {
			y1s = fmt.Sprintf("%+.1f%%", a.Y1Pct)
		}
		if a.Y3OK {
			y3s = fmt.Sprintf("%+.1f%%", a.Y3Pct)
		}
		fmt.Fprintf(w, "  %-20s  %7s  %7s\n", a.Name, y1s, y3s)
	}

	fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 38))
	fmt.Fprintf(w, "  %-20s  %+6.1f%%  %+6.1f%%\n", "강남3구 평균", g3.AvgY1Pct, g3.AvgY3Pct)

	fmt.Fprintf(w, "\n%s\n\n", sep)
	fmt.Fprintf(w, "[강남3구 대비 추종율]\n")

	if apt.Y1OK {
		fmt.Fprintf(w, "  1년 추종율: %.1f%% (%s)\n", followY1, evalFollowRate(followY1))
	} else {
		fmt.Fprintf(w, "  1년 추종율: 계산 불가\n")
	}
	if apt.Y3OK {
		fmt.Fprintf(w, "  3년 추종율: %.1f%% (%s)\n", followY3, evalFollowRate(followY3))
	} else {
		fmt.Fprintf(w, "  3년 추종율: 계산 불가\n")
	}

	if len(warnings) > 0 {
		fmt.Fprintf(w, "\n[주의]\n")
		for _, w2 := range warnings {
			fmt.Fprintf(w, "  ※ %s\n", w2)
		}
	}

	fmt.Fprintf(w, "\n%s\n", sep)
}

func evalFollowRate(r float64) string {
	switch {
	case r < 0:
		return "강남3구와 반대 방향 (위험)"
	case r < 50:
		return "강남3구 대비 크게 하회 (주의)"
	case r < 70:
		return "강남3구 상승의 절반 수준 (보통)"
	case r < 90:
		return "강남3구를 어느 정도 추종 (양호)"
	default:
		return "강남3구 수준으로 추종 (우수)"
	}
}

func fmtWon(amount int) string {
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// Round to 1 decimal
func round1(f float64) float64 {
	return math.Round(f*10) / 10
}
