package migrations

import (
	"strings"
	"testing"
)

func TestFilesReturnsInitialMigrationInVersionOrder(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("len(Files()) = %d, want 1", len(files))
	}
	if got, want := files[0].Version, int64(1); got != want {
		t.Fatalf("version = %d, want %d", got, want)
	}
	if got, want := files[0].Name, "initial"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}

	for _, table := range []string{
		"users", "circles", "circle_memberships", "securities", "posts",
		"post_securities", "comments", "reports", "notifications", "admin_audit_logs",
	} {
		if !strings.Contains(files[0].SQL, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Errorf("initial migration does not create %s", table)
		}
	}
}

func TestFilesReturnsDefensiveCopies(t *testing.T) {
	first, err := Files()
	if err != nil {
		t.Fatalf("first Files() error = %v", err)
	}
	first[0].SQL = "changed by caller"

	second, err := Files()
	if err != nil {
		t.Fatalf("second Files() error = %v", err)
	}
	if second[0].SQL == "changed by caller" {
		t.Fatal("Files() exposed mutable package state")
	}
}

func TestInitialMigrationKeepsModerationSeparateFromAuthorDeletion(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() error = %v", err)
	}
	for _, table := range []string{"posts", "comments"} {
		start := strings.Index(files[0].SQL, "CREATE TABLE IF NOT EXISTS "+table)
		if start < 0 {
			t.Fatalf("missing %s table", table)
		}
		remaining := files[0].SQL[start:]
		end := strings.Index(remaining, ") ENGINE=InnoDB")
		if end < 0 {
			t.Fatalf("cannot locate end of %s table", table)
		}
		definition := remaining[:end]
		if !strings.Contains(definition, "visibility ENUM('visible', 'hidden')") {
			t.Errorf("%s must store governance visibility independently", table)
		}
		if !strings.Contains(definition, "deleted_at DATETIME(6) NULL") {
			t.Errorf("%s must store author deletion independently", table)
		}
		if strings.Contains(definition, "'deleted'") {
			t.Errorf("%s visibility must not encode author deletion", table)
		}
	}
}
