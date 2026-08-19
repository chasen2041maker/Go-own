//go:build integration

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestPostAndSecuritiesRollbackTogetherAndFilterPagination(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newPostFixture(t, ctx, database)
	service := mustIntegrationPostService(t, store)

	created := fixture.create(t, ctx, service, "post-atomic", "第一篇", []int64{fixture.securityA})
	_, err := service.CreatePost(ctx, usecase.CreatePostInput{UserID: fixture.userID, CircleID: fixture.circleID,
		IdempotencyKey: "post-rollback-" + fixture.suffix, Title: "不能留下", Body: "正文", SecurityIDs: []int64{fixture.inactiveSecurity}})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("inactive CreatePost() error = %v", err)
	}
	var partial int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE author_id=? AND idempotency_key=?", fixture.userID, "post-rollback-"+fixture.suffix).Scan(&partial); err != nil || partial != 0 {
		t.Fatalf("partial post count = %d, error = %v", partial, err)
	}

	second := fixture.create(t, ctx, service, "post-page-2", "第二篇", []int64{fixture.securityA, fixture.securityB})
	third := fixture.create(t, ctx, service, "post-page-3", "第三篇", []int64{fixture.securityA})
	tied := time.Date(2099, 8, 19, 12, 0, 0, 123000000, time.UTC)
	if _, err := database.ExecContext(ctx, "UPDATE posts SET created_at=?, updated_at=? WHERE id IN (?,?,?)", tied, tied, created.ID, second.ID, third.ID); err != nil {
		t.Fatal(err)
	}
	firstPage, err := service.ListPosts(ctx, usecase.PostListInput{UserID: fixture.userID, CircleID: fixture.circleID, SecurityID: fixture.securityA, Limit: 2})
	if err != nil || len(firstPage.Items) != 2 || firstPage.Items[0].ID != third.ID || firstPage.Items[1].ID != second.ID || firstPage.Next == nil {
		t.Fatalf("first page = %#v, %v", firstPage, err)
	}
	secondPage, err := service.ListPosts(ctx, usecase.PostListInput{UserID: fixture.userID, CircleID: fixture.circleID, SecurityID: fixture.securityA, Limit: 2, After: firstPage.Next})
	if err != nil || len(secondPage.Items) != 1 || secondPage.Items[0].ID != created.ID {
		t.Fatalf("second page = %#v, %v", secondPage, err)
	}

	newTitle := "不应提交"
	badTags := []int64{fixture.inactiveSecurity}
	_, err = service.UpdatePost(ctx, usecase.UpdatePostInput{UserID: fixture.userID, PostID: created.ID, Version: 1, Title: &newTitle, SecurityIDs: &badTags})
	if !errors.As(err, &validation) {
		t.Fatalf("invalid UpdatePost() error = %v", err)
	}
	unchanged, err := service.GetPost(ctx, fixture.userID, created.ID)
	if err != nil || unchanged.Title != "第一篇" || len(unchanged.Securities) != 1 || unchanged.Securities[0].ID != fixture.securityA || unchanged.Version != 1 {
		t.Fatalf("unchanged post = %#v, %v", unchanged, err)
	}
}

func TestConcurrentIdempotencyCreatesOnePost(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newPostFixture(t, ctx, database)
	service := mustIntegrationPostService(t, store)
	input := usecase.CreatePostInput{UserID: fixture.userID, CircleID: fixture.circleID, IdempotencyKey: "concurrent-" + fixture.suffix, Title: "并发幂等", Body: "正文", SecurityIDs: []int64{fixture.securityA, fixture.securityB}}
	const workers = 8
	start := make(chan struct{})
	ids := make(chan int64, workers)
	errs := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			post, err := service.CreatePost(ctx, input)
			if err != nil {
				errs <- err
				return
			}
			ids <- post.ID
		}()
	}
	close(start)
	group.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatalf("CreatePost() error = %v", err)
	}
	var want int64
	for id := range ids {
		if want == 0 {
			want = id
		} else if id != want {
			t.Fatalf("replayed IDs = %d/%d", want, id)
		}
	}
	var count int
	_ = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE author_id=? AND idempotency_key=?", fixture.userID, input.IdempotencyKey).Scan(&count)
	if count != 1 {
		t.Fatalf("post rows = %d", count)
	}
	conflict := input
	conflict.Body = "不同正文"
	if _, err := service.CreatePost(ctx, conflict); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("conflicting replay error = %v", err)
	}
}

func TestConcurrentPostVersionUpdateHasOneWinner(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newPostFixture(t, ctx, database)
	service := mustIntegrationPostService(t, store)
	post := fixture.create(t, ctx, service, "version-race", "原标题", []int64{fixture.securityA})
	start := make(chan struct{})
	errs := make(chan error, 2)
	var group sync.WaitGroup
	for index := range 2 {
		title := fmt.Sprintf("竞争标题%d", index)
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := service.UpdatePost(ctx, usecase.UpdatePostInput{UserID: fixture.userID, PostID: post.ID, Version: 1, Title: &title})
			errs <- err
		}()
	}
	close(start)
	group.Wait()
	close(errs)
	success, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, domain.ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatalf("UpdatePost() error = %v", err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success/conflicts = %d/%d", success, conflicts)
	}
}

