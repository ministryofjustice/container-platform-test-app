package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxNameLen = 100

type store struct {
	pool *pgxpool.Pool
}

// openStore returns a nil store when PGHOST is unset, so the app still runs without a database.
func openStore(ctx context.Context) (*store, error) {
	if os.Getenv("PGHOST") == "" {
		return nil, nil
	}

	// An empty DSN makes pgx read the standard libpq PG* environment variables,
	// so no credentials are ever built into a connection string here.
	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	s := &store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *store) close() {
	if s != nil {
		s.pool.Close()
	}
}

func (s *store) migrate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS entries (
			id         BIGSERIAL PRIMARY KEY,
			name       TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	return err
}

func (s *store) add(ctx context.Context, name string) error {
	_, err := s.pool.Exec(ctx, "INSERT INTO entries (name) VALUES ($1)", name)
	return err
}

func (s *store) exists(ctx context.Context, name string) (bool, error) {
	var found bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM entries WHERE name = $1)", name).Scan(&found)
	return found, err
}

func validName(raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	if name == "" || len(name) > maxNameLen {
		return "", false
	}
	return name, true
}
