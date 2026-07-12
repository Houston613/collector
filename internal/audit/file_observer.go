package audit

import (
	"encoding/json"
	"os"
	"sync"
)

// FileObserver writes audit events to a file.
type FileObserver struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	return &FileObserver{file: f, enc: enc}, nil
}

func (fo *FileObserver) Notify(event AuditEvent) error {
	fo.mu.Lock()
	defer fo.mu.Unlock()
	return fo.enc.Encode(event)
}

func (fo *FileObserver) Close() error {
	fo.mu.Lock()
	defer fo.mu.Unlock()
	return fo.file.Close()
}
