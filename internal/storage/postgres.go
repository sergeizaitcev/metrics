package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/sergeizaitcev/metrics/deployments/migrations"
	"github.com/sergeizaitcev/metrics/internal/metrics"
)

var _ Storage = (*Postgres)(nil)

// Storage определяет хранилище метрик в postgres.
type Postgres struct {
	db *sql.DB
}

// New возвращает новый экземпляр хранилища метрик в postgres.
func NewPostgres(dsn string) (*Postgres, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: connection refused: %w", err)
	}

	return &Postgres{db: db}, nil
}

// MigrateUp запускает миграцию в БД.
func (p *Postgres) MigrateUp(ctx context.Context) error {
	err := migrations.Up(ctx, p.db)
	if err != nil {
		return fmt.Errorf("postgres: migration up: %w", err)
	}
	return nil
}

// MigrateDown откатывает миграцию в БД.
func (p *Postgres) MigrateDown(ctx context.Context) error {
	err := migrations.Down(ctx, p.db)
	if err != nil {
		return fmt.Errorf("postgres: migration down: %w", err)
	}
	return nil
}

// Ping реализует интерфейс Storager.
func (p *Postgres) Ping(context.Context) error {
	err := p.db.Ping()
	if err != nil {
		return fmt.Errorf("postgres: ping to database: %w", err)
	}
	return nil
}

// Close реализует интерфейс Storager.
func (p *Postgres) Close() error {
	err := p.db.Close()
	if err != nil {
		return fmt.Errorf("postgres: closing database: %w", err)
	}
	return nil
}

// Save реализует интерфейс Storager.
func (p *Postgres) Save(ctx context.Context, values ...metrics.Metric) ([]metrics.Metric, error) {
	if len(values) == 0 {
		return nil, errors.New("values is empty")
	}

	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return nil, fmt.Errorf("postgres: begin transaction: %w", err)
	}
	defer tx.Rollback()

	actuals := make([]metrics.Metric, len(values))

	for i, value := range values {
		if value.IsEmpty() {
			continue
		}

		var actual metrics.Metric

		switch value.Kind() {
		case metrics.KindCounter:
			actual, err = p.add(ctx, tx, value)
		case metrics.KindGauge:
			actual, err = p.update(ctx, tx, value)
		}
		if err != nil {
			return nil, fmt.Errorf("postgres: saving metrics: %w", err)
		}

		actuals[i] = actual
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("postgres: commit transaction: %w", err)
	}

	return actuals, nil
}

// add увеличивает значение метрики и возвращает его.
func (p *Postgres) add(
	ctx context.Context,
	tx *sql.Tx,
	value metrics.Metric,
) (metrics.Metric, error) {
	query := `INSERT INTO
		metrics (name, kind, counter)
	VALUES
		($1, $2, $3)
	ON CONFLICT (name, kind) DO
	UPDATE
		SET counter = metrics.counter + $3
	WHERE
		metrics.name = $1 AND metrics.kind = $2
	RETURNING counter;`

	var actual int64

	err := tx.QueryRowContext(
		ctx,
		query,
		value.Name(),
		value.Kind(),
		value.Int64(),
	).Scan(&actual)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("increasing the value: %w", err)
	}

	return metrics.Counter(value.Name(), actual), nil
}

// update обновляет значение метрики и возвращает предыдущее.
func (p *Postgres) update(
	ctx context.Context,
	tx *sql.Tx,
	value metrics.Metric,
) (metrics.Metric, error) {
	query := `WITH old AS (
		SELECT gauge FROM metrics WHERE name = $1 AND kind = $2
	)
	INSERT INTO
		metrics (name, kind, gauge)
	VALUES
		($1, $2, $3)
	ON CONFLICT (name, kind) DO
	UPDATE
		SET gauge = $3
		WHERE metrics.name = $1 AND metrics.kind = $2
	RETURNING (SELECT * FROM old);`

	var (
		old    sql.NullFloat64
		metric metrics.Metric
	)

	err := p.db.QueryRowContext(
		ctx,
		query,
		value.Name(),
		value.Kind(),
		value.Float64(),
	).Scan(&old)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("updating the value: %w", err)
	}

	if old.Valid {
		metric = metrics.Gauge(value.Name(), old.Float64)
	}

	return metric, nil
}

// Get реализует интерфейс storage.Storager.
func (p *Postgres) Get(ctx context.Context, name string) (metrics.Metric, error) {
	query := "SELECT kind, counter, gauge FROM metrics WHERE name = $1 LIMIT 1;"

	row := p.db.QueryRowContext(ctx, query, name)
	err := row.Err()
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("postgres: execution query: %w", err)
	}

	var (
		kind    metrics.Kind
		counter sql.NullInt64
		gauge   sql.NullFloat64
	)

	err = row.Scan(&kind, &counter, &gauge)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return metrics.Metric{}, fmt.Errorf("postgres: scan row: %w", err)
	}

	var metric metrics.Metric

	switch kind {
	case metrics.KindCounter:
		metric = metrics.Counter(name, counter.Int64)
	case metrics.KindGauge:
		metric = metrics.Gauge(name, gauge.Float64)
	}

	return metric, nil
}

// GetAll реализует интерфейс Storager.
func (p *Postgres) GetAll(ctx context.Context) ([]metrics.Metric, error) {
	query := "SELECT name, kind, counter, gauge FROM metrics ORDER BY name;"

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: execution query: %w", err)
	}
	defer rows.Close()

	values := make([]metrics.Metric, 0, 64)

	for rows.Next() {
		var (
			name    string
			kind    metrics.Kind
			counter sql.NullInt64
			gauge   sql.NullFloat64
		)

		err = rows.Scan(&name, &kind, &counter, &gauge)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan row: %w", err)
		}

		var metric metrics.Metric

		switch kind {
		case metrics.KindCounter:
			metric = metrics.Counter(name, counter.Int64)
		case metrics.KindGauge:
			metric = metrics.Gauge(name, gauge.Float64)
		}

		values = append(values, metric)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate by rows: %w", err)
	}
	if len(values) == 0 {
		return nil, ErrNotFound
	}

	return values, nil
}
