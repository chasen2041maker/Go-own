package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerGETReturnsHealthJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var response struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "ok" {
		t.Errorf("status body = %q, want ok", response.Status)
	}
}

func TestHandlerGETReturnsRequestID(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewHandler().ServeHTTP(recorder, request)

	if got := strings.TrimSpace(recorder.Header().Get("X-Request-ID")); got == "" {
		t.Fatal("X-Request-ID is empty")
	}
}

func TestHandlerRejectsOtherMethods(t *testing.T) {
	methods := []string{
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(method, "/healthz", nil)

			NewHandler().ServeHTTP(recorder, request)

			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
			}
			if got := recorder.Header().Get("Allow"); got != http.MethodGet {
				t.Errorf("Allow = %q, want %q", got, http.MethodGet)
			}
		})
	}
}
