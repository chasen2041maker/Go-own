package platform

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher 把 bcrypt 隔离在平台层；用例只依赖它的 Hash/Verify 能力。
type PasswordHasher struct {
	cost int
}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{cost: bcrypt.DefaultCost}
}

func (hasher *PasswordHasher) Hash(password string) (string, error) {
	if length := len([]byte(password)); length < 8 || length > 72 {
		return "", fmt.Errorf("hash password: length must be between 8 and 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), hasher.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func (*PasswordHasher) Verify(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	return nil
}
