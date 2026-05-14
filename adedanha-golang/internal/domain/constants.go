package domain

// Match statuses
const (
	MatchStatusWaiting  = "waiting"
	MatchStatusPlaying  = "playing"
	MatchStatusFinished = "finished"
)

// Round statuses
const (
	RoundStatusPlaying  = "playing"
	RoundStatusFinished = "finished"
)

// Join request / invite statuses
const (
	StatusPending  = "pending"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
	StatusExpired  = "expired"
)

// Valid letters for the game (excluding K, W, X, Y)
var ValidLetters = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
	"L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "Z",
}

// Score values for validation
const (
	ScoreValid     = 10
	ScoreUncertain = 5
	ScoreInvalid   = 0
)

// Round duration in seconds
const RoundDurationSeconds = 60
