package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"time"

	"nezip/internal/api"
	"nezip/internal/cache"
	"nezip/internal/calc"
	"nezip/internal/report"
	_ "modernc.org/sqlite"
)

type g3Apt struct {
	Name       string
	LawdCD     string
	TargetArea float64
}

type AptPrices struct {
	Name   string
	LawdCD string
	Area   float64
	Prices map[string]int // month → median
}

var g3Apts84 = []g3Apt{
	{"잠실주공5단지", "11710", 84},
	{"래미안퍼스티지", "11650", 84},
	{"반포자이", "11650", 84},
	{"아크로리버파크", "11650", 84},
	{"래미안대치팰리스", "11680", 84},
	{"헬리오시티", "11710", 84},
	{"반포리체", "11650", 84},
	{"서초그랑자이", "11650", 84},
	{"도곡렉슬", "11680", 84},
	{"래미안블레스티지", "11680", 84},
	{"개포래미안포레스트", "11680", 84},
	{"디에이치아너힐즈", "11680", 84},
	{"파크리오", "11710", 84},
	{"리센츠", "11710", 84},
	{"엘스", "11710", 84},
	{"트리지움", "11710", 84},
}

var g3Apts59 = []g3Apt{
	{"아크로리버파크", "11650", 59},
	{"래미안퍼스티지", "11650", 59},
	{"반포자이", "11650", 59},
	{"헬리오시티", "11710", 59},
	{"반포리체", "11650", 59},
	{"서초그랑자이", "11650", 59},
	{"도곡렉슬", "11680", 59},
	{"래미안블레스티지", "11680", 59},
	{"개포래미안포레스트", "11680", 59},
	{"디에이치아너힐즈", "11680", 59},
	{"파크리오", "11710", 59},
	{"리센츠", "11710", 59},
	{"엘스", "11710", 59},
	{"트리지움", "11710", 59},
}

