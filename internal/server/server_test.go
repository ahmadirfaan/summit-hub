package server

import (
	"net/http/httptest"
	"testing"

	"backend-summithub/internal/config"
)

func TestHealthRoute(t *testing.T) {
	s := NewServer(config.Config{JWTSecret: "secret", ServerPort: ":0"}, nil, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := s.App.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 status")
	}
}
