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

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
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

func TestMeRejectsRequestWithoutAccessToken(t *testing.T) {
	router := NewRouter(nil, time.Second)

	response := serveRequest(router, http.MethodGet, "/api/v1/me", "")

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
	if got, want := decodeError(t, response).Code, "unauthenticated"; got != want {
		t.Fatalf("error code = %q, want %q", got, want)
	}
}

func TestRegisterAppliesStrictJSONBoundary(t *testing.T) {
	router := NewRouter(nil, time.Second)
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantCode    string
	}{
		{
			name:       "missing content type",
			body:       `{}`,
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "unsupported_media_type",
		},
		{
			name:        "unknown field",
			contentType: "application/json",
			body:        `{"email":"learner@example.com","display_name":"学习者","password":"password123","admin":true}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "field names are case sensitive",
			contentType: "application/json",
			body:        `{"EMAIL":"learner@example.com","display_name":"学习者","password":"password123"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "trailing value",
			contentType: "application/json",
			body:        `{"email":"learner@example.com","display_name":"学习者","password":"password123"} {}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_json",
		},
		{
			name:        "top level null is not an object",
			contentType: "application/json",
			body:        `null`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "invalid UTF-8 is rejected",
			contentType: "application/json",
			body:        `{"email":"` + string([]byte{0xff}) + `","display_name":"学习者","password":"password1234"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_json",
		},
		{
			name:        "body over one MiB",
			contentType: "application/json",
			body:        `{"email":"` + strings.Repeat("a", (1<<20)+1) + `"}`,
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantCode:    "payload_too_large",
		},
		{
			name:        "malformed prefix still respects raw body limit",
			contentType: "application/json",
			body:        `{` + strings.Repeat("x", (1<<20)+1),
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantCode:    "payload_too_large",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if got := decodeError(t, response).Code; got != test.wantCode {
				t.Fatalf("error code = %q, want %q", got, test.wantCode)
			}
		})
	}
}

func TestRegisterRejectsInvalidAccountFields(t *testing.T) {
	application := &fakeAuthApplication{register: func(_ context.Context, input usecase.RegisterInput) (usecase.AuthResult, error) {
		if _, err := domain.NormalizeEmail(input.Email); err != nil {
			return usecase.AuthResult{}, err
		}
		if _, err := domain.NormalizeDisplayName(input.DisplayName); err != nil {
			return usecase.AuthResult{}, err
		}
		if err := domain.ValidatePassword(input.Password); err != nil {
			return usecase.AuthResult{}, err
		}
		return usecase.AuthResult{}, errors.New("unexpected valid input")
	}}
	router := NewRouter(nil, time.Second, application)
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{
			name:      "email must be one bare address",
			body:      `{"email":"Learner <learner@example.com>","display_name":"学习者","password":"password123"}`,
			wantField: "email",
		},
		{
			name:      "display name cannot become empty after trimming",
			body:      `{"email":"learner@example.com","display_name":" ","password":"password1234"}`,
			wantField: "display_name",
		},
		{
			name:      "password is at least twelve Unicode characters",
			body:      `{"email":"learner@example.com","display_name":"学习者","password":"12345678901"}`,
			wantField: "password",
		},
		{
			name:      "password respects bcrypt byte ceiling",
			body:      `{"email":"learner@example.com","display_name":"学习者","password":"` + strings.Repeat("p", 73) + `"}`,
			wantField: "password",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
			}
			problem := decodeError(t, response)
			if got, want := problem.Code, "validation_failed"; got != want {
				t.Fatalf("code = %q, want %q", got, want)
			}
			if len(problem.Details) != 1 || problem.Details[0].Field != test.wantField {
				t.Fatalf("details = %#v, want field %q", problem.Details, test.wantField)
			}
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
