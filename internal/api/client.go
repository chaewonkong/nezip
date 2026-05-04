package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://apis.data.go.kr/1613000/RTMSDataSvcAptTrade/getRTMSDataSvcAptTrade"

type Client struct {
	APIKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type Trade struct {
	AptName string
	Area    float64
	Amount  int    // 만원
	Month   string // YYYYMM
}

// FetchMonths fetches all trades in lawdCD for the given months in parallel.
// Returns map[month][]Trade (unfiltered, all apartments in the district).
// Partial results are returned even when some months fail.
func (c *Client) FetchMonths(ctx context.Context, lawdCD string, months []string) (map[string][]Trade, error) {
	var (
		mu     sync.Mutex
		result = make(map[string][]Trade, len(months))
		errs   []error
		wg     sync.WaitGroup
		sem    = make(chan struct{}, 1)
	)

	for _, month := range months {
		month := month
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			trades, err := c.fetchOne(ctx, lawdCD, month)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("month %s: %w", month, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			result[month] = trades
			mu.Unlock()
		}()
	}

	wg.Wait()
	return result, errors.Join(errs...)
}

func (c *Client) fetchOne(ctx context.Context, lawdCD, month string) ([]Trade, error) {
	params := url.Values{}
	params.Set("LAWD_CD", lawdCD)
	params.Set("DEAL_YMD", month)
	params.Set("serviceKey", c.APIKey)
	params.Set("numOfRows", "1000")
	params.Set("_type", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var raw struct {
		Response struct {
			Body struct {
				Items json.RawMessage `json:"items"`
			} `json:"body"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		preview := string(body)
		if len(preview) > 80 {
			preview = preview[:80]
		}
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, preview)
	}

	rawItems := raw.Response.Body.Items
	if len(rawItems) == 0 || string(rawItems) == "null" || string(rawItems) == `""` {
		return nil, nil
	}

	var itemsObj struct {
		Item json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(rawItems, &itemsObj); err != nil {
		// API가 items를 string으로 반환하는 경우 (데이터 없음)
		return nil, nil
	}

	item := itemsObj.Item
	if len(item) == 0 || string(item) == "null" || string(item) == `""` {
		return nil, nil
	}

	type rawTrade struct {
		AptNm      interface{} `json:"aptNm"`
		ExcluUseAr interface{} `json:"excluUseAr"`
		DealAmount string      `json:"dealAmount"`
		DealYear   interface{} `json:"dealYear"`
		DealMonth  interface{} `json:"dealMonth"`
	}

	var items []rawTrade
	if item[0] == '[' {
		if err := json.Unmarshal(item, &items); err != nil {
			return nil, fmt.Errorf("unmarshal items: %w", err)
		}
	} else {
		var single rawTrade
		if err := json.Unmarshal(item, &single); err != nil {
			return nil, fmt.Errorf("unmarshal item: %w", err)
		}
		items = []rawTrade{single}
	}

	trades := make([]Trade, 0, len(items))
	for _, it := range items {
		area, err := toFloat(it.ExcluUseAr)
		if err != nil {
			continue
		}
		amount, err := parseDealAmount(it.DealAmount)
		if err != nil {
			continue
		}
		year := toInt(it.DealYear)
		mon := toInt(it.DealMonth)

		trades = append(trades, Trade{
			AptName: strings.TrimSpace(fmt.Sprintf("%v", it.AptNm)),
			Area:    area,
			Amount:  amount,
			Month:   fmt.Sprintf("%04d%02d", year, mon),
		})
	}
	return trades, nil
}

// FilterByApt filters trades by apt name (contains) and area within tolerance.
func FilterByApt(trades []Trade, aptName string, targetArea, tolerance float64) []Trade {
	var result []Trade
	for _, t := range trades {
		if strings.Contains(t.AptName, aptName) && math.Abs(t.Area-targetArea) <= tolerance {
			result = append(result, t)
		}
	}
	return result
}

// Median returns the median amount from a slice of trades.
func Median(trades []Trade) int {
	if len(trades) == 0 {
		return 0
	}
	amounts := make([]int, len(trades))
	for i, t := range trades {
		amounts[i] = t.Amount
	}
	sort.Ints(amounts)
	n := len(amounts)
	if n%2 == 0 {
		return (amounts[n/2-1] + amounts[n/2]) / 2
	}
	return amounts[n/2]
}

// Last36Months returns the last 36 months in YYYYMM format, newest first.
func Last36Months() []string {
	now := time.Now()
	months := make([]string, 36)
	for i := range months {
		months[i] = now.AddDate(0, -i, 0).Format("200601")
	}
	return months
}

// Last3Months returns the most recent 3 months in YYYYMM format, newest first.
func Last3Months() []string {
	now := time.Now()
	months := make([]string, 3)
	for i := range months {
		months[i] = now.AddDate(0, -i, 0).Format("200601")
	}
	return months
}

func parseDealAmount(s string) (int, error) {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
}

func toFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(val), 64)
	}
	return 0, fmt.Errorf("cannot convert %T to float64", v)
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(val))
		return n
	}
	return 0
}
