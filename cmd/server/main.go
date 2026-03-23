package main

import (
	"collector/internal/handler"
	"collector/internal/middleware"
	"collector/internal/repository"
	"flag"
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	storage := repository.NewStructMem()

	e := echo.New()
	// Pre-middleware: убираем trailing slash до роутинга,
	// чтобы /update/ и /update обрабатывались одинаково
	e.Pre(echomiddleware.RemoveTrailingSlash())
	// логгер должен быть реализован через middleware
	e.Use(middleware.RequestLogger(log))

	metricsHandler := handler.NewMetricsHandler(storage)
	metricsHandler.RegisterRoutes(e)
	if err := e.Start(*addr); err != nil {
		log.Fatal("сервер завершил работу с ошибкой", zap.Error(err))
	}
}
