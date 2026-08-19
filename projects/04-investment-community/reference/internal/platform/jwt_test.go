package platform

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testJWTSecret   = "0123456789abcdef0123456789abcdef"
	testJWTIssuer   = "investment-community"
	testJWTAudience = "investment-community-api"
)

func TestTokenManagerIssuesOnlyStandardClaimsAndVerifiesSubject(t *testing.T) {
	manager, err := NewTokenManager(testJWTSecret, testJWTIssuer, testJWTAudience, time.Minute)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	raw, expiresIn, err := manager.Issue(42)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if expiresIn != time.Minute {
		t.Fatalf("expiresIn = %s, want %s", expiresIn, time.Minute)
	}
	userID, err := manager.Verify(raw)
	if err != nil || userID != 42 {
		t.Fatalf("Verify() = (%d, %v), want (42, nil)", userID, err)
	}

	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return []byte(testJWTSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})); err != nil {
		t.Fatalf("parse issued token: %v", err)
	}
	for claim := range claims {
		switch claim {
		case "iss", "sub", "aud", "exp", "iat":
		default:
			t.Errorf("issued token contains non-standard or unnecessary claim %q", claim)
		}
	}
	if _, exists := claims["role"]; exists {
		t.Fatal("issued token contains role claim")
	}
}

func TestTokenManagerRejectsInvalidStandardClaimsAndAlgorithm(t *testing.T) {
	manager, err := NewTokenManager(testJWTSecret, testJWTIssuer, testJWTAudience, time.Minute)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	now := time.Now().UTC()
	missingExpiration := registeredClaims("42", testJWTIssuer, testJWTAudience, now.Add(time.Minute), now)
	missingExpiration.ExpiresAt = nil
	missingIssuedAt := registeredClaims("42", testJWTIssuer, testJWTAudience, now.Add(time.Minute), now)
	missingIssuedAt.IssuedAt = nil

	tests := []struct {
		name   string
		method jwt.SigningMethod
		claims jwt.RegisteredClaims
	}{
		{
			name:   "expired",
			method: jwt.SigningMethodHS256,
			claims: registeredClaims("42", testJWTIssuer, testJWTAudience, now.Add(-time.Minute), now.Add(-2*time.Minute)),
		},
		{
			name:   "future issued at",
			method: jwt.SigningMethodHS256,
			claims: registeredClaims("42", testJWTIssuer, testJWTAudience, now.Add(time.Minute), now.Add(time.Minute)),
		},
		{
			name:   "missing expiration",
			method: jwt.SigningMethodHS256,
			claims: missingExpiration,
		},
		{
			name:   "missing issued at",
			method: jwt.SigningMethodHS256,
			claims: missingIssuedAt,
		},
		{
			name:   "wrong issuer",
			method: jwt.SigningMethodHS256,
			claims: registeredClaims("42", "other", testJWTAudience, now.Add(time.Minute), now),
		},
		{
			name:   "wrong audience",
			method: jwt.SigningMethodHS256,
			claims: registeredClaims("42", testJWTIssuer, "other", now.Add(time.Minute), now),
		},
		{
			name:   "wrong algorithm",
			method: jwt.SigningMethodHS384,
			claims: registeredClaims("42", testJWTIssuer, testJWTAudience, now.Add(time.Minute), now),
		},
		{
			name:   "noncanonical subject",
			method: jwt.SigningMethodHS256,
			claims: registeredClaims("+42", testJWTIssuer, testJWTAudience, now.Add(time.Minute), now),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw, err := jwt.NewWithClaims(test.method, test.claims).SignedString([]byte(testJWTSecret))
			if err != nil {
				t.Fatalf("SignedString() error = %v", err)
			}
			if _, err := manager.Verify(raw); err == nil {
				t.Fatal("Verify() error = nil")
			}
		})
	}
}

func registeredClaims(subject, issuer, audience string, expiresAt, issuedAt time.Time) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   subject,
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(issuedAt),
	}
}

func TestTokenManagerRejectsNonPositiveSubject(t *testing.T) {
	manager, err := NewTokenManager(testJWTSecret, testJWTIssuer, testJWTAudience, time.Minute)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	now := time.Now().UTC()
	for _, id := range []int64{0, -1} {
		claims := registeredClaims(strconv.FormatInt(id, 10), testJWTIssuer, testJWTAudience, now.Add(time.Minute), now)
		raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
		if err != nil {
			t.Fatalf("SignedString() error = %v", err)
		}
		if _, err := manager.Verify(raw); err == nil {
			t.Fatalf("Verify(subject=%d) error = nil", id)
		}
	}
}
