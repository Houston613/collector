package repository

import (
	models "collector/internal/model"
	"context"
	"encoding/json"
	"os"
	"sync"

	"go.uber.org/zap"
)

// FileBackedStorage extends StructMem with functionality to persist metrics to a file on disk.
type FileBackedStorage struct {
	*StructMem
	filePath string
	syncMode bool // true to save to disk after every write (storeInterval == 0)
	fileMu   sync.Mutex
	log      *zap.Logger
}

// NewFileBackedStorage creates and returns a new FileBackedStorage instance.
func NewFileBackedStorage(filePath string, syncMode bool, log *zap.Logger) *FileBackedStorage {
	return &FileBackedStorage{
		StructMem: NewStructMem(),
		filePath:  filePath,
		syncMode:  syncMode,
		log:       log,
	}
}

// UpdateGauge updates a gauge metric and persists the state if syncMode is enabled.
func (f *FileBackedStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	f.StructMem.UpdateGauge(ctx, name, value)
	if f.syncMode {
		if err := f.Save(); err != nil {
			f.log.Error("failed to synchronize metrics", zap.String("path", f.filePath), zap.Error(err))
			return err
		}
	}
	return nil
}

// UpdateCounter updates a counter metric and persists the state if syncMode is enabled.
func (f *FileBackedStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	f.StructMem.UpdateCounter(ctx, name, value)
	if f.syncMode {
		if err := f.Save(); err != nil {
			f.log.Error("failed to synchronize metrics", zap.String("path", f.filePath), zap.Error(err))
			return err
		}
	}
	return nil
}

// UpdateMetrics updates multiple metrics and persists the state if syncMode is enabled.
func (f *FileBackedStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	f.StructMem.UpdateMetrics(ctx, metrics)
	if f.syncMode {
		if err := f.Save(); err != nil {
			f.log.Error("failed to synchronize metrics", zap.String("path", f.filePath), zap.Error(err))
			return err
		}
	}
	return nil
}

// Save serializes all current metrics to JSON and writes them to a file.
func (f *FileBackedStorage) Save() error {
	ctx := context.Background()
	gauges, _ := f.GetAllGauges(ctx)
	counters, _ := f.GetAllCounters(ctx)

	metrics := make([]models.Metrics, 0, len(gauges)+len(counters))
	for name, v := range gauges {
		val := v
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Gauge, Value: &val})
	}
	for name, d := range counters {
		delta := d
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Counter, Delta: &delta})
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	f.fileMu.Lock()
	defer f.fileMu.Unlock()
	return os.WriteFile(f.filePath, data, 0644)
}

// Load reads metrics from a file and loads them into memory.
// If the file does not exist, it is not an error; it simply skips loading.
func (f *FileBackedStorage) Load() error {
	f.fileMu.Lock()
	data, err := os.ReadFile(f.filePath)
	f.fileMu.Unlock()

	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	ctx := context.Background()
	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				f.StructMem.UpdateGauge(ctx, m.ID, *m.Value)
			}
		case models.Counter:
			if m.Delta != nil {
				f.StructMem.UpdateCounter(ctx, m.ID, *m.Delta)
			}
		}
	}
	return nil
}
