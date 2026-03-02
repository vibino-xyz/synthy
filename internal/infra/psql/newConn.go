package psql

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnection() (*pgxpool.Pool, error) {
	dbPool, err := pgxpool.New(context.Background(), "postgres://postgres:password@localhost:5432/synthy")
	if err != nil {
		slog.Error("Unable to create connection pool", "error", err)
		return nil, err
	}

	defer dbPool.Close()

	slog.Info("Connected to database successfully")
	return dbPool, nil
}
