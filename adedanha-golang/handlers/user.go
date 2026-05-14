package handlers

import (
	"log"
	"net/http"
	"time"

	"adedanha-golang/database"
	"adedanha-golang/internal/domain"
	"adedanha-golang/internal/httputil"
	"adedanha-golang/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.RespondError(w, err)
		return
	}

	if err := domain.ValidateCreateUser(req.Name, req.Email); err != nil {
		httputil.RespondError(w, err)
		return
	}

	user := domain.User{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	_, err := database.DB.Exec(
		"INSERT INTO users (id, name, email, avatar, created_at) VALUES ($1, $2, $3, '', $4)",
		user.ID, user.Name, user.Email, user.CreatedAt,
	)
	if err != nil {
		httputil.RespondError(w, domain.ErrEmailExists)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, user)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	var req models.UpdateUserRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.RespondError(w, err)
		return
	}

	if err := domain.ValidateCreateUser(req.Name, req.Email); err != nil {
		httputil.RespondError(w, err)
		return
	}

	result, err := database.DB.Exec("UPDATE users SET name = $1, email = $2 WHERE id = $3", req.Name, req.Email, userID)
	if err != nil {
		httputil.RespondError(w, domain.ErrEmailExists)
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		httputil.RespondError(w, domain.ErrUserNotFound)
		return
	}

	var user domain.User
	if err := database.DB.QueryRow("SELECT id, name, email, avatar, created_at FROM users WHERE id = $1", userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.Avatar, &user.CreatedAt); err != nil {
		httputil.RespondError(w, domain.ErrUserNotFound)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, user)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	var user domain.User
	if err := database.DB.QueryRow("SELECT id, name, email, avatar, created_at FROM users WHERE id = $1", userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.Avatar, &user.CreatedAt); err != nil {
		httputil.RespondError(w, domain.ErrUserNotFound)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, user)
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.RespondError(w, err)
		return
	}
	if req.Email == "" {
		httputil.RespondError(w, domain.ErrEmailRequired)
		return
	}

	var user domain.User
	if err := database.DB.QueryRow("SELECT id, name, email, avatar, created_at FROM users WHERE email = $1", req.Email).
		Scan(&user.ID, &user.Name, &user.Email, &user.Avatar, &user.CreatedAt); err != nil {
		httputil.RespondError(w, domain.ErrUserNotFound)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, user)
}

func GetOnlineUsers(w http.ResponseWriter, r *http.Request) {
	onlineIDs := GameHub.GetOnlineUserIDs()
	if len(onlineIDs) == 0 {
		httputil.RespondJSON(w, http.StatusOK, []domain.OnlineUser{})
		return
	}

	users, err := queryUsersByIDs(onlineIDs)
	if err != nil {
		httputil.RespondJSON(w, http.StatusOK, []domain.OnlineUser{})
		return
	}
	httputil.RespondJSON(w, http.StatusOK, users)
}

func GetAvailablePlayers(w http.ResponseWriter, r *http.Request) {
	onlineIDs := GameHub.GetOnlineUserIDs()
	if len(onlineIDs) == 0 {
		httputil.RespondJSON(w, http.StatusOK, []domain.OnlineUser{})
		return
	}

	placeholders, args := buildPlaceholders(onlineIDs)
	rows, err := database.DB.Query(`
		SELECT id, name, COALESCE(avatar, '') FROM users 
		WHERE id IN (`+placeholders+`)
		AND id NOT IN (
			SELECT mp.user_id FROM match_players mp
			JOIN matches m ON mp.match_id = m.id
			WHERE mp.active = TRUE AND m.status != '`+domain.MatchStatusFinished+`'
		)`, args...)
	if err != nil {
		httputil.RespondJSON(w, http.StatusOK, []domain.OnlineUser{})
		return
	}
	defer rows.Close()

	users := []domain.OnlineUser{}
	for rows.Next() {
		var u domain.OnlineUser
		if rows.Scan(&u.ID, &u.Name, &u.Avatar) == nil {
			users = append(users, u)
		}
	}
	httputil.RespondJSON(w, http.StatusOK, users)
}

func GetPendingInvites(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]

	rows, err := database.DB.Query(
		"SELECT id, match_id, match_name, inviter_name, status FROM invites WHERE target_user_id = $1 AND status = $2 ORDER BY created_at DESC",
		userID, domain.StatusPending,
	)
	if err != nil {
		httputil.RespondJSON(w, http.StatusOK, []domain.Invite{})
		return
	}
	defer rows.Close()

	invites := []domain.Invite{}
	for rows.Next() {
		var inv domain.Invite
		if rows.Scan(&inv.ID, &inv.MatchID, &inv.MatchName, &inv.InviterName, &inv.Status) == nil {
			invites = append(invites, inv)
		}
	}
	httputil.RespondJSON(w, http.StatusOK, invites)
}

