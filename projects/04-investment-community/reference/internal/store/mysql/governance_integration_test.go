//go:build integration

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestConcurrentDecisionHasOneStateChange(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newGovernanceFixture(t, ctx, database, store)
	service := mustGovernanceService(t, store)

	start := make(chan struct{})
	errorsOut := make(chan error, 2)
	var group sync.WaitGroup
	for index, decision := range []domain.ReportDecision{domain.ReportDecisionIgnore, domain.ReportDecisionHide} {
		group.Add(1)
		go func(admin domain.User, choice domain.ReportDecision) {
			defer group.Done()
			<-start
			_, err := service.DecideReport(ctx, usecase.DecideReportInput{Admin: admin, ReportID: fixture.reportA, Decision: choice, Note: "并发决策", RequestID: "race-" + string(rune('a'+index))})
			errorsOut <- err
		}(fixture.admins[index], decision)
	}
	close(start)
	group.Wait()
	close(errorsOut)
	successes, conflicts := 0, 0
	for err := range errorsOut {
		if err == nil {
			successes++
		} else if errors.Is(err, domain.ErrReportAlreadyDecided) {
			conflicts++
		} else {
			t.Fatalf("decision error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("success/conflict = %d/%d", successes, conflicts)
	}
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM admin_audit_logs WHERE report_id=?", fixture.reportA, 1)
}

func TestConcurrentSameHideReturnsSuccessWithOneAuditAndNotification(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newGovernanceFixture(t, ctx, database, store)
	service := mustGovernanceService(t, store)

	start := make(chan struct{})
	results := make(chan domain.AdminReport, 2)
	errorsOut := make(chan error, 2)
	var group sync.WaitGroup
	for index, reportID := range []int64{fixture.reportA, fixture.reportB} {
		group.Add(1)
		go func(admin domain.User, id int64, requestID string) {
			defer group.Done()
			<-start
			result, err := service.DecideReport(ctx, usecase.DecideReportInput{Admin: admin, ReportID: id,
				Decision: domain.ReportDecisionHide, Note: "并发相同隐藏", RequestID: requestID})
			results <- result
			errorsOut <- err
		}(fixture.admins[index], reportID, "same-hide-"+string(rune('a'+index)))
	}
	close(start)
	group.Wait()
	close(results)
	close(errorsOut)
	for err := range errorsOut {
		if err != nil {
			t.Fatalf("same hide error = %v", err)
		}
	}
	for result := range results {
		if result.Target.Visibility != domain.VisibilityHidden || result.Decision == nil || *result.Decision != domain.ReportDecisionHide {
			t.Fatalf("same hide result = %#v", result)
		}
	}
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM reports WHERE post_id=? AND status='resolved' AND resolution_action='hide'", fixture.postID, 2)
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM admin_audit_logs WHERE post_id=? AND action='content_hidden'", fixture.postID, 1)
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM notifications WHERE post_id=? AND type='content_hidden'", fixture.postID, 1)
}

func TestAuditInsertFailureRollsBackGovernance(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newGovernanceFixture(t, ctx, database, store)

	_, err := store.DecideReport(ctx, domain.DecideReportParams{AdminID: fixture.admins[0].ID, ReportID: fixture.reportA,
		Decision: domain.ReportDecisionHide, Note: "必须回滚", RequestID: strings.Repeat("x", 65)})
	if err == nil {
		t.Fatal("DecideReport error = nil, want audit insert failure")
	}
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM reports WHERE id=? AND status='pending'", fixture.reportA, 1)
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM posts WHERE id=? AND visibility='visible' AND moderation_version=1", fixture.postID, 1)
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM notifications WHERE post_id=? AND type='content_hidden'", fixture.postID, 0)
}

func TestGovernanceRetryNotificationAndRestoreRejectABA(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newGovernanceFixture(t, ctx, database, store)
	service := mustGovernanceService(t, store)

	hidden, err := service.DecideReport(ctx, usecase.DecideReportInput{Admin: fixture.admins[0], ReportID: fixture.reportA,
		Decision: domain.ReportDecisionHide, Note: "隐藏", RequestID: "hide-first"})
	if err != nil || hidden.Target.Visibility != domain.VisibilityHidden || hidden.Target.ModerationVersion != 2 {
		t.Fatalf("hide = %#v, error = %v", hidden, err)
	}
	if _, err := service.DecideReport(ctx, usecase.DecideReportInput{Admin: fixture.admins[1], ReportID: fixture.reportA,
		Decision: domain.ReportDecisionHide, Note: "重试", RequestID: "hide-retry"}); err != nil {
		t.Fatalf("same decision retry: %v", err)
	}
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM admin_audit_logs WHERE report_id=?", fixture.reportA, 1)
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM notifications WHERE post_id=? AND type='content_hidden'", fixture.postID, 1)

	restored, err := service.RestoreContent(ctx, usecase.RestoreContentInput{Admin: fixture.admins[0], TargetType: domain.ContentTypePost,
		TargetID: fixture.postID, ExpectedModerationVersion: 2, RequestID: "restore-first"})
	if err != nil || restored.ModerationVersion != 3 || restored.Visibility != domain.VisibilityVisible {
		t.Fatalf("restore = %#v, error = %v", restored, err)
	}
	replayed, err := service.RestoreContent(ctx, usecase.RestoreContentInput{Admin: fixture.admins[1], TargetType: domain.ContentTypePost,
		TargetID: fixture.postID, ExpectedModerationVersion: 2, RequestID: "restore-retry"})
	if err != nil || replayed.ModerationVersion != 3 || !replayed.RestoredAt.Equal(restored.RestoredAt) {
		t.Fatalf("restore replay = %#v, error = %v", replayed, err)
	}
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM admin_audit_logs WHERE post_id=? AND action='content_restored'", fixture.postID, 1)
	assertGovernanceCount(t, ctx, database, "SELECT COUNT(*) FROM notifications WHERE post_id=? AND type='content_restored'", fixture.postID, 1)

	reporter := insertIntegrationUser(t, ctx, database, "governance-later-"+fixture.suffix)
	t.Cleanup(func() { deleteIntegrationIDs(database, "users", []int64{reporter}) })
	result, err := database.ExecContext(ctx, "INSERT INTO reports (reporter_id,post_id,reason_code,details) VALUES (?,?,'spam','再次举报')", reporter, fixture.postID)
	if err != nil {
		t.Fatal(err)
	}
	reportB, _ := result.LastInsertId()
	if _, err := service.DecideReport(ctx, usecase.DecideReportInput{Admin: fixture.admins[1], ReportID: reportB,
		Decision: domain.ReportDecisionHide, Note: "再次隐藏", RequestID: "hide-second"}); err != nil {
		t.Fatal(err)
	}
	_, err = service.RestoreContent(ctx, usecase.RestoreContentInput{Admin: fixture.admins[0], TargetType: domain.ContentTypePost,
		TargetID: fixture.postID, ExpectedModerationVersion: 2, RequestID: "restore-stale"})
	if !errors.Is(err, domain.ErrModerationVersionConflict) {
		t.Fatalf("stale restore error = %v", err)
	}
}

type governanceFixture struct {
	suffix           string
	postID           int64
	reportA, reportB int64
	admins           [2]domain.User
}

func newGovernanceFixture(t *testing.T, ctx context.Context, database *sql.DB, store *Store) governanceFixture {
	t.Helper()
	postFixture := newPostFixture(t, ctx, database)
	post := postFixture.create(t, ctx, mustIntegrationPostService(t, store), "governance", "治理目标", []int64{postFixture.securityA})
	fixture := governanceFixture{suffix: postFixture.suffix, postID: post.ID}
	reporters := []int64{insertIntegrationUser(t, ctx, database, "governance-reporter-a-"+fixture.suffix), insertIntegrationUser(t, ctx, database, "governance-reporter-b-"+fixture.suffix)}
	adminIDs := []int64{insertIntegrationUser(t, ctx, database, "governance-admin-a-"+fixture.suffix), insertIntegrationUser(t, ctx, database, "governance-admin-b-"+fixture.suffix)}
	for index, adminID := range adminIDs {
		if _, err := database.ExecContext(ctx, "UPDATE users SET role='admin' WHERE id=?", adminID); err != nil {
			t.Fatal(err)
		}
		fixture.admins[index] = domain.User{ID: adminID, DisplayName: "管理员", Role: domain.RoleAdmin, Status: domain.UserStatusActive}
	}
	for index, reporterID := range reporters {
		result, err := database.ExecContext(ctx, "INSERT INTO reports (reporter_id,post_id,reason_code,details) VALUES (?,?,'spam','虚构举报')", reporterID, post.ID)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := result.LastInsertId()
		if index == 0 {
			fixture.reportA = id
		} else {
			fixture.reportB = id
		}
	}
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM admin_audit_logs WHERE post_id=?", post.ID)
		_, _ = database.Exec("DELETE FROM notifications WHERE post_id=?", post.ID)
		_, _ = database.Exec("DELETE FROM reports WHERE post_id=?", post.ID)
		deleteIntegrationIDs(database, "users", append(reporters, adminIDs...))
	})
	return fixture
}

func mustGovernanceService(t *testing.T, store *Store) *usecase.GovernanceService {
	t.Helper()
	service, err := usecase.NewGovernanceService(store)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func assertGovernanceCount(t *testing.T, ctx context.Context, database *sql.DB, query string, id int64, want int) {
	t.Helper()
	var count int
	if err := database.QueryRowContext(ctx, query, id).Scan(&count); err != nil || count != want {
		t.Fatalf("count = %d, error = %v, want %d", count, err, want)
	}
}
