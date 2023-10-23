package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var fsys embed.FS

// Up запускает миграцию в БД.
func Up(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(fsys)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrations: set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("migrations: up migrations: %w", err)
	}

	return nil
}

// Down откатывает миграцию в БД.
func Down(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(fsys)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrations: set dialect: %w", err)
	}

	if err := goose.DownContext(ctx, db, "."); err != nil {
		return fmt.Errorf("migrations: down migrations: %w", err)
	}

	return nil
}
