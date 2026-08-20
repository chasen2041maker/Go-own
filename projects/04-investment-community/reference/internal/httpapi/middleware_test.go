package httpapi

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestSecurityHeadersProtectEveryAPIResponse(t *testing.T) {
	handler := NewRouter(nil, time.Second)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	for name, want := range map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"Cache-Control":           "no-store",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
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

func TestSecurityHeadersAndSensitiveLogRedaction(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	handler := SecurityHeaders(WithRequestID(Recover(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("synthetic failure")
	}))))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts?token=query-secret", strings.NewReader(`{"password":"body-secret"}`))
	request.Header.Set("Authorization", "Bearer header-secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	for _, secret := range []string{"query-secret", "body-secret", "header-secret", "Authorization", "password"} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("log contains sensitive value %q: %s", secret, logs.String())
		}
	}
	if !strings.Contains(logs.String(), "method=POST") || !strings.Contains(logs.String(), "path=/api/v1/posts") {
		t.Errorf("log omitted safe diagnostic context: %s", logs.String())
	}
}

func TestAccessLogRecordsSafeFieldsForSuccessClientErrorAndPanic(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := SecurityHeaders(WithRequestID(AccessLog(logger, Recover(logger, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/ok":
			request.Pattern = "GET /ok"
			WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
		case "/bad":
			request.Pattern = "POST /bad"
			WriteError(writer, request, http.StatusBadRequest, "invalid_request", "invalid", nil)
		default:
			request.Pattern = "POST /panic"
			panic("sentinel-panic-secret")
		}
	})))))

	for _, test := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/ok?token=sentinel-query", http.StatusOK},
		{http.MethodPost, "/bad?token=sentinel-query", http.StatusBadRequest},
		{http.MethodPost, "/panic?token=sentinel-query", http.StatusInternalServerError},
	} {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(`{"body":"sentinel-body"}`))
		request.Header.Set("Authorization", "Bearer sentinel-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("%s status = %d, want %d", test.path, response.Code, test.status)
		}
	}

	output := logs.String()
	for _, required := range []string{`"request_id":`, `"method":`, `"route":`, `"status":200`, `"status":400`, `"status":500`, `"duration_ms":`} {
		if !strings.Contains(output, required) {
			t.Errorf("access log omitted %s: %s", required, output)
		}
	}
	for _, secret := range []string{"sentinel-query", "sentinel-body", "sentinel-token", "Authorization", "sentinel-panic-secret"} {
		if strings.Contains(output, secret) {
			t.Errorf("access log leaked %q: %s", secret, output)
		}
	}
}
