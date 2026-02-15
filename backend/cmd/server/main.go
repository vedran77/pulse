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
	workspaceRepo := postgresrepo.NewWorkspaceRepo(pool)
	channelRepo := postgresrepo.NewChannelRepo(pool)
	messageRepo := postgresrepo.NewMessageRepo(pool)

	// Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	workspaceService := service.NewWorkspaceService(workspaceRepo, userRepo)
	channelService := service.NewChannelService(channelRepo, workspaceRepo)
	messageService := service.NewMessageService(messageRepo, channelRepo, workspaceRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	workspaceHandler := handlers.NewWorkspaceHandler(workspaceService)
	channelHandler := handlers.NewChannelHandler(channelService)
	messageHandler := handlers.NewMessageHandler(messageService)

	// Auth middleware
	auth := middleware.Auth(cfg.JWTSecret)

	// Routes
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// Protected - Workspaces
	mux.Handle("POST /api/v1/workspaces", auth(http.HandlerFunc(workspaceHandler.Create)))
	mux.Handle("GET /api/v1/workspaces", auth(http.HandlerFunc(workspaceHandler.List)))
	mux.Handle("GET /api/v1/workspaces/{id}", auth(http.HandlerFunc(workspaceHandler.Get)))
	mux.Handle("PATCH /api/v1/workspaces/{id}", auth(http.HandlerFunc(workspaceHandler.Update)))
	mux.Handle("DELETE /api/v1/workspaces/{id}", auth(http.HandlerFunc(workspaceHandler.Delete)))

	// Protected - Workspace Members
	mux.Handle("POST /api/v1/workspaces/{id}/members", auth(http.HandlerFunc(workspaceHandler.AddMember)))
	mux.Handle("DELETE /api/v1/workspaces/{id}/members/{uid}", auth(http.HandlerFunc(workspaceHandler.RemoveMember)))
	mux.Handle("GET /api/v1/workspaces/{id}/members", auth(http.HandlerFunc(workspaceHandler.ListMembers)))

	// Protected - Channels
	mux.Handle("POST /api/v1/workspaces/{wid}/channels", auth(http.HandlerFunc(channelHandler.Create)))
	mux.Handle("GET /api/v1/workspaces/{wid}/channels", auth(http.HandlerFunc(channelHandler.List)))
	mux.Handle("GET /api/v1/channels/{id}", auth(http.HandlerFunc(channelHandler.Get)))
	mux.Handle("PATCH /api/v1/channels/{id}", auth(http.HandlerFunc(channelHandler.Update)))
	mux.Handle("DELETE /api/v1/channels/{id}", auth(http.HandlerFunc(channelHandler.Archive)))

	// Protected - Channel Members
	mux.Handle("POST /api/v1/channels/{id}/join", auth(http.HandlerFunc(channelHandler.Join)))
	mux.Handle("POST /api/v1/channels/{id}/members", auth(http.HandlerFunc(channelHandler.AddMember)))
	mux.Handle("DELETE /api/v1/channels/{id}/members/{uid}", auth(http.HandlerFunc(channelHandler.RemoveMember)))
	mux.Handle("GET /api/v1/channels/{id}/members", auth(http.HandlerFunc(channelHandler.ListMembers)))

	// Protected - Messages
	mux.Handle("POST /api/v1/channels/{id}/messages", auth(http.HandlerFunc(messageHandler.Send)))
	mux.Handle("GET /api/v1/channels/{id}/messages", auth(http.HandlerFunc(messageHandler.List)))
	mux.Handle("PATCH /api/v1/messages/{id}", auth(http.HandlerFunc(messageHandler.Edit)))
	mux.Handle("DELETE /api/v1/messages/{id}", auth(http.HandlerFunc(messageHandler.Delete)))

	// Start server with CORS
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, middleware.CORS(mux)))
}
