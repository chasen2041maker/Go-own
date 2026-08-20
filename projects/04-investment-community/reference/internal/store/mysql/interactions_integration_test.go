//go:build integration

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestCommentReplyNotificationsIdempotencyOwnershipAndDeleteReportClosure(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture := newPostFixture(t, ctx, database)
	postService := mustIntegrationPostService(t, store)
	post := fixture.create(t, ctx, postService, "interaction-post", "互动帖子", []int64{fixture.securityA})
	commenterID := insertIntegrationUser(t, ctx, database, "commenter-"+fixture.suffix)
	if _, err := database.ExecContext(ctx,
		"INSERT INTO circle_memberships (circle_id,user_id) VALUES (?,?)", fixture.circleID, commenterID); err != nil {
		t.Fatalf("insert commenter membership: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM notifications WHERE post_id=?", post.ID)
		_, _ = database.Exec("DELETE FROM reports WHERE comment_id IN (SELECT id FROM comments WHERE post_id=?)", post.ID)
		_, _ = database.Exec("DELETE FROM comments WHERE post_id=?", post.ID)
		_, _ = database.Exec("DELETE FROM circle_memberships WHERE user_id=?", commenterID)
		deleteIntegrationIDs(database, "users", []int64{commenterID})
	})
	service, err := usecase.NewInteractionService(store)
	if err != nil {
		t.Fatal(err)
	}

	topInput := usecase.CreateCommentInput{UserID: commenterID, PostID: post.ID, Body: "顶级评论", IdempotencyKey: "top-" + fixture.suffix}
	top, err := service.CreateComment(ctx, topInput)
	if err != nil {
		t.Fatalf("CreateComment(top) error = %v", err)
	}
	replayed, err := service.CreateComment(ctx, topInput)
	if err != nil || replayed.ID != top.ID {
		t.Fatalf("replayed comment = %#v, error = %v", replayed, err)
	}
	conflict := topInput
	conflict.Body = "不同正文"
	if _, err := service.CreateComment(ctx, conflict); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("conflicting comment error = %v", err)
	}
	assertNotificationCount(t, ctx, database, fixture.userID, "comment", 1)

	reply, err := service.CreateComment(ctx, usecase.CreateCommentInput{UserID: fixture.userID, PostID: post.ID,
		ParentCommentID: &top.ID, Body: "一级回复", IdempotencyKey: "reply-" + fixture.suffix})
	if err != nil || reply.ParentCommentID == nil || *reply.ParentCommentID != top.ID {
		t.Fatalf("reply = %#v, error = %v", reply, err)
	}
	assertNotificationCount(t, ctx, database, commenterID, "reply", 1)
	if _, err := service.CreateComment(ctx, usecase.CreateCommentInput{UserID: commenterID, PostID: post.ID,
		ParentCommentID: &top.ID, Body: "回复自己", IdempotencyKey: "self-" + fixture.suffix}); err != nil {
		t.Fatalf("CreateComment(self reply) error = %v", err)
	}
	assertNotificationCount(t, ctx, database, commenterID, "reply", 1)

	page, err := service.ListNotifications(ctx, usecase.NotificationListInput{UserID: commenterID, UnreadOnly: true, Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].Actor == nil || page.Items[0].Actor.ID != fixture.userID {
		t.Fatalf("notification page = %#v, error = %v", page, err)
	}
	read, err := service.MarkAllNotificationsRead(ctx, commenterID)
	if err != nil || read.ReadCount != 1 {
		t.Fatalf("MarkAllNotificationsRead() = %#v, %v", read, err)
	}
	otherRead, err := service.MarkAllNotificationsRead(ctx, fixture.userID)
	if err != nil || otherRead.ReadCount != 1 {
		t.Fatalf("author MarkAllNotificationsRead() = %#v, %v", otherRead, err)
	}

	if _, err := database.ExecContext(ctx, `
INSERT INTO reports (reporter_id,comment_id,reason_code,details) VALUES (?,?,'spam','虚构举报')`,
		fixture.userID, top.ID); err != nil {
		t.Fatalf("insert comment report: %v", err)
	}
	if err := service.DeleteComment(ctx, commenterID, top.ID); err != nil {
		t.Fatalf("DeleteComment() error = %v", err)
	}
	var closed int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM reports
WHERE comment_id=? AND status='resolved' AND resolution_action='author_deleted'
AND handled_by IS NULL AND handled_at IS NOT NULL`, top.ID).Scan(&closed); err != nil || closed != 1 {
		t.Fatalf("closed comment reports = %d, error = %v", closed, err)
	}
	comments, err := service.ListComments(ctx, usecase.CommentListInput{UserID: fixture.userID, PostID: post.ID, Limit: 20})
	if err != nil {
		t.Fatalf("ListComments() error = %v", err)
	}
	for _, comment := range comments.Items {
		if comment.ID == top.ID || (comment.ParentCommentID != nil && *comment.ParentCommentID == top.ID) {
			t.Fatalf("deleted parent or its reply remained visible: %#v", comment)
		}
	}
}

func assertNotificationCount(t *testing.T, ctx context.Context, database *sql.DB, userID int64, kind string, want int) {
	t.Helper()
	var count int
	if err := database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM notifications WHERE user_id=? AND type=?", userID, kind,
	).Scan(&count); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != want {
		t.Fatalf("notification count (%d,%s) = %d, want %d", userID, kind, count, want)
	}
}
