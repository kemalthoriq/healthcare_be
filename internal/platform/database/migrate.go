package database

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS, log *slog.Logger) error {
	db := stdlib.OpenDBFromPool(pool)

	defer db.Close()

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("create migration locker: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	for _, r := range results {
		log.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}

	current, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	log.Info("database schema up to date", "version", current, "applied", len(results))

	return nil
}
