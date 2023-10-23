package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"

	"github.com/sergeizaitcev/metrics/deployments/migrations"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage"
)

// Storage определяет хранилище метрик в postgres.
type Storage struct {
	db *sql.DB
}

// New возвращает новый экземпляр хранилища метрик в postgres.
func New(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// Up запускает миграцию в БД.
func (s *Storage) Up(ctx context.Context) error {
	return migrations.Up(ctx, s.db)
}

// Down откатывает миграцию в БД.
func (s *Storage) Down(ctx context.Context) error {
	return migrations.Down(ctx, s.db)
}

// Ping выполняет пинг к БД.
func (s *Storage) Ping(context.Context) error {
	err := s.db.Ping()
	if err != nil {
		return fmt.Errorf("postgres: ping to database: %w", err)
	}
	return nil
}

// Close закрывает соединение с БД.
func (s *Storage) Close() error {
	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("postgres: closing database: %w", err)
	}
	return nil
}

// SaveMany устанавливает или увеличивает значения метрики.
func (s *Storage) SaveMany(ctx context.Context, values []metrics.Metric) error {
	if len(values) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin transaction: %w", err)
	}
	defer tx.Rollback()

	addQuery := `INSERT INTO
		metrics (name, kind, counter)
	VALUES
		($1, $2, $3)
	ON CONFLICT (name, kind) DO
	UPDATE
		SET counter = metrics.counter + $3
		WHERE metrics.name = $1 AND metrics.kind = $2`

	setQuery := `INSERT INTO
		metrics (name, kind, gauge)
	VALUES
		($1, $2, $3)
	ON CONFLICT (name, kind) DO
	UPDATE
		SET gauge = $3
		WHERE metrics.name = $1 AND metrics.kind = $2;`

	for _, value := range values {
		if value.IsEmpty() {
			continue
		}

		var err error

		switch value.Kind() {
		case metrics.KindCounter:
			_, err = tx.ExecContext(ctx, addQuery, value.Name(), value.Kind(), value.Int64())
		case metrics.KindGauge:
			_, err = tx.ExecContext(ctx, setQuery, value.Name(), value.Kind(), value.Float64())
		}

		if err != nil {
			impl, ok := err.(*pq.Error)
			if ok && pgerrcode.IsIntegrityConstraintViolation(string(impl.Code)) {
				continue
			}
			return fmt.Errorf("postgres: save metric: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("postgres: commit transaction: %w", err)
	}

	return nil
}

// Add увеличивает метрику и возвращает актуальное значение.
func (s *Storage) Add(ctx context.Context, value metrics.Metric) (metrics.Metric, error) {
	if value.IsEmpty() {
		return metrics.Metric{}, errors.New("postgres: metric is empty")
	}
	if value.Kind() != metrics.KindCounter {
		return metrics.Metric{}, fmt.Errorf("postgres: expected to get a metric kind %s, got %s",
			metrics.KindCounter, value.Kind(),
		)
	}

	query := `INSERT INTO
		metrics (name, kind, counter)
	VALUES
		($1, $2, $3)
	ON CONFLICT (name, kind) DO
	UPDATE
		SET counter = metrics.counter + $3
		WHERE metrics.name = $1 AND metrics.kind = $2
	RETURNING counter;`

	var actual int64

	err := s.db.QueryRowContext(
		ctx,
		query,
		value.Name(),
		value.Kind(),
		value.Int64(),
	).Scan(&actual)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("postgres: execution query: %w", err)
	}

	return metrics.Counter(value.Name(), actual), nil
}

// Set устанавливает новое значение метрики и возвращает предыдущее значение.
func (s *Storage) Set(ctx context.Context, value metrics.Metric) (metrics.Metric, error) {
	if value.IsEmpty() {
		return metrics.Metric{}, errors.New("local: metric is empty")
	}
	if value.Kind() != metrics.KindGauge {
		return metrics.Metric{}, fmt.Errorf("postgres: expected to get a metric kind %s, got %s",
			metrics.KindGauge, value.Kind(),
		)
	}

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

	err := s.db.QueryRowContext(
		ctx,
		query,
		value.Name(),
		value.Kind(),
		value.Float64(),
	).Scan(&old)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("postgres: execution query: %w", err)
	}

	if old.Valid {
		metric = metrics.Gauge(value.Name(), old.Float64)
	}

	return metric, nil
}

// Get возвращает метрику.
func (s *Storage) Get(ctx context.Context, name string) (metrics.Metric, error) {
	query := "SELECT kind, counter, gauge FROM metrics WHERE name = $1 LIMIT 1;"

	row := s.db.QueryRowContext(ctx, query, name)
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
			err = storage.ErrNotFound
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

// GetAll возвращает все метрики.
func (s *Storage) GetAll(ctx context.Context) ([]metrics.Metric, error) {
	query := "SELECT name, kind, counter, gauge FROM metrics ORDER BY name;"

	rows, err := s.db.QueryContext(ctx, query)
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
		if errors.Is(err, sql.ErrNoRows) {
			err = storage.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: iterate by rows: %w", err)
	}

	return values, nil
}
