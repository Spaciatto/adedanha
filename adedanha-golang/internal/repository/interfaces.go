package repository

import "adedanha-golang/internal/domain"

// UserRepository defines data access for users
type UserRepository interface {
	Create(user domain.User) error
	GetByID(id string) (domain.User, error)
	GetByEmail(email string) (domain.User, error)
	Update(id, name, email string) (domain.User, error)
	UpdateAvatar(id, avatar string) error
	Exists(id string) (bool, error)
	GetOnlineUsersByIDs(ids []string) ([]domain.OnlineUser, error)
	GetAvailablePlayersByIDs(ids []string) ([]domain.OnlineUser, error)
}

// MatchRepository defines data access for matches
type MatchRepository interface {
	Create(match domain.Match) error
	GetByID(id string) (domain.Match, error)
	GetWithPlayers(id string) (domain.Match, error)
	UpdateStatus(id, status string) error
	UpdateCurrentRound(id string, round int) error
	IsCreator(matchID, userID string) (bool, error)
	ListOpen() ([]domain.OpenMatch, error)

	// Players
	AddPlayer(matchID, userID string) error
	SetPlayerActive(matchID, userID string, active bool) error
	IsPlayerActive(matchID, userID string) (bool, error)
	PlayerExists(matchID, userID string) (bool, error)
	CountActivePlayers(matchID string) (int, error)
	IsUserInActiveMatch(userID string) (bool, error)
	IsUserInOtherActiveMatch(userID, excludeMatchID string) (bool, error)
	GetUserActiveMatchID(userID string) (string, error)
	DeactivateAllPlayerMatches(userID string) ([]string, error)
	FinishCreatorMatches(userID string) error
}

// RoundRepository defines data access for rounds
type RoundRepository interface {
	Create(round domain.Round) error
	GetByID(id string) (domain.Round, error)
	GetLatestByMatch(matchID string) (domain.Round, error)
	UpdateStatus(id, status string) error
	GetUsedLetters(matchID string) ([]string, error)
	CountAnswers(roundID string) (int, error)
}

// AnswerRepository defines data access for answers
type AnswerRepository interface {
	Upsert(answer domain.Answer) error
	GetByRound(roundID string) ([]domain.Answer, error)
	UpdateScore(roundID, userID string, score int) error
	GetRanking(matchID string) ([]domain.RankingEntry, error)
}

// InviteRepository defines data access for invites
type InviteRepository interface {
	Create(id, matchID, matchName, inviterName, targetUserID string) error
	GetPending(targetUserID string) ([]domain.Invite, error)
	GetByID(id string) (matchID, targetUserID string, err error)
	UpdateStatus(id, status string) error
	ExpireForUser(userID string) error
}

// JoinRequestRepository defines data access for join requests
type JoinRequestRepository interface {
	Create(req domain.JoinRequest) error
	GetPendingByMatch(matchID string) ([]domain.JoinRequest, error)
	GetByID(id, matchID string) (domain.JoinRequest, error)
	UpdateStatus(id, status string) error
	HasPending(matchID, userID string) (bool, error)
}
