package psql

import (
	"context"
	"embed"
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

func NewConnection() (*pgxpool.Pool, error) {
	connectionString := "postgres://postgres:password@localhost:5432/synthy"

	if err := runMigrations(connectionString); err != nil {
		slog.Error("Failed to run database migrations", "error", err)
		return nil, err
	}

	dbPool, err := pgxpool.New(context.Background(), connectionString)
	if err != nil {
		slog.Error("Unable to create connection pool", "error", err)
		return nil, err
	}

	defer dbPool.Close()

	slog.Info("Connected to database successfully")
	return dbPool, nil
}

func runMigrations(connectionString string) error {
	slog.Info("Running database migrations...")

	src, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, toPgx5DSN(connectionString))
	if err != nil {
		return err
	}

	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	version, dirty, err := m.Version()
	if err != nil {
		return err
	}

	slog.Info("Migrations complete", "version", version, "dirty", dirty)
	return nil
}

func toPgx5DSN(connectionString string) string {
	prefix := "postgres://"
	return "pgx5://" + connectionString[len(prefix):]
}
