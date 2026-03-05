package main

import (
	"collector/internal/agent"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	addr := flag.String("a", "localhost:8080", "адрес HTTP-сервера")
	reportInterval := flag.Int("r", 10, "частота отправки метрик")
	pollInterval := flag.Int("p", 2, "частота опроса метрик")
	flag.Parse()


	//если подали "непонятные" аргументы, то сообщаем об этом и завершаем программу
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "неизвестные аргументы: %v\n", flag.Args())
		os.Exit(1)
	}

	a := agent.NewAgent(
		//передаем только URL, сами решаем, что это будет http
		"http://"+*addr,
		time.Duration(*pollInterval)*time.Second,
		time.Duration(*reportInterval)*time.Second,
	)
	a.Run()
}
