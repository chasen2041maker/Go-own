//go:build integration

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/migrations"
)

func TestUserStoreRoundTripAndDuplicateEmail(t *testing.T) {
	dsn := os.Getenv("COMMUNITY_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNITY_TEST_DSN is not set")
	}
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := migrations.Apply(ctx, database); err != nil {
		t.Fatalf("migrations.Apply() error = %v", err)
	}
	store, err := New(database)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	email := fmt.Sprintf("integration-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = database.ExecContext(cleanupContext, "DELETE FROM users WHERE email = ?", email)
	})
	want := domain.User{
		Email:        email,
		PasswordHash: "$2a$10$integration-only-hash",
		DisplayName:  "集成测试用户",
		Role:         domain.RoleUser,
		Status:       domain.UserStatusActive,
	}
	created, err := store.CreateUser(ctx, want)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("CreateUser() ID = %d", created.ID)
	}

	byEmail, err := store.FindUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindUserByEmail() error = %v", err)
	}
	byID, err := store.FindUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindUserByID() error = %v", err)
	}
	if byEmail != byID || byID.Email != want.Email || byID.Role != domain.RoleUser {
		t.Fatalf("round trip users differ: byEmail=%#v byID=%#v", byEmail, byID)
	}
	if _, err := store.CreateUser(ctx, want); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("duplicate CreateUser() error = %v, want ErrEmailTaken", err)
	}
}
