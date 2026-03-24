package repository

import (
	models "collector/internal/model"
	"encoding/json"
	"os"
	"sync"
)

//переиспользуем StructMem, чтобы не дублировать код по работе с метриками в памяти
type FileBackedStorage struct {
	*StructMem
	filePath string
	fileMu   sync.Mutex
}

func NewFileBackedStorage(filePath string) *FileBackedStorage {
	return &FileBackedStorage{
		StructMem: NewStructMem(),
		filePath:  filePath,
	}
}

func (f *FileBackedStorage) UpdateGauge(name string, value float64) {
	f.StructMem.UpdateGauge(name, value)
	f.Save()
}

func (f *FileBackedStorage) UpdateCounter(name string, value int64) {
	f.StructMem.UpdateCounter(name, value)
	f.Save()
}

// Save сериализует все текущие метрики в JSON и записывает в файл.
func (f *FileBackedStorage) Save() error {
	gauges := f.GetAllGauges()
	counters := f.GetAllCounters()

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

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				f.StructMem.UpdateGauge(m.ID, *m.Value)
			}
		case models.Counter:
			if m.Delta != nil {
				f.StructMem.UpdateCounter(m.ID, *m.Delta)
			}
		}
	}
	return nil
}
