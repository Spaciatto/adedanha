package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"adedanha-golang/database"
	"adedanha-golang/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" {
		http.Error(w, `{"error":"Name and email are required"}`, http.StatusBadRequest)
		return
	}

	user := models.User{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	_, err := database.DB.Exec(
		"INSERT INTO users (id, name, email, created_at) VALUES (?, ?, ?, ?)",
		user.ID, user.Name, user.Email, user.CreatedAt,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to create user. Email may already exist."}`, http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" {
		http.Error(w, `{"error":"Name and email are required"}`, http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(
		"UPDATE users SET name = ?, email = ? WHERE id = ?",
		req.Name, req.Email, userID,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to update user"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	var user models.User
	err = database.DB.QueryRow("SELECT id, name, email, created_at FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"Failed to retrieve updated user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var user models.User
	err := database.DB.QueryRow("SELECT id, name, email, created_at FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetOnlineUsers returns users that currently have an active WebSocket connection
func GetOnlineUsers(w http.ResponseWriter, r *http.Request) {
	onlineUserIDs := GameHub.GetOnlineUserIDs()

	if len(onlineUserIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]models.OnlineUser{})
		return
	}

	// Build query with placeholders
	placeholders := ""
	args := make([]interface{}, len(onlineUserIDs))
	for i, id := range onlineUserIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	rows, err := database.DB.Query(
		"SELECT id, name FROM users WHERE id IN ("+placeholders+")", args...,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to get online users"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []models.OnlineUser{}
	for rows.Next() {
		var u models.OnlineUser
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			continue
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// LoginUser handles login by email - returns existing user or error
func LoginUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, `{"error":"Email is required"}`, http.StatusBadRequest)
		return
	}

	var user models.User
	err := database.DB.QueryRow("SELECT id, name, email, created_at FROM users WHERE email = ?", req.Email).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetAvailablePlayers returns online users that are NOT in any active match
func GetAvailablePlayers(w http.ResponseWriter, r *http.Request) {
	onlineUserIDs := GameHub.GetOnlineUserIDs()

	if len(onlineUserIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]models.OnlineUser{})
		return
	}

	// Build query with placeholders
	placeholders := ""
	args := make([]interface{}, len(onlineUserIDs))
	for i, id := range onlineUserIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	// Get online users who are NOT in any active match
	rows, err := database.DB.Query(`
		SELECT id, name FROM users 
		WHERE id IN (`+placeholders+`)
		AND id NOT IN (
			SELECT mp.user_id FROM match_players mp
			JOIN matches m ON mp.match_id = m.id
			WHERE mp.active = 1 AND m.status != 'finished'
		)`, args...,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to get available players"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []models.OnlineUser{}
	for rows.Next() {
		var u models.OnlineUser
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			continue
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetPendingInvites returns pending invites for a user (polling fallback for mobile)
func GetPendingInvites(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	rows, err := database.DB.Query(
		"SELECT id, match_id, match_name, inviter_name, status FROM invites WHERE target_user_id = ? AND status = 'pending' ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to get invites"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Invite struct {
		ID          string `json:"id"`
		MatchID     string `json:"match_id"`
		MatchName   string `json:"match_name"`
		InviterName string `json:"inviter_name"`
		Status      string `json:"status"`
	}

	invites := []Invite{}
	for rows.Next() {
		var inv Invite
		if err := rows.Scan(&inv.ID, &inv.MatchID, &inv.MatchName, &inv.InviterName, &inv.Status); err != nil {
			continue
		}
		invites = append(invites, inv)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invites)
}

// RespondInvite handles accept/reject of an invite
func RespondInvite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	inviteID := vars["inviteId"]

	var reqBody struct {
		UserID   string `json:"user_id"`
		Accepted bool   `json:"accepted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Get invite
	var matchID, targetUserID string
	err := database.DB.QueryRow(
		"SELECT match_id, target_user_id FROM invites WHERE id = ? AND status = 'pending'",
		inviteID,
	).Scan(&matchID, &targetUserID)
	if err != nil {
		http.Error(w, `{"error":"Invite not found or already processed"}`, http.StatusNotFound)
		return
	}

	if reqBody.UserID != targetUserID {
		http.Error(w, `{"error":"This invite is not for you"}`, http.StatusForbidden)
		return
	}

	if reqBody.Accepted {
		database.DB.Exec("UPDATE invites SET status = 'accepted' WHERE id = ?", inviteID)

		// Add player to match
		var userName string
		database.DB.QueryRow("SELECT name FROM users WHERE id = ?", reqBody.UserID).Scan(&userName)

		database.DB.Exec(
			"INSERT OR IGNORE INTO match_players (match_id, user_id, active, joined_at) VALUES (?, ?, 1, ?)",
			matchID, reqBody.UserID, time.Now(),
		)

		// Broadcast player_joined
		BroadcastToMatch(matchID, models.WSMessage{
			Type:     "player_joined",
			UserID:   reqBody.UserID,
			UserName: userName,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Invite accepted", "match_id": matchID})
	} else {
		database.DB.Exec("UPDATE invites SET status = 'rejected' WHERE id = ?", inviteID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Invite rejected"})
	}
}

// LeaveAllMatches removes the user from all active matches (used on logout)
func LeaveAllMatches(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	// Get all active matches where user is a non-creator active player
	rows, err := database.DB.Query(`
		SELECT mp.match_id FROM match_players mp
		JOIN matches m ON mp.match_id = m.id
		WHERE mp.user_id = ? AND mp.active = 1 AND m.status != 'finished' AND m.creator_id != ?
	`, userID, userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to leave matches"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	matchIDs := []string{}
	for rows.Next() {
		var matchID string
		if err := rows.Scan(&matchID); err != nil {
			continue
		}
		matchIDs = append(matchIDs, matchID)
	}

	// Mark player as inactive in all those matches
	for _, matchID := range matchIDs {
		database.DB.Exec("UPDATE match_players SET active = 0 WHERE match_id = ? AND user_id = ?", matchID, userID)

		var userName string
		database.DB.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&userName)

		BroadcastToMatch(matchID, models.WSMessage{
			Type:     "player_left",
			UserID:   userID,
			UserName: userName,
		})
	}

	// For matches where user IS the creator and match is still 'waiting', end them
	database.DB.Exec(`
		UPDATE matches SET status = 'finished' 
		WHERE creator_id = ? AND status = 'waiting'
	`, userID)

	// Cancel pending invites for this user
	database.DB.Exec("UPDATE invites SET status = 'expired' WHERE target_user_id = ? AND status = 'pending'", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Left all matches"})
}

// GetActiveMatch returns the active match for a user (if any)
func GetActiveMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var matchID string
	err := database.DB.QueryRow(`
		SELECT mp.match_id FROM match_players mp
		JOIN matches m ON mp.match_id = m.id
		WHERE mp.user_id = ? AND mp.active = 1 AND m.status != 'finished'
		LIMIT 1
	`, userID).Scan(&matchID)

	if err != nil {
		// No active match
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"match_id": nil})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"match_id": matchID})
}
