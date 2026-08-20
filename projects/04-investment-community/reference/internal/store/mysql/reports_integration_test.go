//go:build integration

package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestReportDuplicateBeforeVisibilitySelfRuleAndAdminQueue(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newPostFixture(t, ctx, database)
	postService := mustIntegrationPostService(t, store)
	post := fixture.create(t, ctx, postService, "report-target", "举报目标", []int64{fixture.securityA})
	reporterID := insertIntegrationUser(t, ctx, database, "reporter-"+fixture.suffix)
	adminID := insertIntegrationUser(t, ctx, database, "admin-"+fixture.suffix)
	if _, err := database.ExecContext(ctx, "UPDATE users SET role='admin' WHERE id=?", adminID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM reports WHERE post_id=?", post.ID)
		_, _ = database.Exec("UPDATE posts SET visibility='visible' WHERE id=?", post.ID)
		deleteIntegrationIDs(database, "users", []int64{reporterID, adminID})
	})
	service, err := usecase.NewReportService(store)
	if err != nil {
		t.Fatal(err)
	}
	input := usecase.CreateReportInput{ReporterID: reporterID, TargetType: domain.ContentTypePost, TargetID: post.ID, Reason: domain.ReportReasonSpam, Details: " 虚构举报 "}
	created, err := service.CreateReport(ctx, input)
	if err != nil || created.Existing {
		t.Fatalf("CreateReport()=%#v,%v", created, err)
	}
	if _, err := database.ExecContext(ctx, "UPDATE posts SET visibility='hidden',moderation_version=moderation_version+1 WHERE id=?", post.ID); err != nil {
		t.Fatal(err)
	}
	replayed, err := service.CreateReport(ctx, input)
	if err != nil || !replayed.Existing || replayed.Report.ID != created.Report.ID {
		t.Fatalf("replayed=%#v,%v", replayed, err)
	}
	if _, err := database.ExecContext(ctx, "UPDATE posts SET visibility='visible' WHERE id=?", post.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateReport(ctx, usecase.CreateReportInput{ReporterID: fixture.userID, TargetType: domain.ContentTypePost, TargetID: post.ID, Reason: domain.ReportReasonSpam}); !errors.Is(err, domain.ErrSelfReportForbidden) {
		t.Fatalf("self report error=%v", err)
	}
	page, err := service.ListReports(ctx, usecase.ReportListInput{Admin: domain.User{ID: adminID, Role: domain.RoleAdmin, Status: domain.UserStatusActive}, Status: domain.ReportStatusPending, Limit: 10})
	if err != nil || len(page.Items) < 1 {
		t.Fatalf("ListReports()=%#v,%v", page, err)
	}
	found := false
	for _, item := range page.Items {
		if item.ID == created.Report.ID {
			found = true
			if item.Target.ID != post.ID || item.Target.Title == nil {
				t.Fatalf("admin report=%#v", item)
			}
		}
	}
	if !found {
		t.Fatal("created report missing from admin queue")
	}
}

func TestCreateReportLocksTargetBeforeCheckingExistingReport(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newPostFixture(t, ctx, database)
	post := fixture.create(t, ctx, mustIntegrationPostService(t, store), "report-lock-order", "锁序目标", []int64{fixture.securityA})
	reporterID := insertIntegrationUser(t, ctx, database, "report-lock-order-"+fixture.suffix)
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM reports WHERE post_id=?", post.ID)
		deleteIntegrationIDs(database, "users", []int64{reporterID})
	})
	service, err := usecase.NewReportService(store)
	if err != nil {
		t.Fatal(err)
	}
	input := usecase.CreateReportInput{ReporterID: reporterID, TargetType: domain.ContentTypePost, TargetID: post.ID, Reason: domain.ReportReasonSpam}
	created, err := service.CreateReport(ctx, input)
	if err != nil || created.Existing {
		t.Fatalf("initial report = %#v, error = %v", created, err)
	}

	blocker, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback()
	var lockedPostID int64
	if err := blocker.QueryRowContext(ctx, "SELECT id FROM posts WHERE id=? FOR UPDATE", post.ID).Scan(&lockedPostID); err != nil {
		t.Fatal(err)
	}

	blockedCtx, blockedCancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer blockedCancel()
	_, err = service.CreateReport(blockedCtx, input)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CreateReport while target locked error = %v, want context deadline", err)
	}
	if err := blocker.Rollback(); err != nil {
		t.Fatal(err)
	}

	replayed, err := service.CreateReport(ctx, input)
	if err != nil || !replayed.Existing || replayed.Report.ID != created.Report.ID {
		t.Fatalf("replayed after target unlock = %#v, error = %v", replayed, err)
	}
}
