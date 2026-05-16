package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	return &DBStorage{db: db}, nil
}

func (d *DBStorage) Bootstrap(migrationsPath string) error {
	driver, err := postgres.WithInstance(d.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (d *DBStorage) UpdateGauge(name string, value float64) error {
	_, err := d.db.Exec(`
		INSERT INTO gauges (id, value)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value`,
		name, value)
	return err
}

func (d *DBStorage) UpdateCounter(name string, value int64) error {
	_, err := d.db.Exec(`
		INSERT INTO counters (id, delta)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET delta = counters.delta + EXCLUDED.delta`,
		name, value)
	return err
}

func (d *DBStorage) GetGauge(name string) (float64, bool, error) {
	var val float64
	err := d.db.QueryRow("SELECT value FROM gauges WHERE id = $1", name).Scan(&val)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return val, true, nil
}

func (d *DBStorage) GetCounter(name string) (int64, bool, error) {
	var delta int64
	err := d.db.QueryRow("SELECT delta FROM counters WHERE id = $1", name).Scan(&delta)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return delta, true, nil
}

func (d *DBStorage) GetAllGauges() (map[string]float64, error) {
	rows, err := d.db.Query("SELECT id, value FROM gauges")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gauges := make(map[string]float64)
	for rows.Next() {
		var id string
		var val float64
		if err := rows.Scan(&id, &val); err != nil {
			return nil, err
		}
		gauges[id] = val
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return gauges, nil
}

func (d *DBStorage) GetAllCounters() (map[string]int64, error) {
	rows, err := d.db.Query("SELECT id, delta FROM counters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counters := make(map[string]int64)
	for rows.Next() {
		var id string
		var delta int64
		if err := rows.Scan(&id, &delta); err != nil {
			return nil, err
		}
		counters[id] = delta
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counters, nil
}

func (d *DBStorage) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *DBStorage) Close() error {
	return d.db.Close()
}
