package main

import (
	"collector/internal/handler"
	"collector/internal/repository"
	"net/http"
)

func main() {

	
	storage := repository.NewStructMem()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /update/{type}/{name}/{value}", handler.UpdateMetrics(storage))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}