package agent

import (
	"bytes"
	"collector/pkg/crypto"
	"collector/pkg/retry"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	models "collector/internal/model"
	"collector/pkg/signature"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
)

const (
	// DefaultServerAddress is the default address of the metrics server.
	DefaultServerAddress = "localhost:8080"
	// DefaultPollInterval is the default frequency at which system metrics are gathered.
	DefaultPollInterval = 2 * time.Second
	// DefaultReportInterval is the default frequency at which gathered metrics are sent to the server.
	DefaultReportInterval = 10 * time.Second
)

// Agent collects and periodically reports system runtime metrics to a server.
type Agent struct {
	mu              sync.RWMutex
	gaugesMetrics   map[string]float64
	countersMetrics map[string]int64
	addr            string
	pollInterval    time.Duration
	reportInterval  time.Duration
	key             string
	cryptoKey       *rsa.PublicKey
	rateLimit       int
	client          *http.Client
	log             *zap.Logger
}

// NewAgent creates and configures a new metrics Agent.
func NewAgent(addr string, pollInterval, reportInterval time.Duration, key string, cryptoKey *rsa.PublicKey, rateLimit int, log *zap.Logger) *Agent {
	return &Agent{
		gaugesMetrics:   make(map[string]float64),
		countersMetrics: make(map[string]int64),
		addr:            addr,
		pollInterval:    pollInterval,
		reportInterval:  reportInterval,
		key:             key,
		cryptoKey:       cryptoKey,
		rateLimit:       rateLimit,
		client:          &http.Client{},
		// Add logger to agent struct to log metric transmission errors
		log: log,
	}
}

// MetricsCollect gathers standard memory runtime statistics and stores them internally.
func (a *Agent) MetricsCollect() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	a.mu.Lock()
	defer a.mu.Unlock()

	a.gaugesMetrics["Alloc"] = float64(ms.Alloc)
	a.gaugesMetrics["BuckHashSys"] = float64(ms.BuckHashSys)
	a.gaugesMetrics["Frees"] = float64(ms.Frees)
	a.gaugesMetrics["GCCPUFraction"] = ms.GCCPUFraction
	a.gaugesMetrics["GCSys"] = float64(ms.GCSys)
	a.gaugesMetrics["HeapAlloc"] = float64(ms.HeapAlloc)
	a.gaugesMetrics["HeapIdle"] = float64(ms.HeapIdle)
	a.gaugesMetrics["HeapInuse"] = float64(ms.HeapInuse)
	a.gaugesMetrics["HeapObjects"] = float64(ms.HeapObjects)
	a.gaugesMetrics["HeapReleased"] = float64(ms.HeapReleased)
	a.gaugesMetrics["HeapSys"] = float64(ms.HeapSys)
	a.gaugesMetrics["LastGC"] = float64(ms.LastGC)
	a.gaugesMetrics["Lookups"] = float64(ms.Lookups)
	a.gaugesMetrics["MCacheInuse"] = float64(ms.MCacheInuse)
	a.gaugesMetrics["MCacheSys"] = float64(ms.MCacheSys)
	a.gaugesMetrics["MSpanInuse"] = float64(ms.MSpanInuse)
	a.gaugesMetrics["MSpanSys"] = float64(ms.MSpanSys)
	a.gaugesMetrics["Mallocs"] = float64(ms.Mallocs)
	a.gaugesMetrics["NextGC"] = float64(ms.NextGC)
	a.gaugesMetrics["NumForcedGC"] = float64(ms.NumForcedGC)
	a.gaugesMetrics["NumGC"] = float64(ms.NumGC)
	a.gaugesMetrics["OtherSys"] = float64(ms.OtherSys)
	a.gaugesMetrics["PauseTotalNs"] = float64(ms.PauseTotalNs)
	a.gaugesMetrics["StackInuse"] = float64(ms.StackInuse)
	a.gaugesMetrics["StackSys"] = float64(ms.StackSys)
	a.gaugesMetrics["Sys"] = float64(ms.Sys)
	a.gaugesMetrics["TotalAlloc"] = float64(ms.TotalAlloc)
	// Random value
	a.gaugesMetrics["RandomValue"] = rand.Float64()
	a.countersMetrics["PollCount"]++
}

