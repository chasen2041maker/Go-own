package platform

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const DefaultAccessTokenTTL = 15 * time.Minute

// TokenManager 只签发用户 ID 与标准 JWT claims；角色永远从数据库读取。
type TokenManager struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time
}

func NewTokenManager(secret, issuer, audience string, ttl time.Duration) (*TokenManager, error) {
	if len(secret) < minimumJWTSecretBytes || strings.TrimSpace(secret) != secret {
		return nil, fmt.Errorf("token manager: secret must contain at least %d bytes and no surrounding whitespace", minimumJWTSecretBytes)
	}
	if issuer == "" || strings.TrimSpace(issuer) != issuer {
		return nil, fmt.Errorf("token manager: issuer is required without surrounding whitespace")
	}
	if audience == "" || strings.TrimSpace(audience) != audience {
		return nil, fmt.Errorf("token manager: audience is required without surrounding whitespace")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("token manager: ttl must be positive")
	}
	return &TokenManager{
		secret:   append([]byte(nil), secret...),
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
		now:      time.Now,
	}, nil
}

func (manager *TokenManager) Issue(userID int64) (string, time.Duration, error) {
	if userID <= 0 {
		return "", 0, fmt.Errorf("issue token: user id must be positive")
	}
	now := manager.now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    manager.issuer,
		Subject:   strconv.FormatInt(userID, 10),
		Audience:  jwt.ClaimStrings{manager.audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(manager.ttl)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(manager.secret)
	if err != nil {
		return "", 0, fmt.Errorf("issue token: %w", err)
	}
	return raw, manager.ttl, nil
}

func (manager *TokenManager) Verify(raw string) (int64, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing algorithm")
		}
		return manager.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(manager.issuer),
		jwt.WithAudience(manager.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(manager.now),
	)
	if err != nil || token == nil || !token.Valid {
		return 0, fmt.Errorf("verify token: invalid token")
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil || claims.IssuedAt.Time.After(manager.now()) {
		return 0, fmt.Errorf("verify token: required time claims are invalid")
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 || strconv.FormatInt(userID, 10) != claims.Subject {
		return 0, fmt.Errorf("verify token: subject must be a canonical positive int64")
	}
	return userID, nil
}
