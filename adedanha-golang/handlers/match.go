package handlers

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"adedanha-golang/database"
	"adedanha-golang/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var validLetters = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
	"L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "Z",
}

// roundStopChannels stores cancel channels for active round timers
var roundStopChannels = make(map[string]chan struct{})
var roundStopMu sync.Mutex
var roundStopped = make(map[string]bool) // tracks already-stopped rounds

func CreateMatch(w http.ResponseWriter, r *http.Request) {
	var req models.CreateMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.CreatorID == "" {
		http.Error(w, `{"error":"creator_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}

	var exists bool
	if err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", req.CreatorID).Scan(&exists); err != nil || !exists {
		http.Error(w, `{"error":"Creator user not found"}`, http.StatusNotFound)
		return
	}

	var inActiveMatch bool
	if err := database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM match_players mp
			JOIN matches m ON mp.match_id = m.id
			WHERE mp.user_id = ? AND mp.active = 1 AND m.status != 'finished'
		)`, req.CreatorID).Scan(&inActiveMatch); err != nil {
		log.Printf("Error checking active match: %v", err)
	}
	if inActiveMatch {
		http.Error(w, `{"error":"Você já está em uma partida ativa. Abandone a partida atual antes de criar outra."}`, http.StatusConflict)
		return
	}

	match := models.Match{
		ID:           uuid.New().String(),
		Name:         req.Name,
		CreatorID:    req.CreatorID,
		Status:       "waiting",
		CurrentRound: 0,
		CreatedAt:    time.Now(),
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error":"Failed to create match"}`, http.StatusInternalServerError)
		return
	}

	if _, err = tx.Exec(
		"INSERT INTO matches (id, name, creator_id, status, current_round, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		match.ID, match.Name, match.CreatorID, match.Status, match.CurrentRound, match.CreatedAt,
	); err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"Failed to create match"}`, http.StatusInternalServerError)
		return
	}

	if _, err = tx.Exec(
		"INSERT INTO match_players (match_id, user_id, active, joined_at) VALUES (?, ?, 1, ?)",
		match.ID, req.CreatorID, time.Now(),
	); err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"Failed to add creator to match"}`, http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, `{"error":"Failed to create match"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(match)
}

func JoinMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	var req models.JoinMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, `{"error":"user_id is required"}`, http.StatusBadRequest)
		return
	}

	var status string
	if err := database.DB.QueryRow("SELECT status FROM matches WHERE id = ?", matchID).Scan(&status); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}
	if status == "finished" {
		http.Error(w, `{"error":"Match is already finished"}`, http.StatusBadRequest)
		return
	}

	var userName string
	if err := database.DB.QueryRow("SELECT name FROM users WHERE id = ?", req.UserID).Scan(&userName); err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	var alreadyJoined bool
	database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM match_players WHERE match_id = ? AND user_id = ?)",
		matchID, req.UserID,
	).Scan(&alreadyJoined)

	if alreadyJoined {
		var isActive bool
		database.DB.QueryRow(
			"SELECT active FROM match_players WHERE match_id = ? AND user_id = ?",
			matchID, req.UserID,
		).Scan(&isActive)

		if isActive {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "Reconnected to match successfully"})
			return
		}
		// Reactivate abandoned player
		database.DB.Exec("UPDATE match_players SET active = 1 WHERE match_id = ? AND user_id = ?", matchID, req.UserID)
		BroadcastToMatch(matchID, models.WSMessage{
			Type:     "player_joined",
			UserID:   req.UserID,
			UserName: userName,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Rejoined match successfully"})
		return
	}

	if status != "waiting" {
		http.Error(w, `{"error":"Match is not accepting new players"}`, http.StatusBadRequest)
		return
	}

	var inOtherMatch bool
	database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM match_players mp
			JOIN matches m ON mp.match_id = m.id
			WHERE mp.user_id = ? AND mp.active = 1 AND m.status != 'finished' AND mp.match_id != ?
		)`, req.UserID, matchID).Scan(&inOtherMatch)
	if inOtherMatch {
		http.Error(w, `{"error":"Você já está em uma partida ativa. Abandone a partida atual antes de entrar em outra."}`, http.StatusConflict)
		return
	}

	if _, err := database.DB.Exec(
		"INSERT INTO match_players (match_id, user_id, active, joined_at) VALUES (?, ?, 1, ?)",
		matchID, req.UserID, time.Now(),
	); err != nil {
		http.Error(w, `{"error":"User already in match or failed to join"}`, http.StatusConflict)
		return
	}

	BroadcastToMatch(matchID, models.WSMessage{
		Type:     "player_joined",
		UserID:   req.UserID,
		UserName: userName,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Joined match successfully"})
}

func GetMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	var match models.Match
	if err := database.DB.QueryRow(
		"SELECT id, name, creator_id, status, current_round, created_at FROM matches WHERE id = ?", matchID,
	).Scan(&match.ID, &match.Name, &match.CreatorID, &match.Status, &match.CurrentRound, &match.CreatedAt); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}

	rows, err := database.DB.Query(`
		SELECT mp.match_id, mp.user_id, u.name, mp.active, mp.joined_at 
		FROM match_players mp JOIN users u ON mp.user_id = u.id 
		WHERE mp.match_id = ?`, matchID)
	if err != nil {
		http.Error(w, `{"error":"Failed to get match players"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	match.Players = []models.MatchPlayer{}
	for rows.Next() {
		var p models.MatchPlayer
		if err := rows.Scan(&p.MatchID, &p.UserID, &p.UserName, &p.Active, &p.JoinedAt); err == nil {
			match.Players = append(match.Players, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(match)
}

func StartRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	var match models.Match
	if err := database.DB.QueryRow(
		"SELECT id, creator_id, status, current_round FROM matches WHERE id = ?", matchID,
	).Scan(&match.ID, &match.CreatorID, &match.Status, &match.CurrentRound); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}

	var reqBody struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if reqBody.UserID != match.CreatorID {
		http.Error(w, `{"error":"Only the match creator can start rounds"}`, http.StatusForbidden)
		return
	}
	if match.Status == "finished" {
		http.Error(w, `{"error":"Match is already finished"}`, http.StatusBadRequest)
		return
	}

	if match.Status == "waiting" {
		database.DB.Exec("UPDATE matches SET status = 'playing' WHERE id = ?", matchID)
	}

	letter := validLetters[rand.Intn(len(validLetters))]
	newRoundNumber := match.CurrentRound + 1
	now := time.Now()
	endsAt := now.Add(60 * time.Second)

	round := models.Round{
		ID:          uuid.New().String(),
		MatchID:     matchID,
		RoundNumber: newRoundNumber,
		Letter:      letter,
		Status:      "playing",
		StartedAt:   now,
		EndsAt:      endsAt,
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error":"Failed to start round"}`, http.StatusInternalServerError)
		return
	}

	if _, err = tx.Exec(
		"INSERT INTO rounds (id, match_id, round_number, letter, status, started_at, ends_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		round.ID, round.MatchID, round.RoundNumber, round.Letter, round.Status, round.StartedAt, round.EndsAt,
	); err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"Failed to create round"}`, http.StatusInternalServerError)
		return
	}

	if _, err = tx.Exec("UPDATE matches SET current_round = ? WHERE id = ?", newRoundNumber, matchID); err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"Failed to update match"}`, http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, `{"error":"Failed to start round"}`, http.StatusInternalServerError)
		return
	}

	BroadcastToMatch(matchID, models.WSMessage{
		Type:    "round_started",
		Letter:  letter,
		RoundID: round.ID,
		EndsAt:  endsAt,
	})

	go runRoundTimer(matchID, round.ID, endsAt)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(round)
}

