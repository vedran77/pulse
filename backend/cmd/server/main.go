package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vedran77/pulse/internal/config"
	"github.com/vedran77/pulse/internal/database"
	postgresrepo "github.com/vedran77/pulse/internal/repository/postgres"
	"github.com/vedran77/pulse/internal/service"
	"github.com/vedran77/pulse/internal/transport/http/handlers"
	"github.com/vedran77/pulse/internal/transport/http/middleware"
)

func main() {
	cfg := config.Load()

	// Database
	pool, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	// Repositories
	userRepo := postgresrepo.NewUserRepo(pool)

	// Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// Start server with CORS
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, middleware.CORS(mux)))
}
