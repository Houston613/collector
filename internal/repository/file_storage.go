package repository

import (
	models "collector/internal/model"
	"context"
	"encoding/json"
	"os"
	"sync"

	"go.uber.org/zap"
)

// переиспользуем StructMem, чтобы не дублировать код по работе с метриками в памяти
type FileBackedStorage struct {
	*StructMem
	filePath string
	syncMode bool // true → сохранять на диск после каждой записи (storeInterval == 0)
	fileMu   sync.Mutex
	log      *zap.Logger
}

func NewFileBackedStorage(filePath string, syncMode bool, log *zap.Logger) *FileBackedStorage {
	return &FileBackedStorage{
		StructMem: NewStructMem(),
		filePath:  filePath,
		syncMode:  syncMode,
		log:       log,
	}
}

func (f *FileBackedStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	f.StructMem.UpdateGauge(ctx, name, value)
	if f.syncMode {
		if err := f.Save(); err != nil {
			f.log.Error("не удалось синхронизировать метрики", zap.String("path", f.filePath), zap.Error(err))
			return err
		}
	}
	return nil
}

func (f *FileBackedStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	f.StructMem.UpdateCounter(ctx, name, value)
	if f.syncMode {
		if err := f.Save(); err != nil {
			f.log.Error("не удалость синхронизировать метрики", zap.String("path", f.filePath), zap.Error(err))
			return err
		}
	}
	return nil
}

func (f *FileBackedStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	f.StructMem.UpdateMetrics(ctx, metrics)
	if f.syncMode {
		if err := f.Save(); err != nil {
			f.log.Error("не удалось синхронизировать метрики", zap.String("path", f.filePath), zap.Error(err))
			return err
		}
	}
	return nil
}

// Save сериализует все текущие метрики в JSON и записывает в файл.
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

// Load читает метрики из файла и загружает их в хранилище.
// Если файл не существует — не ошибка, просто ничего не загружается.
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
