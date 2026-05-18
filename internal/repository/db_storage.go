package repository

import (
	models "collector/internal/model"
	"collector/pkg/retry"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	pool *pgxpool.Pool
}

func NewDBStorage(ctx context.Context, dsn string) (*DBStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgxpool: %w", err)
	}
	return &DBStorage{pool: pool}, nil
}

func (d *DBStorage) Bootstrap(migrationsPath string) error {
	// Для миграций используем database/sql, так как golang-migrate лучше всего работает с ним
	db, err := sql.Open("pgx", d.pool.Config().ConnString())
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := retry.Do(context.Background(), func() error {
		err := m.Up()
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		return nil
	}, isRetriableDB); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func isRetriableDB(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		//можно сделать было вот так
		//return strings.HasPrefix(pgErr.Code, "08")
		return pgerrcode.IsConnectionException(pgErr.Code)
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

func (d *DBStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	return retry.Do(ctx, func() error {
		_, err := d.pool.Exec(ctx, `
			INSERT INTO gauges (id, value)
			VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value`,
			name, value)
		return err
	}, isRetriableDB)
}

func (d *DBStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	return retry.Do(ctx, func() error {
		_, err := d.pool.Exec(ctx, `
			INSERT INTO counters (id, delta)
			VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET delta = counters.delta + EXCLUDED.delta`,
			name, value)
		return err
	}, isRetriableDB)
}

func (d *DBStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	return retry.Do(ctx, func() error {
		batch := &pgx.Batch{}

		for _, m := range metrics {
			switch m.MType {
			case models.Gauge:
				if m.Value != nil {
					batch.Queue(`
						INSERT INTO gauges (id, value)
						VALUES ($1, $2)
						ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value`,
						m.ID, *m.Value)
				}
			case models.Counter:
				if m.Delta != nil {
					batch.Queue(`
						INSERT INTO counters (id, delta)
						VALUES ($1, $2)
						ON CONFLICT (id) DO UPDATE SET delta = counters.delta + EXCLUDED.delta`,
						m.ID, *m.Delta)
				}
			}
		}

		br := d.pool.SendBatch(ctx, batch)
		defer br.Close()

		for i := 0; i < batch.Len(); i++ {
			_, err := br.Exec()
			if err != nil {
				return fmt.Errorf("failed to execute batch item %d: %w", i, err)
			}
		}

		return nil
	}, isRetriableDB)
}

func (d *DBStorage) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	var val float64
	var found bool
	err := retry.Do(ctx, func() error {
		err := d.pool.QueryRow(ctx, "SELECT value FROM gauges WHERE id = $1", name).Scan(&val)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				found = false
				return nil
			}
			return err
		}
		found = true
		return nil
	}, isRetriableDB)

	if err != nil {
		return 0, false, err
	}
	return val, found, nil
}

func (d *DBStorage) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	var delta int64
	var found bool
	err := retry.Do(ctx, func() error {
		err := d.pool.QueryRow(ctx, "SELECT delta FROM counters WHERE id = $1", name).Scan(&delta)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				found = false
				return nil
			}
			return err
		}
		found = true
		return nil
	}, isRetriableDB)

	if err != nil {
		return 0, false, err
	}
	return delta, found, nil
}

func (d *DBStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	var gauges map[string]float64
	err := retry.Do(ctx, func() error {
		rows, err := d.pool.Query(ctx, "SELECT id, value FROM gauges")
		if err != nil {
			return err
		}
		defer rows.Close()

		gauges = make(map[string]float64)
		for rows.Next() {
			var id string
			var val float64
			if err := rows.Scan(&id, &val); err != nil {
				return err
			}
			gauges[id] = val
		}
		return rows.Err()
	}, isRetriableDB)

	if err != nil {
		return nil, err
	}
	return gauges, nil
}

func (d *DBStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	var counters map[string]int64
	err := retry.Do(ctx, func() error {
		rows, err := d.pool.Query(ctx, "SELECT id, delta FROM counters")
		if err != nil {
			return err
		}
		defer rows.Close()

		counters = make(map[string]int64)
		for rows.Next() {
			var id string
			var delta int64
			if err := rows.Scan(&id, &delta); err != nil {
				return err
			}
			counters[id] = delta
		}
		return rows.Err()
	}, isRetriableDB)

	if err != nil {
		return nil, err
	}
	return counters, nil
}

func (d *DBStorage) Ping(ctx context.Context) error {
	return retry.Do(ctx, func() error {
		return d.pool.Ping(ctx)
	}, isRetriableDB)
}

func (d *DBStorage) Close() {
	d.pool.Close()
}
