package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

type seedPasswordHasher interface {
	Hash(string) (string, error)
}

type seedExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type seedUser struct {
	Email       string
	DisplayName string
	Password    string
	Role        domain.Role
}

type seedSecurity struct {
	Exchange string
	Code     string
	Name     string
	Status   domain.SecurityStatus
}

type seedCircle struct {
	Slug        string
	Name        string
	Description string
}

type seedData struct {
	Users      []seedUser
	Securities []seedSecurity
	Circles    []seedCircle
}

type preparedSeedUser struct {
	email        string
	passwordHash string
	displayName  string
	role         domain.Role
}

var seedSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func applySeed(ctx context.Context, executor seedExecutor, hasher seedPasswordHasher, data seedData) error {
	if executor == nil || hasher == nil {
		return errors.New("seed executor and password hasher are required")
	}
	preparedUsers := make([]preparedSeedUser, 0, len(data.Users))
	for _, user := range data.Users {
		email, err := domain.NormalizeEmail(user.Email)
		if err != nil {
			return fmt.Errorf("seed user email: %w", err)
		}
		displayName, err := domain.NormalizeDisplayName(user.DisplayName)
		if err != nil {
			return fmt.Errorf("seed user display name: %w", err)
		}
		if err := domain.ValidatePassword(user.Password); err != nil {
			return fmt.Errorf("seed user password: %w", err)
		}
		if user.Role != domain.RoleAdmin && user.Role != domain.RoleUser {
			return errors.New("seed user role must be admin or user")
		}
		passwordHash, err := hasher.Hash(user.Password)
		if err != nil {
			return fmt.Errorf("seed user password hash: %w", err)
		}
		if passwordHash == "" {
			return errors.New("seed user password hash is empty")
		}
		preparedUsers = append(preparedUsers, preparedSeedUser{
			email: email, passwordHash: passwordHash, displayName: displayName, role: user.Role,
		})
	}
	if err := validateCatalogSeed(data); err != nil {
		return err
	}

	for _, user := range preparedUsers {
		_, err := executor.ExecContext(ctx, `
INSERT INTO users (email, password_hash, display_name, role, status)
VALUES (?, ?, ?, ?, 'active')
ON DUPLICATE KEY UPDATE
    password_hash = VALUES(password_hash),
    display_name = VALUES(display_name),
    role = VALUES(role),
    status = 'active'`, user.email, user.passwordHash, user.displayName, user.role)
		if err != nil {
			return fmt.Errorf("upsert seed user: %w", err)
		}
	}
	for _, security := range data.Securities {
		_, err := executor.ExecContext(ctx, `
INSERT INTO securities (market, code, name, status)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE name = VALUES(name), status = VALUES(status)`,
			strings.TrimSpace(security.Exchange), strings.TrimSpace(security.Code),
			strings.TrimSpace(security.Name), security.Status)
		if err != nil {
			return fmt.Errorf("upsert seed security: %w", err)
		}
	}
	for _, circle := range data.Circles {
		// circles 还有 name 唯一键。只有 slug（Seed 的自然键）相同才允许更新；
		// 若命中“同名不同 slug”，把 NOT NULL name 设为 NULL 让事务明确失败，避免串改别的圈子。
		_, err := executor.ExecContext(ctx, `
INSERT INTO circles (slug, name, description, status)
VALUES (?, ?, ?, 'active')
ON DUPLICATE KEY UPDATE
    name = IF(slug = VALUES(slug), VALUES(name), NULL),
    description = IF(slug = VALUES(slug), VALUES(description), description),
    status = IF(slug = VALUES(slug), 'active', status)`,
			strings.TrimSpace(circle.Slug), strings.TrimSpace(circle.Name), strings.TrimSpace(circle.Description))
		if err != nil {
			return fmt.Errorf("upsert seed circle: %w", err)
		}
	}
	return nil
}

func validateCatalogSeed(data seedData) error {
	for _, security := range data.Securities {
		exchange := strings.TrimSpace(security.Exchange)
		code := strings.TrimSpace(security.Code)
		name := strings.TrimSpace(security.Name)
		if !validSeedText(exchange, 1, 16) || !validSeedText(code, 1, 16) || !validSeedText(name, 1, 80) {
			return errors.New("seed security fields exceed catalog limits")
		}
		if security.Status != domain.SecurityStatusActive && security.Status != domain.SecurityStatusInactive {
			return errors.New("seed security status must be active or inactive")
		}
	}
	for _, circle := range data.Circles {
		slug := strings.TrimSpace(circle.Slug)
		if !validSeedText(slug, 1, 64) || !seedSlugPattern.MatchString(slug) ||
			!validSeedText(strings.TrimSpace(circle.Name), 1, 80) ||
			!validSeedText(strings.TrimSpace(circle.Description), 0, 500) {
			return errors.New("seed circle fields exceed catalog limits")
		}
	}
	return nil
}

func validSeedText(value string, minimum, maximum int) bool {
	if !utf8.ValidString(value) {
		return false
	}
	length := utf8.RuneCountInString(value)
	return length >= minimum && length <= maximum
}
