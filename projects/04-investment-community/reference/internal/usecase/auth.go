// Package usecase 编排业务规则；它不读取 HTTP Header，也不知道 SQL 的存在。
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

type UserRepository interface {
	CreateUser(context.Context, domain.User) (domain.User, error)
	FindUserByEmail(context.Context, string) (domain.User, error)
	FindUserByID(context.Context, int64) (domain.User, error)
}

type Passwords interface {
	Hash(string) (string, error)
	Verify(string, string) error
}

type Tokens interface {
	Issue(int64) (string, time.Duration, error)
	Verify(string) (int64, error)
}

type RegisterInput struct {
	Email       string
	DisplayName string
	Password    string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	AccessToken string
	ExpiresIn   time.Duration
	User        domain.User
}

type AuthService struct {
	users     UserRepository
	passwords Passwords
	tokens    Tokens
	dummyHash string
}

const dummyPassword = "invalid-credentials-placeholder"

func NewAuthService(users UserRepository, passwords Passwords, tokens Tokens) (*AuthService, error) {
	if users == nil || passwords == nil || tokens == nil {
		return nil, errors.New("auth service dependencies are required")
	}
	// 启动时生成一次有效 bcrypt hash；不存在邮箱也走相同 Verify 路径，降低账号枚举时序差。
	dummyHash, err := passwords.Hash(dummyPassword)
	if err != nil {
		return nil, fmt.Errorf("auth service: create dummy password hash: %w", err)
	}
	return &AuthService{users: users, passwords: passwords, tokens: tokens, dummyHash: dummyHash}, nil
}

func (service *AuthService) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	email, err := domain.NormalizeEmail(input.Email)
	if err != nil {
		return AuthResult{}, err
	}
	displayName, err := domain.NormalizeDisplayName(input.DisplayName)
	if err != nil {
		return AuthResult{}, err
	}
	if err := domain.ValidatePassword(input.Password); err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := service.passwords.Hash(input.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("register: hash password: %w", err)
	}
	user, err := service.users.CreateUser(ctx, domain.User{
		Email:        email,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
		Role:         domain.RoleUser,
		Status:       domain.UserStatusActive,
	})
	if err != nil {
		return AuthResult{}, fmt.Errorf("register: create user: %w", err)
	}
	return service.issue(user)
}

func (service *AuthService) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	email, err := domain.NormalizeEmail(input.Email)
	if err != nil {
		return AuthResult{}, err
	}
	if err := domain.ValidateLoginPassword(input.Password); err != nil {
		// 即使协议长度无效也执行一次固定 bcrypt 工作，避免形成明显的快速失败侧信道。
		_ = service.passwords.Verify(service.dummyHash, input.Password)
		return AuthResult{}, err
	}
	user, err := service.users.FindUserByEmail(ctx, email)
	if errors.Is(err, domain.ErrUserNotFound) {
		_ = service.passwords.Verify(service.dummyHash, input.Password)
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, fmt.Errorf("login: find user: %w", err)
	}
	// 密码不匹配和账号停用使用同一个公开错误，避免泄露账号状态。
	if err := service.passwords.Verify(user.PasswordHash, input.Password); err != nil || user.Status != domain.UserStatusActive {
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	return service.issue(user)
}

func (service *AuthService) Authenticate(ctx context.Context, rawToken string) (domain.User, error) {
	userID, err := service.tokens.Verify(rawToken)
	if err != nil {
		return domain.User{}, domain.ErrUnauthenticated
	}
	// Token 的 sub 只是定位线索；每个请求都重新读取角色和停用状态，权限变更立即生效。
	user, err := service.users.FindUserByID(ctx, userID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, domain.ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("authenticate: find current user: %w", err)
	}
	if user.Status != domain.UserStatusActive {
		return domain.User{}, domain.ErrUnauthenticated
	}
	return user.WithoutPasswordHash(), nil
}

func RequireRole(user domain.User, role domain.Role) error {
	if user.Role != role {
		return domain.ErrForbidden
	}
	return nil
}

func (service *AuthService) issue(user domain.User) (AuthResult, error) {
	accessToken, expiresIn, err := service.tokens.Issue(user.ID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("issue access token: %w", err)
	}
	return AuthResult{AccessToken: accessToken, ExpiresIn: expiresIn, User: user.WithoutPasswordHash()}, nil
}
