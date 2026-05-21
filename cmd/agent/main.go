package main

import (
	"collector/internal/agent"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {

	//информация из флагов при запуске
	addr := flag.String("a", "localhost:8080", "адрес HTTP-сервера")
	reportInterval := flag.Int("r", 10, "частота отправки метрик")
	pollInterval := flag.Int("p", 2, "частота опроса метрик")
	key := flag.String("k", "", "ключ для подписи данных")
	rateLimit := flag.Int("l", 3, "ограничение количества одновременно исходящих запросов на сервер")
	flag.Parse()

	//если подали "непонятные" аргументы, то сообщаем об этом и завершаем программу
	//если не подали, пофиг
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "неизвестные аргументы: %v\n", flag.Args())
		os.Exit(1)
	}

	// Переменные окружения имеют приоритет над флагами
	//поэтому если они есть - перезатрут стнадартные значения
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}
	if envReport := os.Getenv("REPORT_INTERVAL"); envReport != "" {
		if v, err := strconv.Atoi(envReport); err == nil {
			*reportInterval = v
		}
	}
	if envPoll := os.Getenv("POLL_INTERVAL"); envPoll != "" {
		if v, err := strconv.Atoi(envPoll); err == nil {
			*pollInterval = v
		}
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		*key = envKey
	}
	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		if v, err := strconv.Atoi(envRateLimit); err == nil {
			*rateLimit = v
		}
	}

	// Собираем логгер
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "timestamp"
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	log := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	))
	defer log.Sync()

	a := agent.NewAgent(
		//передаем только URL, сами решаем, что это будет http
		"http://"+*addr,
		time.Duration(*pollInterval)*time.Second,
		time.Duration(*reportInterval)*time.Second,
		*key,
		*rateLimit,
		log,
	)
	a.Run()
}
