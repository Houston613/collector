package main

import (
	"collector/internal/handler"
	"collector/internal/repository"

	"github.com/labstack/echo/v4"
)

func main() {
	storage := repository.NewStructMem()

	e := echo.New()

	metricsHandler := handler.NewMetricsHandler(storage)
	metricsHandler.RegisterRoutes(e)
	if err := e.Start(":8080"); err != nil {
		e.Logger.Fatal(err)
	}
}
