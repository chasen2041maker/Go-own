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
	if len(files[0].Checksum) != 64 {
		t.Fatalf("checksum length = %d, want 64 hex characters", len(files[0].Checksum))
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

func TestMigrationChecksumIsIndependentOfCheckoutLineEndings(t *testing.T) {
	lf := migrationChecksum([]byte("CREATE TABLE example (id BIGINT);\n"))
	crlf := migrationChecksum([]byte("CREATE TABLE example (id BIGINT);\r\n"))
	cr := migrationChecksum([]byte("CREATE TABLE example (id BIGINT);\r"))
	if lf != crlf || lf != cr {
		t.Fatalf("line ending changed checksum: lf=%q crlf=%q cr=%q", lf, crlf, cr)
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

func TestInitialMigrationTracksModerationGeneration(t *testing.T) {
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
		if !strings.Contains(remaining[:end], "moderation_version BIGINT NOT NULL DEFAULT 1") {
			t.Errorf("%s must track a moderation generation for ABA-safe restore", table)
		}
	}
}

func TestInitialMigrationEncodesProtocolIntegrityConstraints(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() error = %v", err)
	}
	for _, required := range []string{
		"idempotency_key VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NULL",
		"request_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL",
		"CONSTRAINT chk_posts_idempotency_pair",
		"CONSTRAINT chk_comments_idempotency_pair",
		"CONSTRAINT chk_reports_state_shape",
		"KEY idx_reports_post_pending (post_id, status, id)",
		"KEY idx_reports_comment_pending (comment_id, status, id)",
		"CONSTRAINT chk_notifications_shape",
		"CONSTRAINT chk_admin_audit_target",
		"UNIQUE KEY uq_admin_audit_report (report_id)",
		"KEY idx_posts_global_feed (visibility, deleted_at, created_at DESC, id DESC)",
		"KEY idx_comments_list (post_id, visibility, deleted_at, created_at, id)",
		"KEY idx_notifications_timeline (user_id, created_at DESC, id DESC)",
		"code VARCHAR(16) NOT NULL",
	} {
		if !strings.Contains(files[0].SQL, required) {
			t.Errorf("initial migration missing %q", required)
		}
	}
}