func runRoundTimer(matchID, roundID string, endsAt time.Time) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	stopCh := make(chan struct{})
	roundStopMu.Lock()
	roundStopChannels[roundID] = stopCh
	roundStopped[roundID] = false
	roundStopMu.Unlock()

	defer func() {
		roundStopMu.Lock()
		delete(roundStopChannels, roundID)
		delete(roundStopped, roundID)
		roundStopMu.Unlock()
	}()

	for {
		select {
		case <-stopCh:
			database.DB.Exec("UPDATE rounds SET status = 'finished' WHERE id = ?", roundID)
			BroadcastToMatch(matchID, models.WSMessage{
				Type:    "round_ended",
				RoundID: roundID,
			})
			return
		case <-ticker.C:
			remaining := int(time.Until(endsAt).Seconds())
			if remaining <= 0 {
				database.DB.Exec("UPDATE rounds SET status = 'finished' WHERE id = ?", roundID)
				BroadcastToMatch(matchID, models.WSMessage{
					Type:    "round_ended",
					RoundID: roundID,
				})
				return
			}
			BroadcastToMatch(matchID, models.WSMessage{
				Type:             "timer_tick",
				SecondsRemaining: remaining,
			})
		}
	}
}

// checkAllPlayersSubmitted ends the round early if all active players submitted
func checkAllPlayersSubmitted(matchID, roundID string) {
	var activePlayers int
	if err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM match_players WHERE match_id = ? AND active = 1", matchID,
	).Scan(&activePlayers); err != nil {
		return
	}

	var answersCount int
	if err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM answers WHERE round_id = ?", roundID,
	).Scan(&answersCount); err != nil {
		return
	}

	if answersCount >= activePlayers && activePlayers > 0 {
		roundStopMu.Lock()
		defer roundStopMu.Unlock()
		// Only close once — prevent double-close panic
		if stopped, exists := roundStopped[roundID]; exists && !stopped {
			roundStopped[roundID] = true
			if stopCh, ok := roundStopChannels[roundID]; ok {
				close(stopCh)
			}
		}
	}
}

