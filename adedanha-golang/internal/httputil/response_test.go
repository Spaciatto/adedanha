package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"adedanha-golang/internal/domain"
)

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "ok"}

	RespondJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["message"] != "ok" {
		t.Errorf("Expected message 'ok', got '%s'", result["message"])
	}
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
	}{
		{domain.ErrMatchNotFound, http.StatusNotFound},
		{domain.ErrUserNotFound, http.StatusNotFound},
		{domain.ErrNotCreator, http.StatusForbidden},
		{domain.ErrCreatorCantLeave, http.StatusForbidden},
		{domain.ErrMatchFinished, http.StatusBadRequest},
		{domain.ErrInvalidEmail, http.StatusBadRequest},
		{domain.ErrAlreadyInMatch, http.StatusConflict},
		{domain.ErrEmailExists, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			w := httptest.NewRecorder()
			RespondError(w, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("RespondError(%v) status = %d, want %d", tt.err, w.Code, tt.wantStatus)
			}

			var result map[string]string
			json.Unmarshal(w.Body.Bytes(), &result)
			if result["error"] == "" {
				t.Error("Expected error message in response body")
			}
		})
	}
}

func TestRespondJSONNilData(t *testing.T) {
	w := httptest.NewRecorder()
	RespondJSON(w, http.StatusNoContent, nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}
