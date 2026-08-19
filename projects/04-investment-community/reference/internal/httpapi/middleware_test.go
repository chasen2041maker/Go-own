package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithRequestIDKeepsValidUpstreamID(t *testing.T) {
	handler := WithRequestID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got, want := RequestIDFromContext(request.Context()), "upstream.trace-42"; got != want {
			t.Errorf("context request ID = %q, want %q", got, want)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(HeaderRequestID, "upstream.trace-42")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if got, want := response.Header().Get(HeaderRequestID), "upstream.trace-42"; got != want {
		t.Errorf("X-Request-ID = %q, want %q", got, want)
	}
}

func TestWithRequestIDReplacesInvalidUpstreamID(t *testing.T) {
	handler := WithRequestID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(HeaderRequestID, strings.Repeat("x", 65))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	got := response.Header().Get(HeaderRequestID)
	if got == "" || len(got) > 64 || got == request.Header.Get(HeaderRequestID) {
		t.Fatalf("replacement X-Request-ID = %q", got)
	}
}

func TestRecoverDiscardsPartialResponseBeforeWritingInternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := WithRequestID(Recover(logger, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"partial":true}`))
		panic("failure after response started")
	})))
	request := httptest.NewRequest(http.MethodPost, "/panic-after-write", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "partial") {
		t.Fatalf("response retained partial body: %s", response.Body.String())
	}
	if got := decodeError(t, response).Code; got != "internal_error" {
		t.Fatalf("code = %q, want internal_error", got)
	}
}

func TestRecoverReturnsInternalErrorWithoutLeakingPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := WithRequestID(Recover(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("database password is secret")
	})))
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	errorObject := decodeError(t, response)
	if got, want := errorObject.Code, "internal_error"; got != want {
		t.Errorf("code = %q, want %q", got, want)
	}
	if strings.Contains(response.Body.String(), "database password") {
		t.Fatalf("response leaked panic detail: %s", response.Body.String())
	}
	if got := response.Header().Get(HeaderRequestID); got == "" || got != errorObject.RequestID {
		t.Errorf("X-Request-ID = %q, body request_id = %q", got, errorObject.RequestID)
	}
}
