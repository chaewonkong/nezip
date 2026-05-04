package cache

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"math"
	"time"

	"nezip/internal/db"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	conn  *sql.DB
	query *db.Queries
}

func Open(ctx context.Context, path string) (*Store, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("conn open failed: %w", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping failed: %w", err)
	}
	if _, err := conn.ExecContext(ctx, schemaSQL); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	return &Store{conn: conn, query: db.New(conn)}, nil
}

func (s *Store) Close() error {
	return s.conn.Close()
}

// SaveTrade saves a trade record. Current month is upserted; past months are ignored if already cached.
func (s *Store) SaveTrade(ctx context.Context, aptName, lawdCD, month string, area float64, areaInt int64, median, count int) error {
	p := db.Price{
		AptName: aptName,
		LawdCd:  lawdCD,
		Month:   month,
		AreaInt: areaInt,
		Area:    area,
		Median:  int64(median),
		Count:   int64(count),
	}
	cur := time.Now().Format("200601")
	if month >= cur {
		if err := s.query.UpsertPrice(ctx, db.UpsertPriceParams(p)); err != nil {
			return fmt.Errorf("upsert %s/%s/%s: %w", aptName, lawdCD, month, err)
		}
		return nil
	}
	if err := s.query.InsertPriceIgnore(ctx, db.InsertPriceIgnoreParams(p)); err != nil {
		return fmt.Errorf("insert %s/%s/%s: %w", aptName, lawdCD, month, err)
	}
	return nil
}

// CachedMonthsSet returns the set of months already in DB for the given apt.
func (s *Store) CachedMonthsSet(ctx context.Context, aptName, lawdCD string, areaInt int64) (map[string]bool, error) {
	months, err := s.query.GetCachedMonths(ctx, db.GetCachedMonthsParams{
		AptName: aptName,
		LawdCd:  lawdCD,
		AreaInt: areaInt,
	})
	if err != nil {
		return nil, fmt.Errorf("get cached months: %w", err)
	}
	set := make(map[string]bool, len(months))
	for _, m := range months {
		set[m] = true
	}
	return set, nil
}

// LoadPrices returns a month→median map for the given apt.
func (s *Store) LoadPrices(ctx context.Context, aptName, lawdCD string, areaInt int64) (map[string]int, error) {
	rows, err := s.query.ListPricesByApt(ctx, db.ListPricesByAptParams{
		AptName: aptName,
		LawdCd:  lawdCD,
		AreaInt: areaInt,
	})
	if err != nil {
		return nil, fmt.Errorf("list prices: %w", err)
	}
	prices := make(map[string]int, len(rows))
	for _, r := range rows {
		prices[r.Month] = int(r.Median)
	}
	return prices, nil
}

// LoadPricesWithTolerance loads prices matching area within ±tolerance.
// Used for the ±10㎡ fallback.
func (s *Store) LoadPricesWithTolerance(ctx context.Context, aptName, lawdCD string, targetArea, tolerance float64) (map[string]int, error) {
	// load all area_int values near target and merge
	prices := make(map[string]int)
	for offset := -int(tolerance); offset <= int(tolerance); offset++ {
		areaInt := int64(math.Round(targetArea)) + int64(offset)
		rows, err := s.query.ListPricesByApt(ctx, db.ListPricesByAptParams{
			AptName: aptName,
			LawdCd:  lawdCD,
			AreaInt: areaInt,
		})
		if err != nil {
			continue
		}
		for _, r := range rows {
			if _, exists := prices[r.Month]; !exists {
				prices[r.Month] = int(r.Median)
			}
		}
	}
	return prices, nil
}
