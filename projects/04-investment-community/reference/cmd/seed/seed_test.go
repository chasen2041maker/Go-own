package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func TestApplySeedHashesBothUserPasswordsAndWritesOnlyFictionalCatalog(t *testing.T) {
	data := seedData{
		Users: []seedUser{
			{Email: "admin@fiction.example.test", DisplayName: "虚构管理员", Password: "admin-password-123", Role: domain.RoleAdmin},
			{Email: "learner@fiction.example.test", DisplayName: "虚构学习者", Password: "user-password-1234", Role: domain.RoleUser},
		},
		Securities: []seedSecurity{{Exchange: "XTEST", Code: "MOON", Name: "月湾材料", Status: domain.SecurityStatusActive}},
		Circles:    []seedCircle{{Slug: "fiction-lab", Name: "虚构研究室", Description: "只讨论虚构案例"}},
	}
	var hashed []string
	hasher := fakeSeedHasher{hash: func(password string) (string, error) {
		hashed = append(hashed, password)
		return "$hashed$" + password, nil
	}}
	executor := &fakeSeedExecutor{}

	if err := applySeed(context.Background(), executor, hasher, data); err != nil {
		t.Fatalf("applySeed() error = %v", err)
	}
	if len(hashed) != 2 || hashed[0] != data.Users[0].Password || hashed[1] != data.Users[1].Password {
		t.Fatalf("Hash() calls = %#v", hashed)
	}
	if len(executor.calls) != 4 {
		t.Fatalf("ExecContext() calls = %d, want 4", len(executor.calls))
	}
	for _, call := range executor.calls {
		for _, argument := range call.arguments {
			if argument == data.Users[0].Password || argument == data.Users[1].Password {
				t.Fatalf("database argument leaked plaintext password in query %q", call.query)
			}
		}
	}
	if !strings.Contains(executor.calls[0].arguments[1].(string), "$hashed$") ||
		!strings.Contains(executor.calls[1].arguments[1].(string), "$hashed$") {
		t.Fatalf("user inserts did not receive hashes: %#v", executor.calls[:2])
	}
}

func TestApplySeedStopsBeforeWritesWhenPasswordValidationOrHashingFails(t *testing.T) {
	tests := []struct {
		name   string
		user   seedUser
		hasher fakeSeedHasher
	}{
		{
			name: "invalid password",
			user: seedUser{Email: "admin@fiction.example.test", DisplayName: "虚构管理员", Password: "short", Role: domain.RoleAdmin},
			hasher: fakeSeedHasher{hash: func(string) (string, error) {
				t.Fatal("invalid password must not be hashed")
				return "", nil
			}},
		},
		{
			name: "hash failure",
			user: seedUser{Email: "admin@fiction.example.test", DisplayName: "虚构管理员", Password: "admin-password-123", Role: domain.RoleAdmin},
			hasher: fakeSeedHasher{hash: func(string) (string, error) {
				return "", errors.New("hasher unavailable")
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &fakeSeedExecutor{}
			err := applySeed(context.Background(), executor, test.hasher, seedData{Users: []seedUser{test.user}})
			if err == nil {
				t.Fatal("applySeed() error = nil")
			}
			if len(executor.calls) != 0 {
				t.Fatalf("ExecContext() calls = %d, want 0", len(executor.calls))
			}
		})
	}
}

type fakeSeedHasher struct {
	hash func(string) (string, error)
}

func (hasher fakeSeedHasher) Hash(password string) (string, error) {
	return hasher.hash(password)
}

type seedExecCall struct {
	query     string
	arguments []any
}

type fakeSeedExecutor struct {
	calls []seedExecCall
}

func (executor *fakeSeedExecutor) ExecContext(_ context.Context, query string, arguments ...any) (sql.Result, error) {
	executor.calls = append(executor.calls, seedExecCall{query: query, arguments: arguments})
	return driver.RowsAffected(1), nil
}
