package domain

import "regexp"

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail checks if an email has valid format
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// ValidateCreateUser validates user creation input
func ValidateCreateUser(name, email string) error {
	if name == "" {
		return ErrNameRequired
	}
	if email == "" {
		return ErrEmailRequired
	}
	if !ValidateEmail(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateCreateMatch validates match creation input
func ValidateCreateMatch(creatorID, name string) error {
	if creatorID == "" {
		return ErrInvalidInput
	}
	if name == "" {
		return ErrMatchNameReq
	}
	return nil
}
