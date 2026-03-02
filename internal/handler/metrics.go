package handler

import (
	"collector/internal/repository"
	"log"
	"mime"
	"net/http"
	"strconv"
)

func UpdateMetrics(repository repository.MemRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Println("Разрешен только метод POST")
			http.Error(w, "Разрешен только метод POST", http.StatusMethodNotAllowed)
			return
		}
		mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mt != "text/plain" {
			log.Println("Content-Type должен быть text/plain")
			http.Error(w, "Content-Type должен быть text/plain", http.StatusUnsupportedMediaType)
			return
		}

		metricType := r.PathValue("type")
		log.Printf("metric type is %s\n",metricType)
		metricName := r.PathValue("name")
		log.Println(metricName)
		metricValue := r.PathValue("value")
		log.Println(metricValue)

		switch metricType {
		case "gauge":
			val, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				log.Println("Невереное значение для метрики Gauge")
				http.Error(w, "Невереное значение для метрики Gauge", http.StatusBadRequest)
				return
			}
			repository.UpdateGauge(metricName, val)

		case "counter":
			val, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				log.Println("Невереное значение для метрики Counter")
				http.Error(w, "Неверное значение для метрики Counter", http.StatusBadRequest)
				return
			}
			repository.UpdateCounter(metricName, val)

		default:
			http.Error(w, "Неверный тип метрики", http.StatusBadRequest)
			return
		}
		log.Println("запрос корректен")
		w.WriteHeader(http.StatusOK)
	}

}
