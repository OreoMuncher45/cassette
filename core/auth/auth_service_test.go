package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeOauthCallbackHandlerRejectsStateMismatch(t *testing.T) {
	service := NewAuthService("http://127.0.0.1:8287/callback")
	handler, errCh := service.MakeOauthCallbackHandler()

	req := httptest.NewRequest(http.MethodGet, "/callback?state=wrong-state&code=test-code", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	default:
		t.Fatal("expected error on errCh")
	}
}

func TestMakeOauthCallbackHandlerRejectsSpotifyError(t *testing.T) {
	service := NewAuthService("http://127.0.0.1:8287/callback")
	handler, errCh := service.MakeOauthCallbackHandler()

	req := httptest.NewRequest(http.MethodGet, "/callback?state="+service.authConfig.state+"&error=access_denied&error_description=user_denied", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	default:
		t.Fatal("expected error on errCh")
	}
}
