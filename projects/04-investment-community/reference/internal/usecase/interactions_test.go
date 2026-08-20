package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func TestCreateCommentNormalizesBodyAndBindsFingerprintToPost(t *testing.T) {
	var calls []domain.CreateCommentParams
	repository := &fakeInteractionsRepository{create: func(_ context.Context, input domain.CreateCommentParams) (domain.Comment, error) {
		calls = append(calls, input)
		return domain.Comment{ID: int64(len(calls))}, nil
	}}
	service := mustInteractionService(t, repository)
	parentID := int64(8)

	for _, postID := range []int64{7, 9} {
		created, err := service.CreateComment(context.Background(), CreateCommentInput{
			UserID: 42, PostID: postID, ParentCommentID: &parentID,
			Body: "  一层回复  ", IdempotencyKey: "comment-key",
		})
		if err != nil || created.ID == 0 {
			t.Fatalf("CreateComment(post=%d) = %#v, %v", postID, created, err)
		}
	}
	if calls[0].Body != "一层回复" || calls[0].ParentCommentID == nil || *calls[0].ParentCommentID != parentID {
		t.Fatalf("repository input = %#v", calls[0])
	}
	if len(calls[0].RequestHash) != 64 || calls[0].RequestHash == calls[1].RequestHash {
		t.Fatalf("post-bound hashes = %q/%q", calls[0].RequestHash, calls[1].RequestHash)
	}
}

func TestCreateCommentRejectsInvalidInputBeforeRepository(t *testing.T) {
	calls := 0
	repository := &fakeInteractionsRepository{create: func(context.Context, domain.CreateCommentParams) (domain.Comment, error) {
		calls++
		return domain.Comment{}, nil
	}}
	service := mustInteractionService(t, repository)
	zero := int64(0)
	tests := []CreateCommentInput{
		{UserID: 0, PostID: 1, Body: "正文", IdempotencyKey: "key"},
		{UserID: 1, PostID: 0, Body: "正文", IdempotencyKey: "key"},
		{UserID: 1, PostID: 1, ParentCommentID: &zero, Body: "正文", IdempotencyKey: "key"},
		{UserID: 1, PostID: 1, Body: " ", IdempotencyKey: "key"},
		{UserID: 1, PostID: 1, Body: "正文", IdempotencyKey: "bad key"},
	}
	for _, input := range tests {
		if _, err := service.CreateComment(context.Background(), input); err == nil {
			t.Fatalf("CreateComment(%#v) error = nil", input)
		}
	}
	if calls != 0 {
		t.Fatalf("repository calls = %d, want 0", calls)
	}
}

func TestInteractionsKeepStablePagesOwnershipAndRepositoryErrors(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 123000000, time.UTC)
	repository := &fakeInteractionsRepository{
		listComments: func(_ context.Context, query domain.CommentListQuery) ([]domain.Comment, error) {
			if query.PostID != 7 || query.Limit != 3 || query.After == nil || query.After.ID != 10 {
				t.Fatalf("ListComments query = %#v", query)
			}
			return []domain.Comment{{ID: 11, CreatedAt: now}, {ID: 12, CreatedAt: now}, {ID: 13, CreatedAt: now}}, nil
		},
		deleteComment: func(_ context.Context, actorID, commentID int64) error {
			if actorID != 42 || commentID != 11 {
				t.Fatalf("DeleteComment ids = %d/%d", actorID, commentID)
			}
			return domain.ErrForbidden
		},
		listNotifications: func(_ context.Context, query domain.NotificationListQuery) ([]domain.Notification, error) {
			if query.UserID != 42 || !query.UnreadOnly || query.Limit != 2 || query.After == nil || query.After.ID != 99 {
				t.Fatalf("ListNotifications query = %#v", query)
			}
			return []domain.Notification{{ID: 98, CreatedAt: now}, {ID: 97, CreatedAt: now.Add(-time.Second)}}, nil
		},
		markRead: func(_ context.Context, userID int64) (domain.NotificationReadResult, error) {
			if userID != 42 {
				t.Fatalf("MarkAllNotificationsRead user = %d", userID)
			}
			return domain.NotificationReadResult{ReadCount: 2, ReadAt: now}, nil
		},
	}
	service := mustInteractionService(t, repository)

	commentPage, err := service.ListComments(context.Background(), CommentListInput{UserID: 42, PostID: 7, Limit: 2, After: &domain.CommentCursor{CreatedAt: now.Add(-time.Second), ID: 10}})
	if err != nil || len(commentPage.Items) != 2 || commentPage.Next == nil || commentPage.Next.ID != 12 {
		t.Fatalf("ListComments() = %#v, %v", commentPage, err)
	}
	if err := service.DeleteComment(context.Background(), 42, 11); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("DeleteComment() error = %v", err)
	}
	notificationPage, err := service.ListNotifications(context.Background(), NotificationListInput{UserID: 42, UnreadOnly: true, Limit: 1, After: &domain.NotificationCursor{CreatedAt: now.Add(time.Second), ID: 99}})
	if err != nil || len(notificationPage.Items) != 1 || notificationPage.Next == nil || notificationPage.Next.ID != 98 {
		t.Fatalf("ListNotifications() = %#v, %v", notificationPage, err)
	}
	read, err := service.MarkAllNotificationsRead(context.Background(), 42)
	if err != nil || read.ReadCount != 2 || !read.ReadAt.Equal(now) {
		t.Fatalf("MarkAllNotificationsRead() = %#v, %v", read, err)
	}
}

type fakeInteractionsRepository struct {
	create            func(context.Context, domain.CreateCommentParams) (domain.Comment, error)
	listComments      func(context.Context, domain.CommentListQuery) ([]domain.Comment, error)
	deleteComment     func(context.Context, int64, int64) error
	listNotifications func(context.Context, domain.NotificationListQuery) ([]domain.Notification, error)
	markRead          func(context.Context, int64) (domain.NotificationReadResult, error)
}

func (repository *fakeInteractionsRepository) CreateComment(ctx context.Context, input domain.CreateCommentParams) (domain.Comment, error) {
	return repository.create(ctx, input)
}
func (repository *fakeInteractionsRepository) ListComments(ctx context.Context, query domain.CommentListQuery) ([]domain.Comment, error) {
	return repository.listComments(ctx, query)
}
func (repository *fakeInteractionsRepository) DeleteComment(ctx context.Context, actorID, commentID int64) error {
	return repository.deleteComment(ctx, actorID, commentID)
}
func (repository *fakeInteractionsRepository) ListNotifications(ctx context.Context, query domain.NotificationListQuery) ([]domain.Notification, error) {
	return repository.listNotifications(ctx, query)
}
func (repository *fakeInteractionsRepository) MarkAllNotificationsRead(ctx context.Context, userID int64) (domain.NotificationReadResult, error) {
	return repository.markRead(ctx, userID)
}

func mustInteractionService(t *testing.T, repository InteractionsRepository) *InteractionService {
	t.Helper()
	service, err := NewInteractionService(repository)
	if err != nil {
		t.Fatalf("NewInteractionService() error = %v", err)
	}
	return service
}
