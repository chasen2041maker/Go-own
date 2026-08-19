//go:build integration

package migrations

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestApplySerializesConcurrentRunners(t *testing.T) {
	database := openIntegrationDatabase(t)
	resetTestSchema(t, database)
	t.Cleanup(func() { resetTestSchema(t, database) })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	start := make(chan struct{})
	errors := make(chan error, 2)
	var runners sync.WaitGroup
	for range 2 {
		runners.Add(1)
		go func() {
			defer runners.Done()
			<-start
			errors <- Apply(ctx, database)
		}()
	}
	close(start)
	runners.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent Apply() error = %v", err)
		}
	}

	if err := Apply(ctx, database); err != nil {
		t.Fatalf("idempotent Apply() error = %v", err)
	}
	var migrations int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if migrations != 1 {
		t.Fatalf("schema_migrations rows = %d, want 1", migrations)
	}
}

func TestApplyCompletesACompatiblePartialInitialMigration(t *testing.T) {
	database := openIntegrationDatabase(t)
	resetTestSchema(t, database)
	t.Cleanup(func() { resetTestSchema(t, database) })
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() error = %v", err)
	}
	firstStatement := splitStatements(files[0].SQL)[0]
	if _, err := database.Exec(firstStatement); err != nil {
		t.Fatalf("create compatible partial table: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := Apply(ctx, database); err != nil {
		t.Fatalf("Apply() after compatible partial DDL error = %v", err)
	}
}

func TestApplyRejectsIncompatiblePreexistingTable(t *testing.T) {
	database := openIntegrationDatabase(t)
	resetTestSchema(t, database)
	t.Cleanup(func() { resetTestSchema(t, database) })
	if _, err := database.Exec("CREATE TABLE users (id BIGINT NOT NULL PRIMARY KEY)"); err != nil {
		t.Fatalf("create incompatible users table: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := Apply(ctx, database)
	if err == nil || !strings.Contains(err.Error(), "incompatible pre-existing tables") {
		t.Fatalf("Apply() error = %v, want incompatible pre-existing table error", err)
	}
}

func openIntegrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("COMMUNITY_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNITY_TEST_DSN is not set")
	}
	if os.Getenv("COMMUNITY_TEST_ALLOW_RESET") != "1" {
		t.Skip("COMMUNITY_TEST_ALLOW_RESET=1 is required because this test resets its dedicated schema")
	}
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func resetTestSchema(t *testing.T, database *sql.DB) {
	t.Helper()
	downSQL, err := migrationFS.ReadFile("001_initial.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	for _, statement := range splitStatements(string(downSQL)) {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("apply down statement: %v", err)
		}
	}
	if _, err := database.Exec("DROP TABLE IF EXISTS schema_migrations"); err != nil {
		t.Fatalf("drop schema_migrations: %v", err)
	}
}
