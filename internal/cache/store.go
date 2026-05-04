package cache

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"slices"
	"time"

	"nezip/internal/db"
)

//go:embed schema.sql
var schemaSQL string

func Open(ctx context.Context, path string) (*Store, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("conn open failed: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping failed")
	}

	if _, err := conn.ExecContext(ctx, schemaSQL); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return &Store{query: db.New(conn)}, nil
}

type Store struct {
	query *db.Queries
}

// SavePrice 당월 upsert, 이전월 ignore
func (s *Store) SavePrice(ctx context.Context, p db.Price) error {
	months, err := s.query.GetCachedMonths(
		ctx,
		db.GetCachedMonthsParams{
			AptName: p.AptName,
			LawdCd:  p.LawdCd,
			AreaInt: p.AreaInt,
		},
	)
	if err != nil {
		return fmt.Errorf("getting cached months failed: %w", err)
	}

	if slices.Contains(months, p.Month) {
		cur := time.Now().Format("200601")
		if p.Month < cur {
			// ignore past data
			return nil
		}

		err := s.query.UpsertPrice(ctx, db.UpsertPriceParams(p))
		if err != nil {
			return fmt.Errorf("upsert failed: %w", err)
		}

		return nil
	}

	return s.query.InsertPriceIgnore(ctx, db.InsertPriceIgnoreParams(p))
}
