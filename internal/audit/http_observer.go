package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (ho *HTTPObserver) Notify(event AuditEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit http observer: marshal: %w", err)
	}

	resp, err := ho.client.Post(ho.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("audit http observer: post %s: %w", ho.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("audit http observer: server returned %d", resp.StatusCode)
	}

	return nil
}

func (ho *HTTPObserver) Close() error {
	return nil
}
