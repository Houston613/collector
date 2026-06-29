// Субъект (Notifier) хранит список наблюдателей (Observer) и уведомляет их
// после успешной обработки каждого пакета метрик.
package audit

import "time"

// AuditEvent — событие аудита, сформированное после успешного обновления метрик.
type AuditEvent struct {
	// TS — unix timestamp события (секунды).
	TS int64 `json:"ts"`
	// Metrics — список наименований полученных метрик.
	Metrics []string `json:"metrics"`
	// IPAddress — IP-адрес входящего запроса.
	IPAddress string `json:"ip_address"`
}

// NewEvent создаёт новое событие аудита с текущим unix timestamp.
func NewEvent(metrics []string, ipAddress string) AuditEvent {
	return AuditEvent{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}
}

// Observer — интерфейс наблюдателя. Каждая реализация знает, как записать событие аудита в свой приёмник.
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

// Notify рассылает событие всем зарегистрированным наблюдателям.
func (n *Notifier) Notify(event AuditEvent) error {
	var firstErr error
	for _, o := range n.observers {
		if err := o.Notify(event); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Close закрывает все наблюдатели.
func (n *Notifier) Close() error {
	var firstErr error
	for _, o := range n.observers {
		if err := o.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
