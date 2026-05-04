package calc

import (
	"math"
	"testing"
	"time"
)

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// fixed time for all tests: 2026-05-01
var now = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

// month returns the YYYYMM string for n months ago from the fixed now.
func month(n int) string {
	return now.AddDate(0, -n, 0).Format("200601")
}

// --- FollowRate ---

func TestFollowRate(t *testing.T) {
	tests := []struct {
		name      string
		targetPct float64
		avgPct    float64
		want      float64
	}{
		{"정확히 일치", 10.0, 10.0, 100.0},
		{"초과 추종", 15.0, 10.0, 150.0},
		{"하회", 5.0, 10.0, 50.0},
		{"반대 방향", -5.0, 10.0, -50.0},
		{"기준 0 (zero div)", 10.0, 0.0, 0.0},
		{"둘 다 하락", -5.0, -10.0, 50.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FollowRate(tt.targetPct, tt.avgPct)
			if got != tt.want {
				t.Errorf("FollowRate(%v, %v) = %v, want %v", tt.targetPct, tt.avgPct, got, tt.want)
			}
		})
	}
}

// --- Gangnam3Avg ---

func TestGangnam3Avg(t *testing.T) {
	t.Run("전체 데이터 평균", func(t *testing.T) {
		apts := []AptResult{
			{Y1OK: true, Y1Pct: 10, Y3OK: true, Y3Pct: 30},
			{Y1OK: true, Y1Pct: 20, Y3OK: true, Y3Pct: 50},
		}
		r := Gangnam3Avg(apts)
		if r.AvgY1Pct != 15 {
			t.Errorf("AvgY1Pct = %v, want 15", r.AvgY1Pct)
		}
		if r.AvgY3Pct != 40 {
			t.Errorf("AvgY3Pct = %v, want 40", r.AvgY3Pct)
		}
	})

	t.Run("일부 데이터 없으면 제외", func(t *testing.T) {
		apts := []AptResult{
			{Y1OK: true, Y1Pct: 10, Y3OK: false},
			{Y1OK: true, Y1Pct: 20, Y3OK: true, Y3Pct: 40},
		}
		r := Gangnam3Avg(apts)
		if r.AvgY1Pct != 15 {
			t.Errorf("AvgY1Pct = %v, want 15", r.AvgY1Pct)
		}
		if r.AvgY3Pct != 40 {
			t.Errorf("AvgY3Pct = %v, want 40 (1개만 유효)", r.AvgY3Pct)
		}
	})

	t.Run("빈 슬라이스", func(t *testing.T) {
		r := Gangnam3Avg(nil)
		if r.AvgY1Pct != 0 || r.AvgY3Pct != 0 {
			t.Error("빈 입력은 0이어야 함")
		}
	})
}

// --- changePct ---

func TestChangePct(t *testing.T) {
	tests := []struct {
		current, base int
		want          float64
	}{
		{110, 100, 10.0},
		{90, 100, -10.0},
		{100, 100, 0.0},
		{100, 0, 0.0},
	}
	for _, tt := range tests {
		got := changePct(tt.current, tt.base)
		if got != tt.want {
			t.Errorf("changePct(%d, %d) = %v, want %v", tt.current, tt.base, got, tt.want)
		}
	}
}

// --- findCurrent ---

func TestFindCurrent(t *testing.T) {
	t.Run("이번달 데이터", func(t *testing.T) {
		prices := map[string]int{month(0): 200000}
		p, ok := findCurrent(prices, now)
		if !ok || p.Median != 200000 || p.Month != month(0) {
			t.Errorf("unexpected: ok=%v p=%v", ok, p)
		}
	})

	t.Run("3개월 전 데이터", func(t *testing.T) {
		prices := map[string]int{month(3): 180000}
		p, ok := findCurrent(prices, now)
		if !ok || p.Month != month(3) {
			t.Errorf("unexpected: ok=%v p=%v", ok, p)
		}
	})

	t.Run("8개월 전 데이터", func(t *testing.T) {
		prices := map[string]int{month(8): 170000}
		p, ok := findCurrent(prices, now)
		if !ok || p.Month != month(8) {
			t.Errorf("unexpected: ok=%v p=%v", ok, p)
		}
	})

	t.Run("9개월 초과는 불가", func(t *testing.T) {
		prices := map[string]int{month(10): 150000}
		_, ok := findCurrent(prices, now)
		if ok {
			t.Error("9개월 초과는 현재가 산출 불가여야 함")
		}
	})

	t.Run("여러 달 중 가장 최근 선택", func(t *testing.T) {
		prices := map[string]int{
			month(5): 160000,
			month(2): 200000,
		}
		p, ok := findCurrent(prices, now)
		if !ok || p.Month != month(2) {
			t.Errorf("가장 최근 달을 선택해야 함, got %v", p.Month)
		}
	})
}

