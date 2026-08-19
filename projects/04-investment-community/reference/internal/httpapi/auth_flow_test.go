package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/platform"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestAuthFlowRegisterLoginAndReloadCurrentUser(t *testing.T) {
	repository := newMemoryUsers()
	tokens, err := platform.NewTokenManager(
		"0123456789abcdef0123456789abcdef", "investment-community", "investment-community-api", time.Minute,
	)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	service, err := usecase.NewAuthService(repository, platform.NewPasswordHasher(), tokens)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	router := NewRouter(nil, time.Second, service)

	registered := performJSONRequest(router, "/api/v1/auth/register",
		`{"email":" Learner@Example.COM ","display_name":" 学习者 ","password":"password1234"}`)
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status = %d; body = %s", registered.Code, registered.Body.String())
	}
	var registerBody authResponse
	if err := json.NewDecoder(registered.Body).Decode(&registerBody); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	stored, err := repository.FindUserByID(context.Background(), registerBody.User.ID)
	if err != nil {
		t.Fatalf("FindUserByID() error = %v", err)
	}
	if stored.Email != "learner@example.com" || stored.DisplayName != "学习者" {
		t.Fatalf("stored normalized user = %#v", stored)
	}
	if stored.PasswordHash == "password1234" || platform.NewPasswordHasher().Verify(stored.PasswordHash, "password1234") != nil {
		t.Fatal("stored password is not a verifiable bcrypt hash")
	}

	missing := performJSONRequest(router, "/api/v1/auth/login",
		`{"email":"missing@example.com","password":"wrong"}`)
	wrong := performJSONRequest(router, "/api/v1/auth/login",
		`{"email":"learner@example.com","password":"wrong"}`)
	if missing.Code != http.StatusUnauthorized || wrong.Code != http.StatusUnauthorized {
		t.Fatalf("credential statuses = %d/%d", missing.Code, wrong.Code)
	}
	missingError := decodeError(t, missing)
	wrongError := decodeError(t, wrong)
	if missingError.Code != "invalid_credentials" || missingError.Code != wrongError.Code || missingError.Message != wrongError.Message {
		t.Fatalf("credential errors differ: missing=%#v wrong=%#v", missingError, wrongError)
	}

	loggedIn := performJSONRequest(router, "/api/v1/auth/login",
		`{"email":"LEARNER@example.com","password":"password1234"}`)
	if loggedIn.Code != http.StatusOK {
		t.Fatalf("login status = %d; body = %s", loggedIn.Code, loggedIn.Body.String())
	}
	var loginBody authResponse
	if err := json.NewDecoder(loggedIn.Body).Decode(&loginBody); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	repository.update(registerBody.User.ID, func(user *domain.User) { user.Role = domain.RoleAdmin })
	current := performMeRequest(router, loginBody.AccessToken)
	if current.Code != http.StatusOK {
		t.Fatalf("me status = %d; body = %s", current.Code, current.Body.String())
	}
	var currentUser privateUserResponse
	if err := json.NewDecoder(current.Body).Decode(&currentUser); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if currentUser.Role != domain.RoleAdmin {
		t.Fatalf("me role = %q, want freshly loaded admin", currentUser.Role)
	}

	repository.update(registerBody.User.ID, func(user *domain.User) { user.Status = domain.UserStatusDisabled })
	disabled := performMeRequest(router, loginBody.AccessToken)
	if disabled.Code != http.StatusUnauthorized || decodeError(t, disabled).Code != "unauthenticated" {
		t.Fatalf("disabled response = %d %s", disabled.Code, disabled.Body.String())
	}
}

func performMeRequest(handler http.Handler, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type memoryUsers struct {
	mu      sync.Mutex
	nextID  int64
	byID    map[int64]domain.User
	byEmail map[string]int64
}

func newMemoryUsers() *memoryUsers {
	return &memoryUsers{nextID: 1, byID: make(map[int64]domain.User), byEmail: make(map[string]int64)}
}

func (users *memoryUsers) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	users.mu.Lock()
	defer users.mu.Unlock()
	if _, exists := users.byEmail[user.Email]; exists {
		return domain.User{}, domain.ErrEmailTaken
	}
	user.ID = users.nextID
	users.nextID++
	users.byID[user.ID] = user
	users.byEmail[user.Email] = user.ID
	return user, nil
}

func (users *memoryUsers) FindUserByEmail(_ context.Context, email string) (domain.User, error) {
	users.mu.Lock()
	defer users.mu.Unlock()
	id, exists := users.byEmail[email]
	if !exists {
		return domain.User{}, domain.ErrUserNotFound
	}
	return users.byID[id], nil
}

func (users *memoryUsers) FindUserByID(_ context.Context, id int64) (domain.User, error) {
	users.mu.Lock()
	defer users.mu.Unlock()
	user, exists := users.byID[id]
	if !exists {
		return domain.User{}, domain.ErrUserNotFound
	}
	return user, nil
}

func (users *memoryUsers) update(id int64, change func(*domain.User)) {
	users.mu.Lock()
	defer users.mu.Unlock()
	user, exists := users.byID[id]
	if !exists {
		panic(errors.New("test user not found"))
	}
	change(&user)
	users.byID[id] = user
}
