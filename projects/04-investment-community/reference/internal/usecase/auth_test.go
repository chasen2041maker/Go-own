package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func TestRegisterNormalizesFieldsHashesPasswordAndIssuesToken(t *testing.T) {
	var stored domain.User
	users := &fakeUsers{create: func(_ context.Context, user domain.User) (domain.User, error) {
		stored = user
		user.ID = 7
		return user, nil
	}}
	passwords := fakePasswords{hash: func(password string) (string, error) {
		if password != "password1234" {
			t.Fatalf("Hash() password = %q", password)
		}
		return "$bcrypt-hash", nil
	}}
	tokens := fakeTokens{issue: func(userID int64) (string, time.Duration, error) {
		if userID != 7 {
			t.Fatalf("Issue() userID = %d, want 7", userID)
		}
		return "access-token", 15 * time.Minute, nil
	}}
	service := mustAuthService(t, users, passwords, tokens)

	result, err := service.Register(context.Background(), RegisterInput{
		Email:       " Learner@Example.COM ",
		DisplayName: " 学习者 ",
		Password:    "password1234",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got, want := stored.Email, "learner@example.com"; got != want {
		t.Errorf("stored email = %q, want %q", got, want)
	}
	if got, want := stored.DisplayName, "学习者"; got != want {
		t.Errorf("stored display name = %q, want %q", got, want)
	}
	if stored.PasswordHash != "$bcrypt-hash" || stored.PasswordHash == "password1234" {
		t.Errorf("stored password hash = %q", stored.PasswordHash)
	}
	if stored.Role != domain.RoleUser || stored.Status != domain.UserStatusActive {
		t.Errorf("stored role/status = %q/%q, want user/active", stored.Role, stored.Status)
	}
	if result.AccessToken != "access-token" || result.ExpiresIn != 15*time.Minute || result.User.ID != 7 {
		t.Errorf("Register() result = %#v", result)
	}
	if result.User.PasswordHash != "" {
		t.Fatal("Register() returned password hash across the usecase boundary")
	}
}

func TestRegisterPreservesStableDuplicateEmailError(t *testing.T) {
	users := &fakeUsers{create: func(context.Context, domain.User) (domain.User, error) {
		return domain.User{}, domain.ErrEmailTaken
	}}
	service := mustAuthService(t, users, fakePasswords{hash: func(string) (string, error) {
		return "$bcrypt-hash", nil
	}}, fakeTokens{})

	_, err := service.Register(context.Background(), RegisterInput{
		Email: "learner@example.com", DisplayName: "学习者", Password: "password1234",
	})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("Register() error = %v, want ErrEmailTaken", err)
	}
}

func TestLoginUsesSameErrorForMissingEmailWrongPasswordAndDisabledUser(t *testing.T) {
	tests := []struct {
		name   string
		lookup func(context.Context, string) (domain.User, error)
		verify func(string, string) error
	}{
		{
			name: "missing email",
			lookup: func(context.Context, string) (domain.User, error) {
				return domain.User{}, domain.ErrUserNotFound
			},
		},
		{
			name: "wrong password",
			lookup: func(context.Context, string) (domain.User, error) {
				return activeUser(8, domain.RoleUser), nil
			},
			verify: func(string, string) error { return errors.New("mismatch") },
		},
		{
			name: "disabled user",
			lookup: func(context.Context, string) (domain.User, error) {
				user := activeUser(8, domain.RoleUser)
				user.Status = domain.UserStatusDisabled
				return user, nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := &fakeUsers{byEmail: test.lookup}
			passwords := fakePasswords{verify: test.verify}
			service := mustAuthService(t, users, passwords, fakeTokens{})

			_, err := service.Login(context.Background(), LoginInput{Email: "learner@example.com", Password: "wrong"})
			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestLoginRunsPasswordVerificationWhenEmailDoesNotExist(t *testing.T) {
	verifyCalls := 0
	users := &fakeUsers{byEmail: func(context.Context, string) (domain.User, error) {
		return domain.User{}, domain.ErrUserNotFound
	}}
	passwords := fakePasswords{verify: func(hash, _ string) error {
		verifyCalls++
		if hash == "" {
			t.Fatal("Verify() received empty dummy hash")
		}
		return errors.New("mismatch")
	}}
	service := mustAuthService(t, users, passwords, fakeTokens{})

	_, err := service.Login(context.Background(), LoginInput{Email: "missing@example.com", Password: "wrong"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if verifyCalls != 1 {
		t.Fatalf("Verify() calls = %d, want 1", verifyCalls)
	}
}

func TestLoginReturnsCurrentDatabaseUserAndToken(t *testing.T) {
	user := activeUser(9, domain.RoleAdmin)
	users := &fakeUsers{byEmail: func(_ context.Context, email string) (domain.User, error) {
		if email != "learner@example.com" {
			t.Fatalf("FindUserByEmail() email = %q", email)
		}
		return user, nil
	}}
	passwords := fakePasswords{verify: func(hash, password string) error {
		if hash != user.PasswordHash || password != "password1234" {
			t.Fatalf("Verify() = (%q, %q)", hash, password)
		}
		return nil
	}}
	tokens := fakeTokens{issue: func(id int64) (string, time.Duration, error) {
		return "access-token", time.Minute, nil
	}}
	service := mustAuthService(t, users, passwords, tokens)

	result, err := service.Login(context.Background(), LoginInput{
		Email: " LEARNER@example.com ", Password: "password1234",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.Role != domain.RoleAdmin || result.AccessToken != "access-token" {
		t.Fatalf("Login() result = %#v", result)
	}
	if result.User.PasswordHash != "" {
		t.Fatal("Login() returned password hash across the usecase boundary")
	}
}

func TestAuthenticateTreatsTokenAsUserLocatorAndReloadsDatabaseRole(t *testing.T) {
	users := &fakeUsers{byID: func(_ context.Context, id int64) (domain.User, error) {
		if id != 42 {
			t.Fatalf("FindUserByID() id = %d, want 42", id)
		}
		return activeUser(42, domain.RoleAdmin), nil
	}}
	tokens := fakeTokens{verify: func(raw string) (int64, error) {
		if raw != "valid-token" {
			t.Fatalf("Verify() token = %q", raw)
		}
		return 42, nil
	}}
	service := mustAuthService(t, users, fakePasswords{}, tokens)

	user, err := service.Authenticate(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if user.Role != domain.RoleAdmin {
		t.Fatalf("Authenticate() role = %q, want current DB admin role", user.Role)
	}
	if user.PasswordHash != "" {
		t.Fatal("Authenticate() placed password hash in request identity")
	}
}

func TestLoginValidatesLengthAndStillRunsDummyPasswordWork(t *testing.T) {
	verifyCalls := 0
	users := &fakeUsers{byEmail: func(context.Context, string) (domain.User, error) {
		t.Fatal("invalid login password must be rejected before user lookup")
		return domain.User{}, nil
	}}
	passwords := fakePasswords{verify: func(hash, _ string) error {
		verifyCalls++
		if hash == "" {
			t.Fatal("Verify() received empty dummy hash")
		}
		return errors.New("mismatch")
	}}
	service := mustAuthService(t, users, passwords, fakeTokens{})

	_, err := service.Login(context.Background(), LoginInput{Email: "learner@example.com", Password: ""})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || validation.Field != "password" {
		t.Fatalf("Login() error = %v, want password ValidationError", err)
	}
	if verifyCalls != 1 {
		t.Fatalf("Verify() calls = %d, want 1", verifyCalls)
	}
}

func TestAuthenticateRejectsInvalidTokenMissingAndDisabledUsers(t *testing.T) {
	tests := []struct {
		name   string
		verify func(string) (int64, error)
		byID   func(context.Context, int64) (domain.User, error)
	}{
		{name: "invalid token", verify: func(string) (int64, error) { return 0, errors.New("bad token") }},
		{name: "missing user", verify: func(string) (int64, error) { return 5, nil }, byID: func(context.Context, int64) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		}},
		{name: "disabled user", verify: func(string) (int64, error) { return 5, nil }, byID: func(context.Context, int64) (domain.User, error) {
			user := activeUser(5, domain.RoleAdmin)
			user.Status = domain.UserStatusDisabled
			return user, nil
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := mustAuthService(t, &fakeUsers{byID: test.byID}, fakePasswords{}, fakeTokens{verify: test.verify})
			if _, err := service.Authenticate(context.Background(), "token"); !errors.Is(err, domain.ErrUnauthenticated) {
				t.Fatalf("Authenticate() error = %v, want ErrUnauthenticated", err)
			}
		})
	}
}

func TestRequireRoleUsesCurrentUserRole(t *testing.T) {
	if err := RequireRole(activeUser(1, domain.RoleAdmin), domain.RoleAdmin); err != nil {
		t.Fatalf("RequireRole(admin) error = %v", err)
	}
	if err := RequireRole(activeUser(2, domain.RoleUser), domain.RoleAdmin); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("RequireRole(user) error = %v, want ErrForbidden", err)
	}
}

type fakeUsers struct {
	create  func(context.Context, domain.User) (domain.User, error)
	byEmail func(context.Context, string) (domain.User, error)
	byID    func(context.Context, int64) (domain.User, error)
}

func (users *fakeUsers) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	if users.create == nil {
		return domain.User{}, errors.New("unexpected CreateUser call")
	}
	return users.create(ctx, user)
}

func (users *fakeUsers) FindUserByEmail(ctx context.Context, email string) (domain.User, error) {
	if users.byEmail == nil {
		return domain.User{}, errors.New("unexpected FindUserByEmail call")
	}
	return users.byEmail(ctx, email)
}

func (users *fakeUsers) FindUserByID(ctx context.Context, id int64) (domain.User, error) {
	if users.byID == nil {
		return domain.User{}, errors.New("unexpected FindUserByID call")
	}
	return users.byID(ctx, id)
}

type fakePasswords struct {
	hash   func(string) (string, error)
	verify func(string, string) error
}

func (passwords fakePasswords) Hash(password string) (string, error) {
	if password == "invalid-credentials-placeholder" {
		return "$dummy-bcrypt-hash", nil
	}
	if passwords.hash == nil {
		return "", errors.New("unexpected Hash call")
	}
	return passwords.hash(password)
}

func (passwords fakePasswords) Verify(hash, password string) error {
	if passwords.verify == nil {
		return nil
	}
	return passwords.verify(hash, password)
}

type fakeTokens struct {
	issue  func(int64) (string, time.Duration, error)
	verify func(string) (int64, error)
}

func (tokens fakeTokens) Issue(userID int64) (string, time.Duration, error) {
	if tokens.issue == nil {
		return "", 0, errors.New("unexpected Issue call")
	}
	return tokens.issue(userID)
}

func (tokens fakeTokens) Verify(raw string) (int64, error) {
	if tokens.verify == nil {
		return 0, errors.New("unexpected Verify call")
	}
	return tokens.verify(raw)
}

func mustAuthService(t *testing.T, users UserRepository, passwords Passwords, tokens Tokens) *AuthService {
	t.Helper()
	service, err := NewAuthService(users, passwords, tokens)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	return service
}

func activeUser(id int64, role domain.Role) domain.User {
	return domain.User{
		ID:           id,
		Email:        "learner@example.com",
		PasswordHash: "$bcrypt-hash",
		DisplayName:  "学习者",
		Role:         role,
		Status:       domain.UserStatusActive,
	}
}
