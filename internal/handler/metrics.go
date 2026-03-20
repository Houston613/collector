package handler

import (
	"collector/internal/repository"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type MetricsHandler struct {
	repo repository.MemRepository
}

func NewMetricsHandler(repo repository.MemRepository) *MetricsHandler {
	return &MetricsHandler{repo: repo}
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
		h.repo.UpdateGauge(metricName, val)

	case "counter":
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return c.String(http.StatusBadRequest, "Невереное значение для метрики Counter")
		}
		h.repo.UpdateCounter(metricName, val)

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
			val, ok := repo.GetGauge(metricName)
			if !ok {
				return c.String(http.StatusNotFound, "Метрика не найдена")
			}
			return c.String(http.StatusOK, strconv.FormatFloat(val, 'f', -1, 64))

		case "counter":
			val, ok := repo.GetCounter(metricName)
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

		for name, val := range repo.GetAllGauges() {
			fmt.Fprintf(&sb, "<li>gauge/%s = %s</li>", name, strconv.FormatFloat(val, 'f', -1, 64))
		}
		for name, val := range repo.GetAllCounters() {
			fmt.Fprintf(&sb, "<li>counter/%s = %d</li>", name, val)
		}
		sb.WriteString("</ul></body></html>")
		return c.HTML(http.StatusOK, sb.String())
	}
}

func (h *MetricsHandler) RegisterRoutes(e *echo.Echo) {
	e.POST("/update/:type/:name/:value", h.UpdateMetrics)
	e.GET("/value/:type/:name", GetMetric(h.repo))
	e.GET("/", ListMetrics(h.repo))
}
