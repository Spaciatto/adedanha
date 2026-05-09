package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Match struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	CreatorID    string        `json:"creator_id"`
	Status       string        `json:"status"`
	CurrentRound int           `json:"current_round"`
	CreatedAt    time.Time     `json:"created_at"`
	Players      []MatchPlayer `json:"players,omitempty"`
}

type MatchPlayer struct {
	MatchID  string    `json:"match_id"`
	UserID   string    `json:"user_id"`
	UserName string    `json:"user_name,omitempty"`
	Active   bool      `json:"active"`
	JoinedAt time.Time `json:"joined_at"`
}

type CreateMatchRequest struct {
	CreatorID string `json:"creator_id"`
	Name      string `json:"name"`
}

type JoinMatchRequest struct {
	UserID string `json:"user_id"`
}

type Round struct {
	ID          string    `json:"id"`
	MatchID     string    `json:"match_id"`
	RoundNumber int       `json:"round_number"`
	Letter      string    `json:"letter"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	EndsAt      time.Time `json:"ends_at"`
}

type Answer struct {
	ID          string    `json:"id"`
	RoundID     string    `json:"round_id"`
	UserID      string    `json:"user_id"`
	Color       string    `json:"color"`
	Fruit       string    `json:"fruit"`
	Object      string    `json:"object"`
	Movie       string    `json:"movie"`
	City        string    `json:"city"`
	Score       int       `json:"score"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type SubmitAnswersRequest struct {
	UserID string `json:"user_id"`
	Color  string `json:"color"`
	Fruit  string `json:"fruit"`
	Object string `json:"object"`
	Movie  string `json:"movie"`
	City   string `json:"city"`
}

type UpdateScoresRequest struct {
	Scores []PlayerScore `json:"scores"`
}

type PlayerScore struct {
	UserID string `json:"user_id"`
	Score  int    `json:"score"`
}

type RoundResult struct {
	RoundID string   `json:"round_id"`
	Letter  string   `json:"letter"`
	Answers []Answer `json:"answers"`
}

// Ranking entry for final match results
type RankingEntry struct {
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
	TotalScore int    `json:"total_score"`
	Position   int    `json:"position"`
}

// Online user info
type OnlineUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Join request
type JoinRequest struct {
	ID        string    `json:"id"`
	MatchID   string    `json:"match_id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Open match listing
type OpenMatch struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CreatorName string `json:"creator_name"`
	PlayerCount int    `json:"player_count"`
	Status      string `json:"status"`
}

// Match state for reconnection
type MatchState struct {
	Match        Match          `json:"match"`
	Phase        string         `json:"phase"`
	CurrentRound *Round         `json:"current_round,omitempty"`
	RoundResult  *RoundResult   `json:"round_result,omitempty"`
	Ranking      []RankingEntry `json:"ranking,omitempty"`
}

// WebSocket message types
type WSMessage struct {
	Type             string         `json:"type"`
	Letter           string         `json:"letter,omitempty"`
	RoundID          string         `json:"round_id,omitempty"`
	EndsAt           time.Time      `json:"ends_at,omitempty"`
	SecondsRemaining int            `json:"seconds_remaining,omitempty"`
	Scores           []PlayerScore  `json:"scores,omitempty"`
	UserID           string         `json:"user_id,omitempty"`
	UserName         string         `json:"user_name,omitempty"`
	Ranking          []RankingEntry `json:"ranking,omitempty"`
	RequestID        string         `json:"request_id,omitempty"`
	MatchID          string         `json:"match_id,omitempty"`
	MatchName        string         `json:"match_name,omitempty"`
}
