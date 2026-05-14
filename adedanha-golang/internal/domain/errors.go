package domain

import "errors"

// Domain errors
var (
	ErrNotFound        = errors.New("not found")
	ErrMatchNotFound   = errors.New("match not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrRoundNotFound   = errors.New("round not found")
	ErrInviteNotFound  = errors.New("invite not found or already processed")
	ErrRequestNotFound = errors.New("join request not found")

	ErrNotCreator       = errors.New("only the match creator can perform this action")
	ErrNotTargetUser    = errors.New("this action is not for you")
	ErrMatchFinished    = errors.New("match is already finished")
	ErrMatchNotWaiting  = errors.New("match is not accepting new players")
	ErrAlreadyInMatch   = errors.New("user is already in an active match")
	ErrAlreadyRequested = errors.New("join request already pending")
	ErrCreatorCantLeave = errors.New("creator cannot leave, end the match instead")
	ErrPlayerNotInMatch = errors.New("user is not in this match")
	ErrPlayerInMatch    = errors.New("player is already in an active match")

	ErrInvalidInput   = errors.New("invalid input")
	ErrInvalidEmail   = errors.New("invalid email format")
	ErrEmailExists    = errors.New("email already exists")
	ErrNameRequired   = errors.New("name is required")
	ErrEmailRequired  = errors.New("email is required")
	ErrMatchNameReq   = errors.New("match name is required")
	ErrAvatarTooLarge = errors.New("avatar too large, max 500KB")

	ErrNoLettersLeft = errors.New("no letters available")
)