// --- findBase ---

func TestFindBase(t *testing.T) {
	t.Run("정확히 12개월 전", func(t *testing.T) {
		prices := map[string]int{month(12): 150000}
		p, ok := findBase(prices, now, 12)
		if !ok || p.Month != month(12) {
			t.Errorf("unexpected: ok=%v p=%v", ok, p)
		}
	})

	t.Run("±3개월 fallback (+2)", func(t *testing.T) {
		prices := map[string]int{month(14): 140000}
		p, ok := findBase(prices, now, 12)
		if !ok || p.Month != month(14) {
			t.Errorf("±3 fallback 실패: ok=%v p=%v", ok, p)
		}
	})

	t.Run("±6개월 fallback (+5)", func(t *testing.T) {
		prices := map[string]int{month(17): 130000}
		p, ok := findBase(prices, now, 12)
		if !ok || p.Month != month(17) {
			t.Errorf("±6 fallback 실패: ok=%v p=%v", ok, p)
		}
	})

	t.Run("±9개월 fallback (+8)", func(t *testing.T) {
		prices := map[string]int{month(20): 120000}
		p, ok := findBase(prices, now, 12)
		if !ok || p.Month != month(20) {
			t.Errorf("±9 fallback 실패: ok=%v p=%v", ok, p)
		}
	})

	t.Run("범위 초과 시 불가", func(t *testing.T) {
		prices := map[string]int{month(22): 110000}
		_, ok := findBase(prices, now, 12)
		if ok {
			t.Error("±9 초과는 불가여야 함")
		}
	})

	t.Run("두 후보 중 가까운 것 선택", func(t *testing.T) {
		// month(11) = -1 offset, month(13) = +1 offset → 둘 다 거리 1
		prices := map[string]int{
			month(11): 140000,
			month(13): 160000,
		}
		p, ok := findBase(prices, now, 12)
		if !ok {
			t.Fatal("데이터 있음에도 불가")
		}
		if p.Month != month(11) && p.Month != month(13) {
			t.Errorf("유효한 달이 아님: %v", p.Month)
		}
	})
}

// --- AnalyzeWithCurrent ---

func TestAnalyzeWithCurrent(t *testing.T) {
	t.Run("정상 케이스", func(t *testing.T) {
		prices := map[string]int{
			month(0):  200000,
			month(12): 160000,
			month(36): 120000,
		}
		cur, r, err := AnalyzeWithCurrent("테스트", prices, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cur.Median != 200000 {
			t.Errorf("current = %d, want 200000", cur.Median)
		}
		if !r.Y1OK || !r.Y3OK {
			t.Error("Y1OK/Y3OK should be true")
		}
		wantY1 := (200000.0 - 160000.0) / 160000.0 * 100
		if r.Y1Pct != wantY1 {
			t.Errorf("Y1Pct = %v, want %v", r.Y1Pct, wantY1)
		}
		wantY3 := (200000.0 - 120000.0) / 120000.0 * 100
		if !approxEq(r.Y3Pct, wantY3) {
			t.Errorf("Y3Pct = %v, want %v", r.Y3Pct, wantY3)
		}
	})

	t.Run("현재가 없음 → 에러", func(t *testing.T) {
		prices := map[string]int{month(15): 100000}
		_, _, err := AnalyzeWithCurrent("테스트", prices, now)
		if err == nil {
			t.Error("현재가 없으면 에러여야 함")
		}
	})

	t.Run("Y1 없음 → Y1OK false", func(t *testing.T) {
		prices := map[string]int{
			month(0):  200000,
			month(36): 120000,
		}
		_, r, err := AnalyzeWithCurrent("테스트", prices, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Y1OK {
			t.Error("Y1OK should be false")
		}
		if !r.Y3OK {
			t.Error("Y3OK should be true")
		}
	})

	t.Run("빈 데이터 → 에러", func(t *testing.T) {
		_, _, err := AnalyzeWithCurrent("테스트", map[string]int{}, now)
		if err == nil {
			t.Error("빈 데이터는 에러여야 함")
		}
	})
}
