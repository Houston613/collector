package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockObserver struct {
	events []AuditEvent
	err    error
	closed bool
}

func (m *mockObserver) Notify(event AuditEvent) error {
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, event)
	return nil
}

func (m *mockObserver) Close() error {
	m.closed = true
	return nil
}

func TestAuditEvent(t *testing.T) {
	evt := NewEvent([]string{"metric1", "metric2"}, "127.0.0.1")
	assert.Equal(t, []string{"metric1", "metric2"}, evt.Metrics)
	assert.Equal(t, "127.0.0.1", evt.IPAddress)
	assert.True(t, evt.TS > 0)
}

func TestNotifier_WorkerPool(t *testing.T) {
	log := zap.NewNop()
	mock := &mockObserver{}

	notifier := NewNotifier(log, mock)
	require.NotNil(t, notifier)

	evt := NewEvent([]string{"Alloc"}, "192.168.1.1")
	err := notifier.Notify(evt)
	assert.NoError(t, err)

	err = notifier.Close()
	assert.NoError(t, err)
	assert.True(t, mock.closed)
	assert.Len(t, mock.events, 1)
	assert.Equal(t, "Alloc", mock.events[0].Metrics[0])
}

func TestFileObserver(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "audit.log")

	fo, err := NewFileObserver(filePath)
	require.NoError(t, err)
	require.NotNil(t, fo)

	evt := NewEvent([]string{"PollCount"}, "10.0.0.1")
	err = fo.Notify(evt)
	assert.NoError(t, err)

	err = fo.Close()
	assert.NoError(t, err)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var readEvt AuditEvent
	err = json.Unmarshal(data, &readEvt)
	require.NoError(t, err)
	assert.Equal(t, "PollCount", readEvt.Metrics[0])
	assert.Equal(t, "10.0.0.1", readEvt.IPAddress)
}

func TestHTTPObserver(t *testing.T) {
	received := make(chan AuditEvent, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var evt AuditEvent
		_ = json.NewDecoder(r.Body).Decode(&evt)
		received <- evt
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ho := NewHTTPObserver(ts.URL)
	evt := NewEvent([]string{"HeapAlloc"}, "127.0.0.1")
	err := ho.Notify(evt)
	assert.NoError(t, err)

	select {
	case got := <-received:
		assert.Equal(t, "HeapAlloc", got.Metrics[0])
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for HTTP audit event")
	}

	assert.NoError(t, ho.Close())
}