func TestDeletePostClosesPendingReportsWithoutAdminAudit(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newPostFixture(t, ctx, database)
	service := mustIntegrationPostService(t, store)
	post := fixture.create(t, ctx, service, "delete-reports", "待删除帖子", []int64{fixture.securityA})
	reporterA := insertIntegrationUser(t, ctx, database, "report-a-"+fixture.suffix)
	reporterB := insertIntegrationUser(t, ctx, database, "report-b-"+fixture.suffix)
	for _, reporterID := range []int64{reporterA, reporterB} {
		if _, err := database.ExecContext(ctx, `
INSERT INTO reports (reporter_id, post_id, reason_code, details)
VALUES (?, ?, 'spam', '虚构举报')`, reporterID, post.ID); err != nil {
			t.Fatalf("insert pending report: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM reports WHERE post_id=?", post.ID)
		deleteIntegrationIDs(database, "users", []int64{reporterA, reporterB})
	})

	if err := service.DeletePost(ctx, fixture.userID, post.ID); err != nil {
		t.Fatalf("DeletePost() error = %v", err)
	}
	var deletedAt sql.NullTime
	var visibility string
	if err := database.QueryRowContext(ctx,
		"SELECT deleted_at, visibility FROM posts WHERE id=?", post.ID,
	).Scan(&deletedAt, &visibility); err != nil {
		t.Fatalf("read deleted post: %v", err)
	}
	if !deletedAt.Valid || visibility != "visible" {
		t.Fatalf("deleted_at/visibility = %v/%q, want set/visible", deletedAt, visibility)
	}
	var resolved int
	if err := database.QueryRowContext(ctx, `
SELECT COUNT(*) FROM reports
WHERE post_id=? AND status='resolved' AND resolution_action='author_deleted'
  AND handled_by IS NULL AND handled_at IS NOT NULL`, post.ID).Scan(&resolved); err != nil {
		t.Fatalf("count author_deleted reports: %v", err)
	}
	if resolved != 2 {
		t.Fatalf("author_deleted reports = %d, want 2", resolved)
	}
	var audits int
	if err := database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM admin_audit_logs WHERE post_id=?", post.ID,
	).Scan(&audits); err != nil || audits != 0 {
		t.Fatalf("admin audit rows = %d, error = %v, want 0", audits, err)
	}
}

type postFixture struct {
	suffix                                                   string
	userID, circleID, securityA, securityB, inactiveSecurity int64
	database                                                 *sql.DB
	postIDs                                                  []int64
}

func newPostFixture(t *testing.T, ctx context.Context, database *sql.DB) *postFixture {
	t.Helper()
	f := &postFixture{suffix: integrationSuffix(), database: database}
	f.userID = insertIntegrationUser(t, ctx, database, "post-"+f.suffix)
	result, err := database.ExecContext(ctx, "INSERT INTO circles (slug,name,description,status) VALUES (?,?, '', 'active')", "post-"+f.suffix, "帖子圈子"+f.suffix)
	if err != nil {
		t.Fatal(err)
	}
	f.circleID, _ = result.LastInsertId()
	_, err = database.ExecContext(ctx, "INSERT INTO circle_memberships (circle_id,user_id) VALUES (?,?)", f.circleID, f.userID)
	if err != nil {
		t.Fatal(err)
	}
	insertSecurity := func(code, status string) int64 {
		result, err := database.ExecContext(ctx, "INSERT INTO securities (market,code,name,status) VALUES (?,?,?,?)", "X"+f.suffix, code+f.suffix, "虚构证券"+code, status)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	f.securityA = insertSecurity("A", "active")
	f.securityB = insertSecurity("B", "active")
	f.inactiveSecurity = insertSecurity("I", "inactive")
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM post_securities WHERE post_id IN (SELECT id FROM posts WHERE circle_id=?)", f.circleID)
		_, _ = database.Exec("DELETE FROM posts WHERE circle_id=?", f.circleID)
		_, _ = database.Exec("DELETE FROM circle_memberships WHERE circle_id=?", f.circleID)
		deleteIntegrationIDs(database, "securities", []int64{f.securityA, f.securityB, f.inactiveSecurity})
		deleteIntegrationIDs(database, "circles", []int64{f.circleID})
		deleteIntegrationIDs(database, "users", []int64{f.userID})
	})
	return f
}
func (f *postFixture) create(t *testing.T, ctx context.Context, service *usecase.PostService, key, title string, tags []int64) domain.Post {
	t.Helper()
	post, err := service.CreatePost(ctx, usecase.CreatePostInput{UserID: f.userID, CircleID: f.circleID, IdempotencyKey: key + "-" + f.suffix, Title: title, Body: "虚构正文", SecurityIDs: tags})
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	return post
}
func mustIntegrationPostService(t *testing.T, store *Store) *usecase.PostService {
	t.Helper()
	service, err := usecase.NewPostService(store)
	if err != nil {
		t.Fatal(err)
	}
	return service
}
