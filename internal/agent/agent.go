package agent

import (
	"maps"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	DefaultServerAddress    = "localhost:8080"
	DefaultPollInterval   = 2 * time.Second
	DefaultReportInterval = 10 * time.Second
)

type Agent struct {
	mu              sync.RWMutex
	gaugesMetrics   map[string]float64
	countersMetrics map[string]int64
	addr            string
	pollInterval    time.Duration
	reportInterval  time.Duration
	client          *http.Client
}

func NewAgent(addr string, pollInterval, reportInterval time.Duration) *Agent {
	return &Agent{
		gaugesMetrics:   make(map[string]float64),
		countersMetrics: make(map[string]int64),
		addr:            addr,
		pollInterval:    pollInterval,
		reportInterval:  reportInterval,
		client:          &http.Client{},
	}
}


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
	//cлучайное значение
	a.gaugesMetrics["RandomValue"] = rand.Float64()
	a.countersMetrics["PollCount"]++
}


func (a *Agent) MetricsSend() {
	//сналчала копируем данные в локальные переменные, чтобы не держать блокировку на время отправки данных
	a.mu.RLock()
	gauges := make(map[string]float64, len(a.gaugesMetrics))
	maps.Copy(gauges, a.gaugesMetrics)
	counters := make(map[string]int64, len(a.countersMetrics))
	maps.Copy(counters, a.countersMetrics)
	//отпускаем блокировку
	a.mu.RUnlock()
	//отправляем данные
	for name, value := range gauges {
		url := fmt.Sprintf(
			"%s/update/gauge/%s/%s",
			a.addr, name,
			strconv.FormatFloat(value, 'g', -1, 64),
		)
		if err := a.sendRequest(url); err != nil {
			log.Printf("error sending gauge %s: %v", name, err)
		}
	}

	for name, value := range counters {
		url := fmt.Sprintf("%s/update/counter/%s/%d", a.addr, name, value)
		if err := a.sendRequest(url); err != nil {
			log.Printf("error sending counter %s: %v", name, err)
		}
	}
}

func (a *Agent) sendRequest(url string) error {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	return nil
}


func (a *Agent) Run() {
	// Сбор метрик в отдельной горутине
	//значит будет конкурентный доступ к данным, поэтому используем мьютекс для защиты данных
	go func() {
		for {
			a.MetricsCollect()
			time.Sleep(a.pollInterval)
		}
	}()

	for {
		time.Sleep(a.reportInterval)
		a.MetricsSend()
	}
}
