package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"adedanha-golang/internal/domain"
)

// RespondJSON writes a JSON response with the given status code
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// RespondError maps domain errors to HTTP status codes and writes error response
func RespondError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := err.Error()

	switch {
	case errors.Is(err, domain.ErrNotFound),
		errors.Is(err, domain.ErrMatchNotFound),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrRoundNotFound),
		errors.Is(err, domain.ErrInviteNotFound),
		errors.Is(err, domain.ErrRequestNotFound):
		status = http.StatusNotFound

	case errors.Is(err, domain.ErrNotCreator),
		errors.Is(err, domain.ErrNotTargetUser),
		errors.Is(err, domain.ErrPlayerNotInMatch),
		errors.Is(err, domain.ErrCreatorCantLeave):
		status = http.StatusForbidden

	case errors.Is(err, domain.ErrMatchFinished),
		errors.Is(err, domain.ErrMatchNotWaiting),
		errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrNameRequired),
		errors.Is(err, domain.ErrEmailRequired),
		errors.Is(err, domain.ErrMatchNameReq),
		errors.Is(err, domain.ErrAvatarTooLarge):
		status = http.StatusBadRequest

	case errors.Is(err, domain.ErrAlreadyInMatch),
		errors.Is(err, domain.ErrAlreadyRequested),
		errors.Is(err, domain.ErrEmailExists),
		errors.Is(err, domain.ErrPlayerInMatch):
		status = http.StatusConflict
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// DecodeJSON decodes a JSON request body into the given struct
func DecodeJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return domain.ErrInvalidInput
	}
	return nil
}
