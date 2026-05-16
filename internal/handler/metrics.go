package handler

import (
	models "collector/internal/model"
	"collector/internal/repository"
	"context"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type MetricsHandler struct {
	repo repository.MemRepository
}

func NewMetricsHandler(repo repository.MemRepository, dbDSN string) *MetricsHandler {
	return &MetricsHandler{
		repo: repo,
	}
}

func (h *MetricsHandler) UpdateMetrics(c echo.Context) error {
	ct := c.Request().Header.Get("Content-Type")
	if ct != "" {
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil || mt != "text/plain" {
			return c.String(http.StatusUnsupportedMediaType, "Content-Type должен быть text/plain")
		}
	}

	metricType := c.Param("type")
	metricName := c.Param("name")
	metricValue := c.Param("value")

	switch metricType {
	case "gauge":
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return c.String(http.StatusBadRequest, "Невереное значение для метрики Gauge")
		}
		if err := h.repo.UpdateGauge(c.Request().Context(), metricName, val); err != nil {
			return c.String(http.StatusInternalServerError, "Ошибка обновления метрики")
		}

	case "counter":
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return c.String(http.StatusBadRequest, "Невереное значение для метрики Counter")
		}
		if err := h.repo.UpdateCounter(c.Request().Context(), metricName, val); err != nil {
			return c.String(http.StatusInternalServerError, "Ошибка обновления метрики")
		}

	default:
		return c.String(http.StatusBadRequest, "Неверный тип метрики")
	}

	return c.NoContent(http.StatusOK)
}

func GetMetric(repo repository.MemRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		metricType := c.Param("type")
		metricName := c.Param("name")

		switch metricType {
		case "gauge":
			val, ok, err := repo.GetGauge(c.Request().Context(), metricName)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Ошибка получения метрики")
			}
			if !ok {
				return c.String(http.StatusNotFound, "Метрика не найдена")
			}
			return c.String(http.StatusOK, strconv.FormatFloat(val, 'f', -1, 64))

		case "counter":
			val, ok, err := repo.GetCounter(c.Request().Context(), metricName)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Ошибка получения метрики")
			}
			if !ok {
				return c.String(http.StatusNotFound, "Метрика не найдена")
			}
			return c.String(http.StatusOK, strconv.FormatInt(val, 10))

		default:
			return c.String(http.StatusNotFound, "Неизвестный тип метрики")
		}
	}
}

func ListMetrics(repo repository.MemRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		var sb strings.Builder
		sb.WriteString("<html><body><h1>Метрики</h1><ul>")

		gauges, err := repo.GetAllGauges(c.Request().Context())
		if err != nil {
			return c.String(http.StatusInternalServerError, "Ошибка получения списка метрик")
		}
		for name, val := range gauges {
			fmt.Fprintf(&sb, "<li>gauge/%s = %s</li>", name, strconv.FormatFloat(val, 'f', -1, 64))
		}

		counters, err := repo.GetAllCounters(c.Request().Context())
		if err != nil {
			return c.String(http.StatusInternalServerError, "Ошибка получения списка метрик")
		}
		for name, val := range counters {
			fmt.Fprintf(&sb, "<li>counter/%s = %d</li>", name, val)
		}
		sb.WriteString("</ul></body></html>")
		return c.HTML(http.StatusOK, sb.String())
	}
}

func (h *MetricsHandler) Ping(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 1*time.Second)
	defer cancel()

	if err := h.repo.Ping(ctx); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to ping storage: %v", err))
	}

	return c.NoContent(http.StatusOK)
}

func (h *MetricsHandler) UpdateMetricJSON(c echo.Context) error {
	var m models.Metrics
	if err := c.Bind(&m); err != nil {
		return c.JSON(http.StatusBadRequest, "не удалось распарсить JSON")
	}

	switch m.MType {
	case models.Gauge:
		if m.Value == nil {
			return c.JSON(http.StatusBadRequest, "value обязательно для gauge")
		}
		if err := h.repo.UpdateGauge(c.Request().Context(), m.ID, *m.Value); err != nil {
			return c.JSON(http.StatusInternalServerError, "ошибка обновления метрики")
		}
	case models.Counter:
		if m.Delta == nil {
			return c.JSON(http.StatusBadRequest, "delta обязательно для counter")
		}
		if err := h.repo.UpdateCounter(c.Request().Context(), m.ID, *m.Delta); err != nil {
			return c.JSON(http.StatusInternalServerError, "ошибка обновления метрики")
		}
	default:
		return c.JSON(http.StatusBadRequest, "неверный тип метрики")
	}

	return c.JSON(http.StatusOK, m)
}

func (h *MetricsHandler) UpdatesMetricsJSON(c echo.Context) error {
	var metrics []models.Metrics
	if err := c.Bind(&metrics); err != nil {
		return c.JSON(http.StatusBadRequest, "не удалось распарсить JSON")
	}

	if err := h.repo.UpdateMetrics(c.Request().Context(), metrics); err != nil {
		return c.JSON(http.StatusInternalServerError, "ошибка обновления метрик")
	}

	return c.NoContent(http.StatusOK)
}

// GetMetricJSON принимает JSON с ID и MType, возвращает JSON с заполнением значением.
func (h *MetricsHandler) GetMetricJSON(c echo.Context) error {
	var m models.Metrics
	if err := c.Bind(&m); err != nil {
		return c.JSON(http.StatusBadRequest, "не удалось прочитать JSON")
	}
	switch m.MType {
	case models.Gauge:
		val, ok, err := h.repo.GetGauge(c.Request().Context(), m.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "ошибка получения метрики")
		}
		if !ok {
			return c.JSON(http.StatusNotFound, "метрика не найдена")
		}
		m.Value = &val
	case models.Counter:
		val, ok, err := h.repo.GetCounter(c.Request().Context(), m.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "ошибка получения метрики")
		}
		if !ok {
			return c.JSON(http.StatusNotFound, "метрика не найдена")
		}
		m.Delta = &val
	default:
		return c.JSON(http.StatusNotFound, "неизвестный тип метрики")
	}

	return c.JSON(http.StatusOK, m)
}

func (h *MetricsHandler) RegisterRoutes(e *echo.Echo) {
	e.POST("/update/:type/:name/:value", h.UpdateMetrics)
	e.POST("/update", h.UpdateMetricJSON)
	e.POST("/updates/", h.UpdatesMetricsJSON)
	e.POST("/value", h.GetMetricJSON)
	e.GET("/value/:type/:name", GetMetric(h.repo))
	e.GET("/", ListMetrics(h.repo))
	e.GET("/ping", h.Ping)
}
