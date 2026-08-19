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

func TestRegisterReturnsCreatedAuthResponseWithoutPasswordHash(t *testing.T) {
	application := &fakeAuthApplication{register: func(_ context.Context, input usecase.RegisterInput) (usecase.AuthResult, error) {
		if input.Email != "learner@example.com" || input.DisplayName != "学习者" || input.Password != "password1234" {
			t.Fatalf("Register() input = %#v", input)
		}
		return authResult(12, domain.RoleUser), nil
	}}
	router := NewRouter(nil, time.Second, application)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(
		`{"email":"learner@example.com","display_name":"学习者","password":"password1234"}`,
	))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["access_token"] != "access-token" || body["token_type"] != "Bearer" || body["expires_in"] != float64(900) {
		t.Errorf("auth fields = %#v", body)
	}
	user, ok := body["user"].(map[string]any)
	if !ok || user["id"] != float64(12) || user["role"] != "user" {
		t.Fatalf("user = %#v", body["user"])
	}
	if _, exists := user["password_hash"]; exists {
		t.Fatal("response contains password_hash")
	}
}

func TestRegisterMapsDuplicateEmailAndValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "duplicate", err: domain.ErrEmailTaken, wantStatus: http.StatusConflict, wantCode: "email_taken"},
		{name: "validation", err: &domain.ValidationError{Field: "email", Reason: "必须是单个裸邮箱地址"}, wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := &fakeAuthApplication{register: func(context.Context, usecase.RegisterInput) (usecase.AuthResult, error) {
				return usecase.AuthResult{}, test.err
			}}
			router := NewRouter(nil, time.Second, application)
			response := performJSONRequest(router, "/api/v1/auth/register",
				`{"email":"learner@example.com","display_name":"学习者","password":"password1234"}`)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if got := decodeError(t, response).Code; got != test.wantCode {
				t.Fatalf("code = %q, want %q", got, test.wantCode)
			}
		})
	}
}

func TestLoginMapsAllCredentialFailuresToSamePublicError(t *testing.T) {
	application := &fakeAuthApplication{login: func(context.Context, usecase.LoginInput) (usecase.AuthResult, error) {
		return usecase.AuthResult{}, domain.ErrInvalidCredentials
	}}
	router := NewRouter(nil, time.Second, application)

	response := performJSONRequest(router, "/api/v1/auth/login",
		`{"email":"missing@example.com","password":"wrong"}`)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
	problem := decodeError(t, response)
	if problem.Code != "invalid_credentials" || problem.Message != "邮箱或密码不正确" {
		t.Fatalf("error = %#v", problem)
	}
}

func TestLoginReturnsOKAuthResponse(t *testing.T) {
	application := &fakeAuthApplication{login: func(_ context.Context, input usecase.LoginInput) (usecase.AuthResult, error) {
		if input.Email != "learner@example.com" || input.Password != "password1234" {
			t.Fatalf("Login() input = %#v", input)
		}
		return authResult(18, domain.RoleUser), nil
	}}
	router := NewRouter(nil, time.Second, application)

	response := performJSONRequest(router, "/api/v1/auth/login",
		`{"email":"learner@example.com","password":"password1234"}`)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestMeAuthenticatesBearerTokenAndReturnsCurrentDatabaseUser(t *testing.T) {
	application := &fakeAuthApplication{authenticate: func(_ context.Context, raw string) (domain.User, error) {
		if raw != "current-token" {
			t.Fatalf("Authenticate() token = %q", raw)
		}
		return domain.User{
			ID: 42, Email: "admin@example.com", PasswordHash: "must-not-leak",
			DisplayName: "管理员", Role: domain.RoleAdmin, Status: domain.UserStatusActive,
		}, nil
	}}
	router := NewRouter(nil, time.Second, application)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", "Bearer current-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "must-not-leak") {
		t.Fatalf("response leaked password hash: %s", response.Body.String())
	}
	var user struct {
		ID     int64             `json:"id"`
		Role   domain.Role       `json:"role"`
		Status domain.UserStatus `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 42 || user.Role != domain.RoleAdmin || user.Status != domain.UserStatusActive {
		t.Fatalf("user = %#v", user)
	}
}

func TestMeRejectsMalformedExpiredAndUnexpectedAuthFailures(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		authErr    error
		wantStatus int
		wantCode   string
	}{
		{name: "malformed header", header: "Basic secret", wantStatus: http.StatusUnauthorized, wantCode: "unauthenticated"},
		{name: "expired or disabled", header: "Bearer expired", authErr: domain.ErrUnauthenticated, wantStatus: http.StatusUnauthorized, wantCode: "unauthenticated"},
		{name: "store failure", header: "Bearer valid", authErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := &fakeAuthApplication{authenticate: func(context.Context, string) (domain.User, error) {
				return domain.User{}, test.authErr
			}}
			router := NewRouter(nil, time.Second, application)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if got := decodeError(t, response).Code; got != test.wantCode {
				t.Fatalf("code = %q, want %q", got, test.wantCode)
			}
		})
	}
}

func TestRequireAdminUsesAuthenticatedDatabaseRole(t *testing.T) {
	for _, test := range []struct {
		name       string
		role       domain.Role
		wantStatus int
	}{
		{name: "admin", role: domain.RoleAdmin, wantStatus: http.StatusNoContent},
		{name: "user", role: domain.RoleUser, wantStatus: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			application := &fakeAuthApplication{authenticate: func(context.Context, string) (domain.User, error) {
				return domain.User{ID: 5, Role: test.role, Status: domain.UserStatusActive}, nil
			}}
			handler := WithRequestID(authenticate(application, RequireAdmin(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusNoContent)
			}))))
			request := httptest.NewRequest(http.MethodGet, "/admin-test", nil)
			request.Header.Set("Authorization", "Bearer current-token")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus == http.StatusForbidden && decodeError(t, response).Code != "forbidden" {
				t.Fatalf("body = %s", response.Body.String())
			}
		})
	}
}

type fakeAuthApplication struct {
	register     func(context.Context, usecase.RegisterInput) (usecase.AuthResult, error)
	login        func(context.Context, usecase.LoginInput) (usecase.AuthResult, error)
	authenticate func(context.Context, string) (domain.User, error)
}

func (application *fakeAuthApplication) Register(ctx context.Context, input usecase.RegisterInput) (usecase.AuthResult, error) {
	if application.register == nil {
		return usecase.AuthResult{}, errors.New("unexpected Register call")
	}
	return application.register(ctx, input)
}

func (application *fakeAuthApplication) Login(ctx context.Context, input usecase.LoginInput) (usecase.AuthResult, error) {
	if application.login == nil {
		return usecase.AuthResult{}, errors.New("unexpected Login call")
	}
	return application.login(ctx, input)
}

func (application *fakeAuthApplication) Authenticate(ctx context.Context, raw string) (domain.User, error) {
	if application.authenticate == nil {
		return domain.User{}, errors.New("unexpected Authenticate call")
	}
	return application.authenticate(ctx, raw)
}

func authResult(id int64, role domain.Role) usecase.AuthResult {
	return usecase.AuthResult{
		AccessToken: "access-token",
		ExpiresIn:   15 * time.Minute,
		User: domain.User{
			ID: id, Email: "learner@example.com", PasswordHash: "must-not-leak",
			DisplayName: "学习者", Role: role, Status: domain.UserStatusActive,
		},
	}
}

func performJSONRequest(handler http.Handler, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
