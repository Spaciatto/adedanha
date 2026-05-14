// Package models re-exports domain types for backward compatibility.
// New code should import from internal/domain directly.
package models

import "adedanha-golang/internal/domain"

// Type aliases for backward compatibility
type User = domain.User
type Match = domain.Match
type MatchPlayer = domain.MatchPlayer
type Round = domain.Round
type Answer = domain.Answer
type RoundResult = domain.RoundResult
type RankingEntry = domain.RankingEntry
type JoinRequest = domain.JoinRequest
type OnlineUser = domain.OnlineUser
type OpenMatch = domain.OpenMatch
type MatchState = domain.MatchState
type WSMessage = domain.WSMessage
type PlayerScore = domain.PlayerScore

// Request types
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateMatchRequest struct {
	CreatorID string `json:"creator_id"`
	Name      string `json:"name"`
}

type JoinMatchRequest struct {
	UserID string `json:"user_id"`
}

type SubmitAnswersRequest struct {
	UserID string `json:"user_id"`
	Color  string `json:"color"`
	Fruit  string `json:"fruit"`
	Object string `json:"object"`
	Movie  string `json:"movie"`
	City   string `json:"city"`
	Animal string `json:"animal"`
	Name   string `json:"name"`
}

type UpdateScoresRequest struct {
	Scores []PlayerScore `json:"scores"`
}