func RespondInvite(w http.ResponseWriter, r *http.Request) {
	inviteID := mux.Vars(r)["inviteId"]

	var reqBody struct {
		UserID   string `json:"user_id"`
		Accepted bool   `json:"accepted"`
	}
	if err := httputil.DecodeJSON(r, &reqBody); err != nil {
		httputil.RespondError(w, err)
		return
	}

	var matchID, targetUserID string
	if err := database.DB.QueryRow(
		"SELECT match_id, target_user_id FROM invites WHERE id = $1 AND status = $2", inviteID, domain.StatusPending,
	).Scan(&matchID, &targetUserID); err != nil {
		httputil.RespondError(w, domain.ErrInviteNotFound)
		return
	}

	if reqBody.UserID != targetUserID {
		httputil.RespondError(w, domain.ErrNotTargetUser)
		return
	}

	if reqBody.Accepted {
		database.DB.Exec("UPDATE invites SET status = $1 WHERE id = $2", domain.StatusAccepted, inviteID)

		var userName string
		if err := database.DB.QueryRow("SELECT name FROM users WHERE id = $1", reqBody.UserID).Scan(&userName); err != nil {
			httputil.RespondError(w, domain.ErrUserNotFound)
			return
		}

		if _, err := database.DB.Exec(
			"INSERT INTO match_players (match_id, user_id, active, joined_at) VALUES ($1, $2, TRUE, $3) ON CONFLICT (match_id, user_id) DO NOTHING",
			matchID, reqBody.UserID, time.Now(),
		); err != nil {
			log.Printf("Error adding player to match: %v", err)
		}

		BroadcastToMatch(matchID, domain.WSMessage{
			Type:     "player_joined",
			UserID:   reqBody.UserID,
			UserName: userName,
		})

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"message": "Invite accepted", "match_id": matchID})
	} else {
		database.DB.Exec("UPDATE invites SET status = $1 WHERE id = $2", domain.StatusRejected, inviteID)
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"message": "Invite rejected"})
	}
}

func LeaveAllMatches(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	rows, err := database.DB.Query(`
		SELECT mp.match_id FROM match_players mp
		JOIN matches m ON mp.match_id = m.id
		WHERE mp.user_id = $1 AND mp.active = TRUE AND m.status != $2 AND m.creator_id != $1
	`, userID, domain.MatchStatusFinished)
	if err != nil {
		httputil.RespondError(w, domain.ErrNotFound)
		return
	}
	defer rows.Close()

	var matchIDs []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			matchIDs = append(matchIDs, id)
		}
	}

	var userName string
	database.DB.QueryRow("SELECT name FROM users WHERE id = $1", userID).Scan(&userName)

	for _, matchID := range matchIDs {
		database.DB.Exec("UPDATE match_players SET active = FALSE WHERE match_id = $1 AND user_id = $2", matchID, userID)
		BroadcastToMatch(matchID, domain.WSMessage{
			Type:     "player_left",
			UserID:   userID,
			UserName: userName,
		})
	}

	database.DB.Exec("UPDATE matches SET status = $1 WHERE creator_id = $2 AND status = $3", domain.MatchStatusFinished, userID, domain.MatchStatusWaiting)
	database.DB.Exec("UPDATE matches SET status = $1 WHERE creator_id = $2 AND status = $3", domain.MatchStatusFinished, userID, domain.MatchStatusPlaying)
	database.DB.Exec("UPDATE invites SET status = $1 WHERE target_user_id = $2 AND status = $3", domain.StatusExpired, userID, domain.StatusPending)

	httputil.RespondJSON(w, http.StatusOK, map[string]string{"message": "Left all matches"})
}

func GetActiveMatch(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	var matchID string
	err := database.DB.QueryRow(`
		SELECT mp.match_id FROM match_players mp
		JOIN matches m ON mp.match_id = m.id
		WHERE mp.user_id = $1 AND mp.active = TRUE AND m.status != $2
		LIMIT 1
	`, userID, domain.MatchStatusFinished).Scan(&matchID)

	if err != nil {
		httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{"match_id": nil})
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]string{"match_id": matchID})
}

func UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	var reqBody struct {
		Avatar string `json:"avatar"`
	}
	if err := httputil.DecodeJSON(r, &reqBody); err != nil {
		httputil.RespondError(w, err)
		return
	}

	if reqBody.Avatar != "" && len(reqBody.Avatar) > 500000 {
		httputil.RespondError(w, domain.ErrAvatarTooLarge)
		return
	}

	result, err := database.DB.Exec("UPDATE users SET avatar = $1 WHERE id = $2", reqBody.Avatar, userID)
	if err != nil {
		httputil.RespondError(w, domain.ErrUserNotFound)
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		httputil.RespondError(w, domain.ErrUserNotFound)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, map[string]string{"message": "Avatar updated successfully"})
}

// --- Helpers ---

func queryUsersByIDs(ids []string) ([]domain.OnlineUser, error) {
	placeholders, args := buildPlaceholders(ids)
	rows, err := database.DB.Query("SELECT id, name, COALESCE(avatar, '') FROM users WHERE id IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.OnlineUser
	for rows.Next() {
		var u domain.OnlineUser
		if rows.Scan(&u.ID, &u.Name, &u.Avatar) == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

func buildPlaceholders(ids []string) (string, []interface{}) {
	parts := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		parts[i] = "$" + itoa(i+1)
		args[i] = id
	}
	return joinStrings(parts, ","), args
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
