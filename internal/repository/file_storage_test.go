package repository

import (
	models "collector/internal/model"
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFileBackedStorage_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "metrics.json")
	log := zap.NewNop()

	storage := NewFileBackedStorage(filePath, false, log)
	ctx := context.Background()

	err := storage.UpdateGauge(ctx, "Alloc", 123.45)
	require.NoError(t, err)

	err = storage.UpdateCounter(ctx, "PollCount", 10)
	require.NoError(t, err)

	val := 99.9
	delta := int64(5)
	err = storage.UpdateMetrics(ctx, []models.Metrics{
		{ID: "HeapAlloc", MType: models.Gauge, Value: &val},
		{ID: "PollCount", MType: models.Counter, Delta: &delta},
	})
	require.NoError(t, err)

	err = storage.Save()
	require.NoError(t, err)

	// Restore into new storage
	newStorage := NewFileBackedStorage(filePath, false, log)
	err = newStorage.Load()
	require.NoError(t, err)

	gVal, ok, err := newStorage.GetGauge(ctx, "Alloc")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 123.45, gVal)

	cVal, ok, err := newStorage.GetCounter(ctx, "PollCount")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(15), cVal)
}

func TestFileBackedStorage_SyncMode(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sync_metrics.json")
	log := zap.NewNop()

	storage := NewFileBackedStorage(filePath, true, log)
	ctx := context.Background()

	err := storage.UpdateGauge(ctx, "CpuUtil", 45.5)
	require.NoError(t, err)

	// File should exist immediately after update in sync mode
	newStorage := NewFileBackedStorage(filePath, false, log)
	err = newStorage.Load()
	require.NoError(t, err)

	gVal, ok, err := newStorage.GetGauge(ctx, "CpuUtil")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 45.5, gVal)
}
