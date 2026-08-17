package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	t.Run("GET returns a healthy JSON response", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		response := httptest.NewRecorder()

		newHandler().ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
		}
		if got, want := response.Header().Get("Content-Type"), "application/json"; got != want {
			t.Fatalf("Content-Type = %q, want %q", got, want)
		}

		var body struct {
			Status string `json:"status"`
		}
		decoder := json.NewDecoder(response.Body)
		if err := decoder.Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Status != "ok" {
			t.Fatalf("status = %q, want %q", body.Status, "ok")
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			t.Fatalf("response contains trailing JSON data: %v", err)
		}
	})

	t.Run("unsupported method returns method not allowed", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/health", nil)
		response := httptest.NewRecorder()

		newHandler().ServeHTTP(response, request)

		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status code = %d, want %d", response.Code, http.StatusMethodNotAllowed)
		}
	})
}
