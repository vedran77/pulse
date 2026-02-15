package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vedran77/pulse/internal/config"
)

func main() {
	cfg := config.Load()
	fmt.Printf("Starting server on port %s\n", cfg.ServerPort)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
