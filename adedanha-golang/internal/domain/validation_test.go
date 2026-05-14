package domain

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"user.name@domain.co", true},
		{"user+tag@sub.domain.com", true},
		{"user@domain.com.br", true},
		{"", false},
		{"invalid", false},
		{"@domain.com", false},
		{"user@", false},
		{"user@.com", false},
		{"user@domain", false},
		{"user space@domain.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := ValidateEmail(tt.email)
			if result != tt.valid {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, result, tt.valid)
			}
		})
	}
}

func TestValidateCreateUser(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"John", "john@example.com", nil},
		{"", "john@example.com", ErrNameRequired},
		{"John", "", ErrEmailRequired},
		{"John", "invalid", ErrInvalidEmail},
		{"", "", ErrNameRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.email, func(t *testing.T) {
			err := ValidateCreateUser(tt.name, tt.email)
			if err != tt.wantErr {
				t.Errorf("ValidateCreateUser(%q, %q) = %v, want %v", tt.name, tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCreateMatch(t *testing.T) {
	tests := []struct {
		creatorID string
		name      string
		wantErr   error
	}{
		{"user-123", "My Match", nil},
		{"", "My Match", ErrInvalidInput},
		{"user-123", "", ErrMatchNameReq},
	}

	for _, tt := range tests {
		t.Run(tt.creatorID+"_"+tt.name, func(t *testing.T) {
			err := ValidateCreateMatch(tt.creatorID, tt.name)
			if err != tt.wantErr {
				t.Errorf("ValidateCreateMatch(%q, %q) = %v, want %v", tt.creatorID, tt.name, err, tt.wantErr)
			}
		})
	}
}