func main() {
	ctx := context.Background()

	if len(os.Args) > 1 && os.Args[1] == "cache" {
		runCache(ctx)
		return
	}

	var (
		aptName string
		area    float64
		lawdCD  string
		human   bool
	)
	flag.StringVar(&aptName, "apt", "", "아파트명")
	flag.Float64Var(&area, "area", 0, "전용면적(㎡)")
	flag.StringVar(&lawdCD, "lawd", "", "법정동코드")
	flag.BoolVar(&human, "human", false, "사람이 읽는 출력")
	flag.Parse()

	if aptName == "" || area == 0 || lawdCD == "" {
		fmt.Fprintln(os.Stderr, "usage: nezip --apt <아파트명> --area <면적> --lawd <LAWD_CD> [--human]")
		os.Exit(1)
	}

	if err := runAnalyze(ctx, aptName, area, lawdCD, human); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runAnalyze(ctx context.Context, aptName string, area float64, lawdCD string, human bool) error {
	store, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	apiKey := os.Getenv("MOLIT_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("MOLIT_API_KEY 환경변수가 설정되지 않았습니다")
	}
	client := api.NewClient(apiKey)

	areaInt := int64(math.Round(area))
	allMonths := api.Last36Months()
	g3list := selectG3List(area, aptName, lawdCD)

	var warnings []string

	cached, _ := store.CachedMonthsSet(ctx, aptName, lawdCD, areaInt)
	cacheUsed := len(cached) > 0

	target, targetWarnings, err := fetchAndCacheApt(ctx, store, client, aptName, lawdCD, area, areaInt, allMonths)
	if err != nil {
		return fmt.Errorf("대상 아파트 조회 실패: %w", err)
	}
	warnings = append(warnings, targetWarnings...)

	g3list2, g3Warnings, err := fetchAndCacheG3(ctx, store, client, g3list, allMonths)
	if err != nil {
		return fmt.Errorf("강남3구 조회 실패: %w", err)
	}
	warnings = append(warnings, g3Warnings...)

	now := time.Now()
	current, aptResult, err := calc.AnalyzeWithCurrent(target.Name, target.Prices, now)
	if err != nil {
		return err
	}

	g3AptResults := make([]calc.AptResult, 0, len(g3list2))
	for _, ap := range g3list2 {
		if len(ap.Prices) == 0 {
			warnings = append(warnings, ap.Name+": 데이터 없음, 강남3구 평균에서 제외")
			continue
		}
		_, r, err := calc.AnalyzeWithCurrent(ap.Name, ap.Prices, now)
		if err != nil {
			warnings = append(warnings, ap.Name+": "+err.Error())
			continue
		}
		g3AptResults = append(g3AptResults, r)
	}

	g3Result := calc.Gangnam3Avg(g3AptResults)
	followY1 := calc.FollowRate(aptResult.Y1Pct, g3Result.AvgY1Pct)
	followY3 := calc.FollowRate(aptResult.Y3Pct, g3Result.AvgY3Pct)

	if human {
		report.WriteHuman(os.Stdout, aptName, area, current, aptResult, g3Result, followY1, followY3, warnings)
		return nil
	}

	breakdown := make([]report.AptBreakdown, len(g3AptResults))
	for i, r := range g3AptResults {
		breakdown[i] = report.AptBreakdown{
			Name:  r.Name,
			Y1Pct: r.Y1Pct,
			Y3Pct: r.Y3Pct,
			Y1OK:  r.Y1OK,
			Y3OK:  r.Y3OK,
		}
	}

	out := report.Output{
		Target: report.TargetJSON{
			Name:         aptName,
			Area:         area,
			Current:      current.Median,
			CurrentMonth: current.Month,
		},
		Gangnam3: report.Gangnam3JSON{
			AvgY1Pct:  g3Result.AvgY1Pct,
			AvgY3Pct:  g3Result.AvgY3Pct,
			Breakdown: breakdown,
		},
		FollowRateY1: followY1,
		FollowRateY3: followY3,
		CacheUsed:    cacheUsed,
		Warnings:     warnings,
	}
	if aptResult.Y1OK {
		out.Target.Y1 = aptResult.Y1.Median
		out.Target.Y1Month = aptResult.Y1.Month
		out.Target.Y1Pct = aptResult.Y1Pct
	}
	if aptResult.Y3OK {
		out.Target.Y3 = aptResult.Y3.Median
		out.Target.Y3Month = aptResult.Y3.Month
		out.Target.Y3Pct = aptResult.Y3Pct
	}

	return report.WriteJSON(os.Stdout, out)
}

func fetchAndCacheApt(ctx context.Context, store *cache.Store, client *api.Client, aptName, lawdCD string, area float64, areaInt int64, allMonths []string) (AptPrices, []string, error) {
	cached, err := store.CachedMonthsSet(ctx, aptName, lawdCD, areaInt)
	if err != nil {
		return AptPrices{}, nil, err
	}

	uncached := make([]string, 0, len(allMonths))
	for _, m := range allMonths {
		if !cached[m] {
			uncached = append(uncached, m)
		}
	}

	var warnings []string
	var dataMonths int

	if len(uncached) > 0 {
		fetched, fetchErr := client.FetchMonths(ctx, lawdCD, uncached)
		if fetchErr != nil {
			warnings = append(warnings, fmt.Sprintf("일부 월 조회 실패: %v", fetchErr))
		}
		for month, trades := range fetched {
			filtered := api.FilterByApt(trades, aptName, area, 5.0)
			if len(filtered) == 0 {
				continue
			}
			dataMonths++
			median := api.Median(filtered)
			if err := store.SaveTrade(ctx, aptName, lawdCD, month, area, areaInt, median, len(filtered)); err != nil {
				return AptPrices{}, nil, err
			}
		}

		if dataMonths < 3 {
			warnings = append(warnings, "면적 필터 ±10㎡ 확장 적용 (±5㎡ 기준 데이터 부족)")
			for month, trades := range fetched {
				filtered := api.FilterByApt(trades, aptName, area, 10.0)
				if len(filtered) == 0 {
					continue
				}
				median := api.Median(filtered)
				store.SaveTrade(ctx, aptName, lawdCD, month, area, areaInt, median, len(filtered))
			}
		}
	}

	prices, err := store.LoadPrices(ctx, aptName, lawdCD, areaInt)
	if err != nil {
		return AptPrices{}, warnings, err
	}
	if len(prices) < 3 {
		warnings = append(warnings, "데이터 신뢰도 낮음 (36개월 중 3개월 미만 거래)")
	}

	return AptPrices{Name: aptName, LawdCD: lawdCD, Area: area, Prices: prices}, warnings, nil
}

func fetchAndCacheG3(ctx context.Context, store *cache.Store, client *api.Client, g3list []g3Apt, allMonths []string) ([]AptPrices, []string, error) {
	byDistrict := make(map[string][]g3Apt)
	for _, a := range g3list {
		byDistrict[a.LawdCD] = append(byDistrict[a.LawdCD], a)
	}

	type distResult struct {
		lawdCD  string
		fetched map[string][]api.Trade
		err     error
	}

	resCh := make(chan distResult, len(byDistrict))
	var wg sync.WaitGroup

	for lawdCD, apts := range byDistrict {
		lawdCD := lawdCD
		apts := apts
		wg.Add(1)
		go func() {
			defer wg.Done()

			uncachedSet := make(map[string]bool)
			for _, apt := range apts {
				areaInt := int64(math.Round(apt.TargetArea))
				cached, _ := store.CachedMonthsSet(ctx, apt.Name, lawdCD, areaInt)
				for _, m := range allMonths {
					if !cached[m] {
						uncachedSet[m] = true
					}
				}
			}

			uncached := make([]string, 0, len(uncachedSet))
			for m := range uncachedSet {
				uncached = append(uncached, m)
			}

			var (
				fetched  map[string][]api.Trade
				fetchErr error
			)
			if len(uncached) > 0 {
				fetched, fetchErr = client.FetchMonths(ctx, lawdCD, uncached)
			}
			resCh <- distResult{lawdCD: lawdCD, fetched: fetched, err: fetchErr}
		}()
	}

	wg.Wait()
	close(resCh)

	// preserve g3list order
	pricesByApt := make(map[string]map[string]int, len(g3list))
	var g3Warnings []string
	for df := range resCh {
		if df.err != nil {
			g3Warnings = append(g3Warnings, fmt.Sprintf("강남3구(%s) 일부 월 조회 실패: %v", df.lawdCD, df.err))
		}
		for _, apt := range byDistrict[df.lawdCD] {
			areaInt := int64(math.Round(apt.TargetArea))
			for month, trades := range df.fetched {
				filtered := api.FilterByApt(trades, apt.Name, apt.TargetArea, 5.0)
				if len(filtered) == 0 {
					continue
				}
				if err := store.SaveTrade(ctx, apt.Name, df.lawdCD, month, apt.TargetArea, areaInt, api.Median(filtered), len(filtered)); err != nil {
					return nil, nil, fmt.Errorf("save trade %s %s: %w", apt.Name, month, err)
				}
			}
			prices, err := store.LoadPrices(ctx, apt.Name, df.lawdCD, areaInt)
			if err != nil {
				return nil, nil, fmt.Errorf("load prices %s: %w", apt.Name, err)
			}
			pricesByApt[apt.Name+"/"+apt.LawdCD] = prices
		}
	}

	result := make([]AptPrices, 0, len(g3list))
	for _, apt := range g3list {
		result = append(result, AptPrices{
			Name:   apt.Name,
			LawdCD: apt.LawdCD,
			Area:   apt.TargetArea,
			Prices: pricesByApt[apt.Name+"/"+apt.LawdCD],
		})
	}
	return result, g3Warnings, nil
}

func selectG3List(area float64, aptName, lawdCD string) []g3Apt {
	var base []g3Apt
	if area >= 54 && area <= 64 {
		base = g3Apts59
	} else {
		base = g3Apts84
	}

	result := make([]g3Apt, 0, len(base))
	for _, a := range base {
		if a.Name == aptName && a.LawdCD == lawdCD {
			continue
		}
		result = append(result, a)
	}
	return result
}

func runCache(ctx context.Context) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: nezip cache <refresh|status>")
		os.Exit(1)
	}
	store, err := openStore(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer store.Close()

	switch os.Args[2] {
	case "status":
		fmt.Println("cache status: not implemented yet")
	case "refresh":
		fmt.Println("cache refresh: not implemented yet")
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func openStore(ctx context.Context) (*cache.Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(home, ".nezip.db")
	return cache.Open(ctx, dbPath)
}
