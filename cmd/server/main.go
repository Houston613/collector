package main

import (
	"collector/internal/audit"
	"collector/internal/handler"
	"collector/internal/middleware"
	"collector/internal/repository"
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	addr := flag.String("a", "localhost:8080", "адрес HTTP-сервера")
	storeInterval := flag.Int("i", 300, "интервал сохранения метрик на диск (секунды, 0 — синхронно)")
	fileStoragePath := flag.String("f", "/tmp/metrics-storage.json", "путь к файлу хранилища метрик")
	restore := flag.Bool("r", true, "загружать ранее сохранённые метрики при старте")
	dbDSN := flag.String("d", "", "строка подключения к базе данных")
	key := flag.String("k", "", "ключ для подписи данных")
	auditFilePath := flag.String("audit-file", "", "путь к файлу аудита. если пусто — аудит в файл отключён")
	auditURL := flag.String("audit-url", "", "URL аудит-сервера. если пусто — аудит по HTTP отключён")
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
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		*dbDSN = v
	}
	if v := os.Getenv("KEY"); v != "" {
		*key = v
	}
	if v := os.Getenv("AUDIT_FILE"); v != "" {
		*auditFilePath = v
	}
	if v := os.Getenv("AUDIT_URL"); v != "" {
		*auditURL = v
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

	// Хранилище: БД, файловое или только в памяти
	var storage repository.MemRepository
	ctx := context.Background()

	if *dbDSN != "" {
		dbStorage, err := repository.NewDBStorage(ctx, *dbDSN)
		if err != nil {
			log.Fatal("не удалось инициализировать БД", zap.Error(err))
		}
		if err := dbStorage.Bootstrap("migrations"); err != nil {
			log.Fatal("не удалось выполнить миграции", zap.Error(err))
		}
		storage = dbStorage
		log.Info("используется хранилище в БД")
	} else if *fileStoragePath != "" {
		fileStorage := repository.NewFileBackedStorage(*fileStoragePath, *storeInterval == 0, log)

		if *restore {
			if err := fileStorage.Load(); err != nil {
				log.Warn("не удалось загрузить метрики из файла", zap.String("path", *fileStoragePath), zap.Error(err))
			} else {
				log.Info("метрики загружены из файла", zap.String("path", *fileStoragePath))
			}
		}

		// Периодическое сохранение
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
		log.Info("используется файловое хранилище", zap.String("path", *fileStoragePath))
	} else {
		storage = repository.NewStructMem()
		log.Info("используется хранилище в памяти")
	}

	// Собираем аудитор: регистрируем наблюдателей по наличию конфига
	var observers []audit.Observer
	if *auditFilePath != "" {
		fo, err := audit.NewFileObserver(*auditFilePath)
		if err != nil {
			log.Fatal("не удалось открыть файл аудита", zap.String("path", *auditFilePath), zap.Error(err))
		}
		observers = append(observers, fo)
		log.Info("аудит в файл включён", zap.String("path", *auditFilePath))
	}
	if *auditURL != "" {
		ho := audit.NewHTTPObserver(*auditURL)
		observers = append(observers, ho)
		log.Info("аудит по HTTP включён", zap.String("url", *auditURL))
	}

	var auditor *audit.Notifier
	if len(observers) > 0 {
		auditor = audit.NewNotifier(observers...)
		defer func() {
			if err := auditor.Close(); err != nil {
				log.Error("ошибка закрытия", zap.Error(err))
			}
		}()
	}

	e := echo.New()
	// логгер должен быть реализован через middleware
	e.Use(middleware.RequestLogger(log))
	// сначала сжимать, а потом подписывать
	e.Use(middleware.GzipMiddleware(log))
	e.Use(middleware.SignatureMiddleware(*key, log))

	//возможно стоит передавать конфиг вместо строки подключения, но пока так
	metricsHandler := handler.NewMetricsHandler(storage, *dbDSN, auditor)
	metricsHandler.RegisterRoutes(e)
	if err := e.Start(*addr); err != nil {
		log.Fatal("сервер завершил работу с ошибкой", zap.Error(err))
	}
}
