package main

import (
	"collector/internal/handler"
	"collector/internal/middleware"
	"collector/internal/repository"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	addr := flag.String("a", "localhost:8080", "адрес HTTP-сервера")
	storeInterval := flag.Int("i", 300, "интервал сохранения метрик на диск (секунды, 0 — синхронно)")
	fileStoragePath := flag.String("f", "/tmp/metrics-storage.json", "путь к файлу хранилища метрик")
	restore := flag.Bool("r", true, "загружать ранее сохранённые метрики при старте")
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "неизвестные аргументы: %v\n", flag.Args())
		os.Exit(1)
	}

	// Переменные окружения имеют приоритет над флагами
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*storeInterval = n
		}
	}
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		*fileStoragePath = v
	}
	if v := os.Getenv("RESTORE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			*restore = b
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

	// Хранилище: файловое или только в памяти
	var storage repository.MemRepository

	if *fileStoragePath != "" {
		fileStorage := repository.NewFileBackedStorage(*fileStoragePath)

		if *restore {
			if err := fileStorage.Load(); err != nil {
				log.Warn("не удалось загрузить метрики из файла", zap.String("path", *fileStoragePath), zap.Error(err))
			} else {
				log.Info("метрики загружены из файла", zap.String("path", *fileStoragePath))
			}
		}

		// Периодическое сохранениt
		if *storeInterval > 0 {
			go func() {
				ticker := time.NewTicker(time.Duration(*storeInterval) * time.Second)
				defer ticker.Stop()
				for range ticker.C {
					if err := fileStorage.Save(); err != nil {
						log.Error("ошибка сохранения метрик в файл", zap.String("path", *fileStoragePath), zap.Error(err))
					} else {
						log.Info("метрики сохранены в файл", zap.String("path", *fileStoragePath))
					}
				}
			}()
		}

		storage = fileStorage
	} else {
		storage = repository.NewStructMem()
	}

	e := echo.New()
	// Pre-middleware: убираем trailing slash до роутинга,
	// чтобы /update/ и /update обрабатывались одинаково
	e.Pre(echomiddleware.RemoveTrailingSlash())
	// логгер должен быть реализован через middleware
	e.Use(middleware.RequestLogger(log))
	e.Use(middleware.GzipMiddleware(log))

	metricsHandler := handler.NewMetricsHandler(storage)
	metricsHandler.RegisterRoutes(e)
	if err := e.Start(*addr); err != nil {
		log.Fatal("сервер завершил работу с ошибкой", zap.Error(err))
	}
}
