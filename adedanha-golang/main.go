package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"adedanha-golang/database"
	"adedanha-golang/handlers"

	"github.com/gorilla/mux"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Initialize database
	dbPath := "./data/adedanha.db"
	database.InitDB(dbPath)

	// Initialize WebSocket hub
	handlers.GameHub = handlers.NewHub()
	go handlers.GameHub.Run()

	// Start periodic cleanup job
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			database.Cleanup()
			log.Println("Database cleanup completed")
		}
	}()

	// Setup router
	r := mux.NewRouter()
	r.Use(corsMiddleware)

	// User routes
	r.HandleFunc("/api/users/login", handlers.LoginUser).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/users", handlers.CreateUser).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/users/{id}", handlers.UpdateUser).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/users/{id}", handlers.GetUser).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/users/{id}/avatar", handlers.UploadAvatar).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/users/{id}/leave-all", handlers.LeaveAllMatches).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/users/{id}/active-match", handlers.GetActiveMatch).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/online-users", handlers.GetOnlineUsers).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/available-players", handlers.GetAvailablePlayers).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/invites/{userId}", handlers.GetPendingInvites).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/invites/{inviteId}/respond", handlers.RespondInvite).Methods("POST", "OPTIONS")

	// Match routes
	r.HandleFunc("/api/matches", handlers.CreateMatch).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/open", handlers.ListOpenMatches).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/join", handlers.JoinMatch).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/leave", handlers.LeaveMatch).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/request-join", handlers.RequestJoinMatch).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/join-requests", handlers.GetJoinRequests).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/join-requests/{requestId}/respond", handlers.RespondJoinRequest).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/invite", handlers.InvitePlayer).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/{id}", handlers.GetMatch).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/state", handlers.GetMatchState).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/end", handlers.EndMatch).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/rounds/start", handlers.StartRound).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/rounds/{roundId}/answers", handlers.SubmitAnswers).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/rounds/{roundId}/scores", handlers.UpdateScores).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/rounds/{roundId}/results", handlers.GetRoundResults).Methods("GET", "OPTIONS")

	// WebSocket routes
	r.HandleFunc("/ws/{matchId}/{userId}", handlers.HandleWebSocket)
	r.HandleFunc("/ws/presence/{userId}", handlers.HandlePresenceWebSocket)

	// Create server with timeouts
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Println("Adedanha server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	database.DB.Close()
	log.Println("Server stopped")
}