func SubmitAnswers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]
	roundID := vars["roundId"]

	var req models.SubmitAnswersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, `{"error":"user_id is required"}`, http.StatusBadRequest)
		return
	}

	var roundStatus string
	if err := database.DB.QueryRow(
		"SELECT status FROM rounds WHERE id = ? AND match_id = ?", roundID, matchID,
	).Scan(&roundStatus); err != nil {
		http.Error(w, `{"error":"Round not found"}`, http.StatusNotFound)
		return
	}

	var playerExists bool
	if err := database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM match_players WHERE match_id = ? AND user_id = ?)",
		matchID, req.UserID,
	).Scan(&playerExists); err != nil || !playerExists {
		http.Error(w, `{"error":"User is not in this match"}`, http.StatusForbidden)
		return
	}

	answer := models.Answer{
		ID:          uuid.New().String(),
		RoundID:     roundID,
		UserID:      req.UserID,
		Color:       strings.ToUpper(req.Color),
		Fruit:       strings.ToUpper(req.Fruit),
		Object:      strings.ToUpper(req.Object),
		Movie:       strings.ToUpper(req.Movie),
		City:        strings.ToUpper(req.City),
		Score:       0,
		SubmittedAt: time.Now(),
	}

	if _, err := database.DB.Exec(
		`INSERT INTO answers (id, round_id, user_id, color, fruit, object, movie, city, score, submitted_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(round_id, user_id) DO UPDATE SET color=excluded.color, fruit=excluded.fruit, object=excluded.object, movie=excluded.movie, city=excluded.city, submitted_at=excluded.submitted_at`,
		answer.ID, answer.RoundID, answer.UserID, answer.Color, answer.Fruit, answer.Object, answer.Movie, answer.City, answer.Score, answer.SubmittedAt,
	); err != nil {
		http.Error(w, `{"error":"Failed to submit answers"}`, http.StatusInternalServerError)
		return
	}

	if roundStatus == "playing" {
		go checkAllPlayersSubmitted(matchID, roundID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(answer)
}

func UpdateScores(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]
	roundID := vars["roundId"]

	var req struct {
		UserID string               `json:"user_id"`
		Scores []models.PlayerScore `json:"scores"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, `{"error":"user_id is required"}`, http.StatusBadRequest)
		return
	}

	var creatorID string
	if err := database.DB.QueryRow("SELECT creator_id FROM matches WHERE id = ?", matchID).Scan(&creatorID); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}
	if req.UserID != creatorID {
		http.Error(w, `{"error":"Only the match creator can update scores"}`, http.StatusForbidden)
		return
	}

	for _, score := range req.Scores {
		if _, err := database.DB.Exec(
			"UPDATE answers SET score = ? WHERE round_id = ? AND user_id = ?",
			score.Score, roundID, score.UserID,
		); err != nil {
			log.Printf("Error updating score for user %s: %v", score.UserID, err)
		}
	}

	BroadcastToMatch(matchID, models.WSMessage{
		Type:    "scores_updated",
		RoundID: roundID,
		Scores:  req.Scores,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Scores updated successfully"})
}

func EndMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	var reqBody struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	var creatorID string
	if err := database.DB.QueryRow("SELECT creator_id FROM matches WHERE id = ?", matchID).Scan(&creatorID); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}
	if reqBody.UserID != creatorID {
		http.Error(w, `{"error":"Only the match creator can end the match"}`, http.StatusForbidden)
		return
	}

	if _, err := database.DB.Exec("UPDATE matches SET status = 'finished' WHERE id = ?", matchID); err != nil {
		http.Error(w, `{"error":"Failed to end match"}`, http.StatusInternalServerError)
		return
	}

	ranking := calculateRanking(matchID)

	BroadcastToMatch(matchID, models.WSMessage{
		Type:    "match_ended",
		Ranking: ranking,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Match ended successfully",
		"ranking": ranking,
	})
}

func calculateRanking(matchID string) []models.RankingEntry {
	rows, err := database.DB.Query(`
		SELECT a.user_id, u.name, COALESCE(SUM(a.score), 0) as total_score
		FROM answers a
		JOIN rounds r ON a.round_id = r.id
		JOIN users u ON a.user_id = u.id
		WHERE r.match_id = ?
		GROUP BY a.user_id, u.name
		ORDER BY total_score DESC
	`, matchID)
	if err != nil {
		return []models.RankingEntry{}
	}
	defer rows.Close()

	var ranking []models.RankingEntry
	position := 1
	for rows.Next() {
		var entry models.RankingEntry
		if err := rows.Scan(&entry.UserID, &entry.UserName, &entry.TotalScore); err == nil {
			entry.Position = position
			ranking = append(ranking, entry)
			position++
		}
	}
	return ranking
}

func GetMatchState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	var match models.Match
	if err := database.DB.QueryRow(
		"SELECT id, name, creator_id, status, current_round, created_at FROM matches WHERE id = ?", matchID,
	).Scan(&match.ID, &match.Name, &match.CreatorID, &match.Status, &match.CurrentRound, &match.CreatedAt); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}

	rows, err := database.DB.Query(`
		SELECT mp.match_id, mp.user_id, u.name, mp.active, mp.joined_at 
		FROM match_players mp JOIN users u ON mp.user_id = u.id 
		WHERE mp.match_id = ?`, matchID)
	if err != nil {
		http.Error(w, `{"error":"Failed to get match players"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	match.Players = []models.MatchPlayer{}
	for rows.Next() {
		var p models.MatchPlayer
		if err := rows.Scan(&p.MatchID, &p.UserID, &p.UserName, &p.Active, &p.JoinedAt); err == nil {
			match.Players = append(match.Players, p)
		}
	}

	state := models.MatchState{Match: match, Phase: "lobby"}

	if match.Status == "finished" {
		state.Phase = "finished"
		state.Ranking = calculateRanking(matchID)
	} else if match.Status == "playing" {
		var round models.Round
		err := database.DB.QueryRow(`
			SELECT id, match_id, round_number, letter, status, started_at, ends_at
			FROM rounds WHERE match_id = ? ORDER BY round_number DESC LIMIT 1`, matchID,
		).Scan(&round.ID, &round.MatchID, &round.RoundNumber, &round.Letter, &round.Status, &round.StartedAt, &round.EndsAt)
		if err == nil {
			state.CurrentRound = &round
			if round.Status == "playing" {
				state.Phase = "playing"
			} else {
				state.Phase = "round_ended"
				answerRows, err := database.DB.Query(
					"SELECT id, round_id, user_id, color, fruit, object, movie, city, score, submitted_at FROM answers WHERE round_id = ?",
					round.ID,
				)
				if err == nil {
					defer answerRows.Close()
					var answers []models.Answer
					for answerRows.Next() {
						var a models.Answer
						if err := answerRows.Scan(&a.ID, &a.RoundID, &a.UserID, &a.Color, &a.Fruit, &a.Object, &a.Movie, &a.City, &a.Score, &a.SubmittedAt); err == nil {
							answers = append(answers, a)
						}
					}
					state.RoundResult = &models.RoundResult{
						RoundID: round.ID,
						Letter:  round.Letter,
						Answers: answers,
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func LeaveMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	var reqBody struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if reqBody.UserID == "" {
		http.Error(w, `{"error":"user_id is required"}`, http.StatusBadRequest)
		return
	}

	var playerActive bool
	if err := database.DB.QueryRow(
		"SELECT active FROM match_players WHERE match_id = ? AND user_id = ?",
		matchID, reqBody.UserID,
	).Scan(&playerActive); err != nil {
		http.Error(w, `{"error":"User is not in this match"}`, http.StatusNotFound)
		return
	}

	var creatorID string
	database.DB.QueryRow("SELECT creator_id FROM matches WHERE id = ?", matchID).Scan(&creatorID)
	if reqBody.UserID == creatorID {
		http.Error(w, `{"error":"O criador não pode abandonar a partida. Encerre a partida ao invés disso."}`, http.StatusForbidden)
		return
	}

	if _, err := database.DB.Exec(
		"UPDATE match_players SET active = 0 WHERE match_id = ? AND user_id = ?",
		matchID, reqBody.UserID,
	); err != nil {
		http.Error(w, `{"error":"Failed to leave match"}`, http.StatusInternalServerError)
		return
	}

	var userName string
	database.DB.QueryRow("SELECT name FROM users WHERE id = ?", reqBody.UserID).Scan(&userName)

	BroadcastToMatch(matchID, models.WSMessage{
		Type:     "player_left",
		UserID:   reqBody.UserID,
		UserName: userName,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Left match successfully"})
}

func ListOpenMatches(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT m.id, m.name, u.name, 
			(SELECT COUNT(*) FROM match_players mp WHERE mp.match_id = m.id AND mp.active = 1) as player_count,
			m.status
		FROM matches m
		JOIN users u ON m.creator_id = u.id
		WHERE m.status = 'waiting'
		ORDER BY m.created_at DESC
	`)
	if err != nil {
		http.Error(w, `{"error":"Failed to list matches"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	matches := []models.OpenMatch{}
	for rows.Next() {
		var m models.OpenMatch
		if err := rows.Scan(&m.ID, &m.Name, &m.CreatorName, &m.PlayerCount, &m.Status); err == nil {
			matches = append(matches, m)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}

func RequestJoinMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	var reqBody struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if reqBody.UserID == "" {
		http.Error(w, `{"error":"user_id is required"}`, http.StatusBadRequest)
		return
	}

	var status string
	if err := database.DB.QueryRow("SELECT status FROM matches WHERE id = ?", matchID).Scan(&status); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}
	if status != "waiting" {
		http.Error(w, `{"error":"Match is not accepting join requests"}`, http.StatusBadRequest)
		return
	}

	var inOtherMatch bool
	database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM match_players mp
			JOIN matches m ON mp.match_id = m.id
			WHERE mp.user_id = ? AND mp.active = 1 AND m.status != 'finished'
		)`, reqBody.UserID).Scan(&inOtherMatch)
	if inOtherMatch {
		http.Error(w, `{"error":"Você já está em uma partida ativa. Abandone a partida atual primeiro."}`, http.StatusConflict)
		return
	}

	var existingRequest bool
	database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM join_requests WHERE match_id = ? AND user_id = ? AND status = 'pending')",
		matchID, reqBody.UserID,
	).Scan(&existingRequest)
	if existingRequest {
		http.Error(w, `{"error":"Você já tem uma solicitação pendente para esta partida"}`, http.StatusConflict)
		return
	}

	var userName string
	if err := database.DB.QueryRow("SELECT name FROM users WHERE id = ?", reqBody.UserID).Scan(&userName); err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	requestID := uuid.New().String()
	if _, err := database.DB.Exec(
		"INSERT INTO join_requests (id, match_id, user_id, user_name, status, created_at) VALUES (?, ?, ?, ?, 'pending', ?)",
		requestID, matchID, reqBody.UserID, userName, time.Now(),
	); err != nil {
		http.Error(w, `{"error":"Failed to create join request"}`, http.StatusInternalServerError)
		return
	}

	BroadcastToMatch(matchID, models.WSMessage{
		Type:      "join_request",
		UserID:    reqBody.UserID,
		UserName:  userName,
		RequestID: requestID,
		MatchID:   matchID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.JoinRequest{
		ID:        requestID,
		MatchID:   matchID,
		UserID:    reqBody.UserID,
		UserName:  userName,
		Status:    "pending",
		CreatedAt: time.Now(),
	})
}

func GetJoinRequests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	rows, err := database.DB.Query(
		"SELECT id, match_id, user_id, user_name, status, created_at FROM join_requests WHERE match_id = ? AND status = 'pending' ORDER BY created_at ASC",
		matchID,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to get join requests"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	requests := []models.JoinRequest{}
	for rows.Next() {
		var jr models.JoinRequest
		if err := rows.Scan(&jr.ID, &jr.MatchID, &jr.UserID, &jr.UserName, &jr.Status, &jr.CreatedAt); err == nil {
			requests = append(requests, jr)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

func RespondJoinRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]
	requestID := vars["requestId"]

	var reqBody struct {
		UserID   string `json:"user_id"`
		Accepted bool   `json:"accepted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	var creatorID string
	if err := database.DB.QueryRow("SELECT creator_id FROM matches WHERE id = ?", matchID).Scan(&creatorID); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}
	if reqBody.UserID != creatorID {
		http.Error(w, `{"error":"Only the match creator can respond to join requests"}`, http.StatusForbidden)
		return
	}

	var jr models.JoinRequest
	if err := database.DB.QueryRow(
		"SELECT id, match_id, user_id, user_name, status FROM join_requests WHERE id = ? AND match_id = ?",
		requestID, matchID,
	).Scan(&jr.ID, &jr.MatchID, &jr.UserID, &jr.UserName, &jr.Status); err != nil {
		http.Error(w, `{"error":"Join request not found"}`, http.StatusNotFound)
		return
	}
	if jr.Status != "pending" {
		http.Error(w, `{"error":"Join request already processed"}`, http.StatusBadRequest)
		return
	}

	if reqBody.Accepted {
		database.DB.Exec(
			"INSERT OR IGNORE INTO match_players (match_id, user_id, active, joined_at) VALUES (?, ?, 1, ?)",
			matchID, jr.UserID, time.Now(),
		)
		database.DB.Exec("UPDATE join_requests SET status = 'accepted' WHERE id = ?", requestID)

		BroadcastToMatch(matchID, models.WSMessage{
			Type:     "player_joined",
			UserID:   jr.UserID,
			UserName: jr.UserName,
		})
		BroadcastToGlobal(models.WSMessage{
			Type:    "join_accepted",
			UserID:  jr.UserID,
			MatchID: matchID,
		})
	} else {
		database.DB.Exec("UPDATE join_requests SET status = 'rejected' WHERE id = ?", requestID)
		BroadcastToGlobal(models.WSMessage{
			Type:    "join_rejected",
			UserID:  jr.UserID,
			MatchID: matchID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Request processed"})
}

func InvitePlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]

	var reqBody struct {
		CreatorID string `json:"creator_id"`
		PlayerID  string `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if reqBody.CreatorID == "" || reqBody.PlayerID == "" {
		http.Error(w, `{"error":"creator_id and player_id are required"}`, http.StatusBadRequest)
		return
	}

	var creatorID, matchName string
	if err := database.DB.QueryRow("SELECT creator_id, name FROM matches WHERE id = ?", matchID).Scan(&creatorID, &matchName); err != nil {
		http.Error(w, `{"error":"Match not found"}`, http.StatusNotFound)
		return
	}
	if reqBody.CreatorID != creatorID {
		http.Error(w, `{"error":"Only the match creator can invite players"}`, http.StatusForbidden)
		return
	}

	var status string
	database.DB.QueryRow("SELECT status FROM matches WHERE id = ?", matchID).Scan(&status)
	if status != "waiting" {
		http.Error(w, `{"error":"Match is not accepting new players"}`, http.StatusBadRequest)
		return
	}

	var playerName string
	if err := database.DB.QueryRow("SELECT name FROM users WHERE id = ?", reqBody.PlayerID).Scan(&playerName); err != nil {
		http.Error(w, `{"error":"Player not found"}`, http.StatusNotFound)
		return
	}

	var inActiveMatch bool
	database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM match_players mp
			JOIN matches m ON mp.match_id = m.id
			WHERE mp.user_id = ? AND mp.active = 1 AND m.status != 'finished'
		)`, reqBody.PlayerID).Scan(&inActiveMatch)
	if inActiveMatch {
		http.Error(w, `{"error":"Jogador já está em uma partida ativa"}`, http.StatusConflict)
		return
	}

	var creatorName string
	database.DB.QueryRow("SELECT name FROM users WHERE id = ?", creatorID).Scan(&creatorName)

	inviteID := uuid.New().String()
	if _, err := database.DB.Exec(
		"INSERT INTO invites (id, match_id, match_name, inviter_name, target_user_id, status, created_at) VALUES (?, ?, ?, ?, ?, 'pending', ?)",
		inviteID, matchID, matchName, creatorName, reqBody.PlayerID, time.Now(),
	); err != nil {
		log.Printf("Error persisting invite: %v", err)
	}

	BroadcastToGlobal(models.WSMessage{
		Type:      "match_invite",
		UserID:    reqBody.PlayerID,
		UserName:  creatorName,
		MatchID:   matchID,
		MatchName: matchName,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Invite sent successfully"})
}

func GetRoundResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]
	roundID := vars["roundId"]

	var letter string
	if err := database.DB.QueryRow(
		"SELECT letter FROM rounds WHERE id = ? AND match_id = ?", roundID, matchID,
	).Scan(&letter); err != nil {
		http.Error(w, `{"error":"Round not found"}`, http.StatusNotFound)
		return
	}

	rows, err := database.DB.Query(
		"SELECT id, round_id, user_id, color, fruit, object, movie, city, score, submitted_at FROM answers WHERE round_id = ?",
		roundID,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to get results"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var answers []models.Answer
	for rows.Next() {
		var a models.Answer
		if err := rows.Scan(&a.ID, &a.RoundID, &a.UserID, &a.Color, &a.Fruit, &a.Object, &a.Movie, &a.City, &a.Score, &a.SubmittedAt); err == nil {
			answers = append(answers, a)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.RoundResult{
		RoundID: roundID,
		Letter:  letter,
		Answers: answers,
	})
}
