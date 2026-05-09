package main

import (
	"log"
	"math/rand"
	"net/http"
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
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Initialize database
	dbPath := "./data/adedanha.db"
	database.InitDB(dbPath)

	// Initialize WebSocket hub
	handlers.GameHub = handlers.NewHub()
	go handlers.GameHub.Run()

	// Setup router
	r := mux.NewRouter()

	// Apply CORS middleware
	r.Use(corsMiddleware)

	// User routes
	r.HandleFunc("/api/users/login", handlers.LoginUser).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/users", handlers.CreateUser).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/users/{id}", handlers.UpdateUser).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/users/{id}", handlers.GetUser).Methods("GET", "OPTIONS")
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

	// Start server
	log.Println("Adedanha server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
