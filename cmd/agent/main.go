package main

import (
	"collector/internal/agent"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {

	//информация из флагов при запуске
	addr := flag.String("a", "localhost:8080", "адрес HTTP-сервера")
	reportInterval := flag.Int("r", 10, "частота отправки метрик")
	pollInterval := flag.Int("p", 2, "частота опроса метрик")
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

	a := agent.NewAgent(
		//передаем только URL, сами решаем, что это будет http
		"http://"+*addr,
		time.Duration(*pollInterval)*time.Second,
		time.Duration(*reportInterval)*time.Second,
	)
	a.Run()
}
