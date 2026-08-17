package agent

import (
	models "collector/internal/model"
	"context"
	"crypto/rsa"
	"fmt"
	"maps"
	"math/rand"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

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
	pollInterval    time.Duration
	reportInterval  time.Duration
	rateLimit       int
	sender          MetricsSender
	log             *zap.Logger
}

// SetGrpcAddress configures the agent to use gRPC for sending metrics.
func (a *Agent) SetGrpcAddress(grpcAddr string) {
	hostIP := getLocalIP(grpcAddr)
	a.sender = NewGRPCSender(grpcAddr, hostIP, a.log)
}

// SetSender sets a custom MetricsSender implementation (useful for tests or custom transports).
func (a *Agent) SetSender(sender MetricsSender) {
	a.sender = sender
}

// NewAgent creates and configures a new metrics Agent using HTTP by default.
func NewAgent(addr string, pollInterval, reportInterval time.Duration, key string, cryptoKey *rsa.PublicKey, rateLimit int, log *zap.Logger) *Agent {
	hostIP := getLocalIP(addr)
	sender := NewHTTPSender(addr, key, cryptoKey, hostIP, log)

	return &Agent{
		gaugesMetrics:   make(map[string]float64),
		countersMetrics: make(map[string]int64),
		pollInterval:    pollInterval,
		reportInterval:  reportInterval,
		rateLimit:       rateLimit,
		sender:          sender,
		log:             log,
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
		if err := a.sender.Send(context.Background(), metrics); err != nil {
			a.log.Error("error sending metrics batch", zap.Error(err))
		}
	}
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
	if a.sender != nil {
		if err := a.sender.Close(); err != nil {
			a.log.Error("failed to close metrics sender on shutdown", zap.Error(err))
		}
	}
	a.log.Info("all agent workers finished, shutdown complete")
}

func getLocalIP(serverAddr string) string {
	target := strings.TrimPrefix(serverAddr, "http://")
	target = strings.TrimPrefix(target, "https://")
	if target == "" {
		return "127.0.0.1"
	}

	host, port, err := net.SplitHostPort(target)
	if err != nil {
		host = target
		port = "80"
	}

	conn, err := net.Dial("udp", net.JoinHostPort(host, port))
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || localAddr.IP == nil {
		return "127.0.0.1"
	}

	return localAddr.IP.String()
}
