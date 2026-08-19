package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthzDoesNotCallReadinessDependency(t *testing.T) {
	checkerCalled := false
	router := NewRouter(readinessFunc(func(context.Context) error {
		checkerCalled = true
		return errors.New("healthz must not ping the database")
	}), time.Second)

	response := serveRequest(router, http.MethodGet, "/healthz", "")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if checkerCalled {
		t.Fatal("GET /healthz called readiness dependency")
	}
	assertStatusBody(t, response, "ok")
	assertRequestID(t, response)
}

func TestReadyzReturnsOKWhenDependencyIsReady(t *testing.T) {
	checkerCalls := 0
	router := NewRouter(readinessFunc(func(ctx context.Context) error {
		checkerCalls++
		if _, ok := ctx.Deadline(); !ok {
			t.Error("readiness context has no deadline")
		}
		return nil
	}), time.Second)

	response := serveRequest(router, http.MethodGet, "/readyz", "")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if checkerCalls != 1 {
		t.Fatalf("readiness calls = %d, want 1", checkerCalls)
	}
	assertStatusBody(t, response, "ok")
}

func TestReadyzReturnsServiceUnavailableWithMatchingRequestID(t *testing.T) {
	router := NewRouter(readinessFunc(func(context.Context) error {
		return errors.New("dial secret-host: connection refused")
	}), time.Second)

	response := serveRequest(router, http.MethodGet, "/readyz", "trace-123")

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
	errorObject := decodeError(t, response)
	if got, want := errorObject.Code, "service_unavailable"; got != want {
		t.Errorf("error code = %q, want %q", got, want)
	}
	if got, want := errorObject.RequestID, "trace-123"; got != want {
		t.Errorf("body request_id = %q, want %q", got, want)
	}
	if got := response.Header().Get(HeaderRequestID); got != errorObject.RequestID {
		t.Errorf("X-Request-ID = %q, body request_id = %q", got, errorObject.RequestID)
	}
	if strings.Contains(errorObject.Message, "secret-host") {
		t.Fatalf("public error leaked dependency detail: %q", errorObject.Message)
	}
}

func TestRouterReturnsUnifiedErrorsForUnknownPathsAndMethods(t *testing.T) {
	router := NewRouter(nil, time.Second)
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
	}{
		{name: "unknown path", method: http.MethodGet, path: "/missing", wantStatus: http.StatusNotFound, wantCode: "not_found"},
		{name: "wrong method", method: http.MethodPost, path: "/healthz", wantStatus: http.StatusMethodNotAllowed, wantCode: "method_not_allowed"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serveRequest(router, test.method, test.path, "")
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if got := decodeError(t, response).Code; got != test.wantCode {
				t.Fatalf("error code = %q, want %q", got, test.wantCode)
			}
			assertRequestID(t, response)
		})
	}
}

type readinessFunc func(context.Context) error

func (function readinessFunc) PingContext(ctx context.Context) error {
	return function(ctx)
}

func serveRequest(handler http.Handler, method, path, requestID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	if requestID != "" {
		request.Header.Set(HeaderRequestID, requestID)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertStatusBody(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v; body = %s", err, response.Body.String())
	}
	if body.Status != want {
		t.Errorf("body status = %q, want %q", body.Status, want)
	}
}

func assertRequestID(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got := response.Header().Get(HeaderRequestID); got == "" {
		t.Fatal("X-Request-ID is empty")
	}
}

func decodeError(t *testing.T, response *httptest.ResponseRecorder) ErrorObject {
	t.Helper()
	var envelope ErrorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error response: %v; body = %s", err, response.Body.String())
	}
	return envelope.Error
}