// MetricsCollectGopsutil gathers additional system metrics like CPU utilization and memory using gopsutil.
func (a *Agent) MetricsCollectGopsutil() {
	v, err := mem.VirtualMemory()
	if err != nil {
		a.log.Error("error getting virtual memory stats", zap.Error(err))
		return
	}

	c, err := cpu.Percent(0, true)
	if err != nil {
		a.log.Error("error getting cpu stats", zap.Error(err))
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.gaugesMetrics["TotalMemory"] = float64(v.Total)
	a.gaugesMetrics["FreeMemory"] = float64(v.Free)

	for i, p := range c {
		a.gaugesMetrics[fmt.Sprintf("CPUutilization%d", i+1)] = p
	}
}

// MetricsSend packages all gathered metrics and writes them to the jobs channel for shipping.
func (a *Agent) MetricsSend(jobs chan<- []models.Metrics) {
	// Copy metrics under RLock to prevent holding the lock during HTTP transmission
	a.mu.RLock()
	gauges := make(map[string]float64, len(a.gaugesMetrics))
	maps.Copy(gauges, a.gaugesMetrics)
	counters := make(map[string]int64, len(a.countersMetrics))
	maps.Copy(counters, a.countersMetrics)
	a.mu.RUnlock()

	metrics := make([]models.Metrics, 0, len(gauges)+len(counters))

	for name, value := range gauges {
		v := value
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Gauge, Value: &v})
	}

	for name, value := range counters {
		v := value
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Counter, Delta: &v})
	}

	if len(metrics) == 0 {
		return
	}

	jobs <- metrics
}

func (a *Agent) worker(jobs <-chan []models.Metrics) {
	for metrics := range jobs {
		if err := a.sendBatchJSON(metrics); err != nil {
			a.log.Error("error sending batch", zap.Error(err))
		}
	}
}

func (a *Agent) sendBatchJSON(metrics []models.Metrics) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}
	// Compress the payload
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("gzip writer: %w", err)
	}
	if _, err = gz.Write(body); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	compressedData := buf.Bytes()
	payload := compressedData

	if a.cryptoKey != nil {
		enc, err := crypto.Encrypt(a.cryptoKey, compressedData)
		if err != nil {
			return fmt.Errorf("encrypt payload: %w", err)
		}
		payload = enc
	}

	return retry.Do(context.Background(), func() error {
		req, err := http.NewRequest(http.MethodPost, a.addr+"/updates/", bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		if a.key != "" {
			hash := signature.Sign(body, a.key)
			req.Header.Set("HashSHA256", hash)
		}

		resp, err := a.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return nil
	}, func(err error) bool {
		var netErr net.Error
		return errors.As(err, &netErr)
	})
}

// Run starts the agent's main loops for collecting and sending metrics.
// It runs until the context is cancelled, after which it performs a final report
// and waits for all outgoing workers to complete.
func (a *Agent) Run(ctx context.Context) {
	jobs := make(chan []models.Metrics, a.rateLimit)
	var wg sync.WaitGroup

	// Workers for sending metrics
	for i := 0; i < a.rateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.worker(jobs)
		}()
	}

	pollTicker := time.NewTicker(a.pollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(a.reportInterval)
	defer reportTicker.Stop()

	// Run collection and reporting loop
	func() {
		for {
			select {
			case <-ctx.Done():
				a.log.Info("agent received cancellation, stopping...")
				return
			case <-pollTicker.C:
				a.MetricsCollect()
				a.MetricsCollectGopsutil()
			case <-reportTicker.C:
				a.MetricsSend(jobs)
			}
		}
	}()

	// Trigger one final metrics send of whatever is currently collected
	a.log.Info("sending final batch of metrics before shutdown...")
	a.MetricsSend(jobs)

	// Close jobs channel to signal workers to drain and exit
	close(jobs)

	// Wait for all workers to finish sending remaining metrics
	wg.Wait()
	a.log.Info("all agent workers finished, shutdown complete")
}
