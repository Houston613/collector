// Package audit implements the observer pattern for auditing metric events.
// The Notifier stores a list of Observers and notifies them after each successfully processed metrics batch.
package audit

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AuditEvent represents an audit event created after successfully updating metrics.
type AuditEvent struct {
	// TS is the unix timestamp of the event (seconds).
	TS int64 `json:"ts"`
	// Metrics is the list of metric names received.
	Metrics []string `json:"metrics"`
	// IPAddress is the IP address of the incoming request.
	IPAddress string `json:"ip_address"`
}

// NewEvent creates a new audit event with the current unix timestamp.
func NewEvent(metrics []string, ipAddress string) AuditEvent {
	return AuditEvent{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}
}

// Observer defines the interface for an observer that records audit events to its destination.
type Observer interface {
	Notify(event AuditEvent) error
	Close() error
}

type Notifier struct {
	observers []Observer
	events    chan AuditEvent
	wg        sync.WaitGroup
	log       *zap.Logger
}

// NewNotifier creates a Notifier with default worker pool (5 workers, buffer 1000).
func NewNotifier(log *zap.Logger, observers ...Observer) *Notifier {
	return NewNotifierWithPool(log, 5, 1000, observers...)
}

// NewNotifierWithPool creates a Notifier with custom worker pool size and buffer capacity.
func NewNotifierWithPool(log *zap.Logger, workers, bufferSize int, observers ...Observer) *Notifier {
	if workers <= 0 {
		workers = 5
	}
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	if log == nil {
		log = zap.NewNop()
	}

	n := &Notifier{
		observers: observers,
		events:    make(chan AuditEvent, bufferSize),
		log:       log,
	}

	for i := 0; i < workers; i++ {
		n.wg.Add(1)
		go n.worker()
	}

	return n
}

func (n *Notifier) worker() {
	defer n.wg.Done()
	for event := range n.events {
		if err := n.notifyObservers(event); err != nil {
			n.log.Error("failed to deliver audit event to observers",
				zap.Error(err),
				zap.Strings("metrics", event.Metrics),
				zap.String("ip", event.IPAddress),
			)
		}
	}
}

func (n *Notifier) notifyObservers(event AuditEvent) error {
	var firstErr error
	for _, o := range n.observers {
		if err := o.Notify(event); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (n *Notifier) Register(o Observer) {
	n.observers = append(n.observers, o)
}

// Notify enqueues the event to be processed asynchronously by the worker pool.
// If the buffer is full, it drops the event and logs a warning to prevent blocking the HTTP handler.
func (n *Notifier) Notify(event AuditEvent) error {
	select {
	case n.events <- event:
		return nil
	default:
		err := errors.New("audit event buffer full")
		n.log.Warn("audit event dropped due to full buffer",
			zap.Error(err),
			zap.Strings("metrics", event.Metrics),
			zap.String("ip", event.IPAddress),
		)
		return err
	}
}

// Close drains remaining events, waits for workers to finish, and closes observers.
func (n *Notifier) Close() error {
	close(n.events)
	n.wg.Wait()

	var firstErr error
	for _, o := range n.observers {
		if err := o.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}


