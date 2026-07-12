// Package audit implements the observer pattern for auditing metric events.
// The Notifier stores a list of Observers and notifies them after each successfully processed metrics batch.
package audit

import "time"

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
}

func NewNotifier(observers ...Observer) *Notifier {
	return &Notifier{observers: observers}
}

func (n *Notifier) Register(o Observer) {
	n.observers = append(n.observers, o)
}

// Notify sends the event to all registered observers.
func (n *Notifier) Notify(event AuditEvent) error {
	var firstErr error
	for _, o := range n.observers {
		if err := o.Notify(event); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Close closes all registered observers.
func (n *Notifier) Close() error {
	var firstErr error
	for _, o := range n.observers {
		if err := o.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
