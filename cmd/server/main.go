package main

import (
	"collector/internal/handler"
	"collector/internal/repository"
	"flag"
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
)

func main() {
	addr := flag.String("a", "localhost:8080", "адрес HTTP-сервера")
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "неизвестные аргументы: %v\n", flag.Args())
		os.Exit(1)
	}

	// Переменная окружения имеет приоритет над флагом
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}

	storage := repository.NewStructMem()

	e := echo.New()
	metricsHandler := handler.NewMetricsHandler(storage)
	metricsHandler.RegisterRoutes(e)
	if err := e.Start(*addr); err != nil {
		e.Logger.Fatal(err)
	}
}
