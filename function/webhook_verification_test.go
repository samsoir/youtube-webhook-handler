package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleVerificationChallenge_Success(t *testing.T) {
	// Create request with challenge parameter
	req := httptest.NewRequest("GET", "/?hub.challenge=test-challenge-123", nil)
	w := httptest.NewRecorder()

	handleVerificationChallenge(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if body != "test-challenge-123" {
		t.Errorf("Expected body 'test-challenge-123', got '%s'", body)
	}
}

func TestHandleVerificationChallenge_MissingChallenge(t *testing.T) {
	// Create request without challenge parameter
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handleVerificationChallenge(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleVerificationChallenge_EmptyChallenge(t *testing.T) {
	// Create request with empty challenge parameter
	req := httptest.NewRequest("GET", "/?hub.challenge=", nil)
	w := httptest.NewRecorder()

	handleVerificationChallenge(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleVerificationChallenge_LongChallenge(t *testing.T) {
	// Test with a longer challenge string
	longChallenge := "test-challenge-with-very-long-string-abcdefghijklmnopqrstuvwxyz-123456789"
	req := httptest.NewRequest("GET", "/?hub.challenge="+longChallenge, nil)
	w := httptest.NewRecorder()

	handleVerificationChallenge(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if body != longChallenge {
		t.Errorf("Expected body '%s', got '%s'", longChallenge, body)
	}
}

func TestHandleVerificationChallenge_SpecialCharacters(t *testing.T) {
	// Test with characters that are safe in URL query parameters
	challenge := "test-challenge-with-safe-chars_123"
	req := httptest.NewRequest("GET", "/?hub.challenge="+challenge, nil)
	w := httptest.NewRecorder()

	handleVerificationChallenge(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if body != challenge {
		t.Errorf("Expected body '%s', got '%s'", challenge, body)
	}
}