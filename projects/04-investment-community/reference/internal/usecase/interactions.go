package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

type InteractionsRepository interface {
	CreateComment(context.Context, domain.CreateCommentParams) (domain.Comment, error)
	ListComments(context.Context, domain.CommentListQuery) ([]domain.Comment, error)
	DeleteComment(context.Context, int64, int64) error
	ListNotifications(context.Context, domain.NotificationListQuery) ([]domain.Notification, error)
	MarkAllNotificationsRead(context.Context, int64) (domain.NotificationReadResult, error)
}

type CreateCommentInput struct {
	UserID          int64
	PostID          int64
	ParentCommentID *int64
	Body            string
	IdempotencyKey  string
}

type CommentListInput struct {
	UserID int64
	PostID int64
	After  *domain.CommentCursor
	Limit  int
}

type CommentPage struct {
	Items []domain.Comment
	Next  *domain.CommentCursor
}

type NotificationListInput struct {
	UserID     int64
	UnreadOnly bool
	After      *domain.NotificationCursor
	Limit      int
}

type NotificationPage struct {
	Items []domain.Notification
	Next  *domain.NotificationCursor
}

type InteractionService struct{ repository InteractionsRepository }

func NewInteractionService(repository InteractionsRepository) (*InteractionService, error) {
	if repository == nil {
		return nil, errors.New("interaction service repository is required")
	}
	return &InteractionService{repository: repository}, nil
}

func (service *InteractionService) CreateComment(ctx context.Context, input CreateCommentInput) (domain.Comment, error) {
	if err := validatePositiveID("user_id", input.UserID); err != nil {
		return domain.Comment{}, err
	}
	if err := validatePositiveID("post_id", input.PostID); err != nil {
		return domain.Comment{}, err
	}
	if input.ParentCommentID != nil {
		if err := validatePositiveID("parent_comment_id", *input.ParentCommentID); err != nil {
			return domain.Comment{}, err
		}
	}
	if err := validateIdempotencyKey(input.IdempotencyKey); err != nil {
		return domain.Comment{}, err
	}
	body, err := normalizePostText("body", input.Body, 2000)
	if err != nil {
		return domain.Comment{}, err
	}
	hash, err := commentRequestHash(input.UserID, input.PostID, body, input.ParentCommentID)
	if err != nil {
		return domain.Comment{}, fmt.Errorf("create comment: fingerprint request: %w", err)
	}
	comment, err := service.repository.CreateComment(ctx, domain.CreateCommentParams{
		AuthorID: input.UserID, PostID: input.PostID, ParentCommentID: input.ParentCommentID,
		Body: body, IdempotencyKey: input.IdempotencyKey, RequestHash: hash,
	})
	if err != nil {
		return domain.Comment{}, fmt.Errorf("create comment: %w", err)
	}
	return comment, nil
}

func (service *InteractionService) ListComments(ctx context.Context, input CommentListInput) (CommentPage, error) {
	if err := validatePositiveID("user_id", input.UserID); err != nil {
		return CommentPage{}, err
	}
	if err := validatePositiveID("post_id", input.PostID); err != nil {
		return CommentPage{}, err
	}
	limit, err := validatePageLimit(input.Limit)
	if err != nil {
		return CommentPage{}, err
	}
	if input.After != nil && (input.After.ID <= 0 || input.After.CreatedAt.IsZero()) {
		return CommentPage{}, &domain.ValidationError{Field: "cursor", Reason: "分页位置无效"}
	}
	items, err := service.repository.ListComments(ctx, domain.CommentListQuery{PostID: input.PostID, After: input.After, Limit: limit + 1})
	if err != nil {
		return CommentPage{}, fmt.Errorf("list comments: %w", err)
	}
	page := CommentPage{Items: items}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.Next = &domain.CommentCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return page, nil
}

func (service *InteractionService) DeleteComment(ctx context.Context, userID, commentID int64) error {
	if err := validatePositiveID("user_id", userID); err != nil {
		return err
	}
	if err := validatePositiveID("comment_id", commentID); err != nil {
		return err
	}
	if err := service.repository.DeleteComment(ctx, userID, commentID); err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}

func (service *InteractionService) ListNotifications(ctx context.Context, input NotificationListInput) (NotificationPage, error) {
	if err := validatePositiveID("user_id", input.UserID); err != nil {
		return NotificationPage{}, err
	}
	limit, err := validatePageLimit(input.Limit)
	if err != nil {
		return NotificationPage{}, err
	}
	if input.After != nil && (input.After.ID <= 0 || input.After.CreatedAt.IsZero()) {
		return NotificationPage{}, &domain.ValidationError{Field: "cursor", Reason: "分页位置无效"}
	}
	items, err := service.repository.ListNotifications(ctx, domain.NotificationListQuery{
		UserID: input.UserID, UnreadOnly: input.UnreadOnly, After: input.After, Limit: limit + 1,
	})
	if err != nil {
		return NotificationPage{}, fmt.Errorf("list notifications: %w", err)
	}
	page := NotificationPage{Items: items}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.Next = &domain.NotificationCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return page, nil
}

func (service *InteractionService) MarkAllNotificationsRead(ctx context.Context, userID int64) (domain.NotificationReadResult, error) {
	if err := validatePositiveID("user_id", userID); err != nil {
		return domain.NotificationReadResult{}, err
	}
	result, err := service.repository.MarkAllNotificationsRead(ctx, userID)
	if err != nil {
		return domain.NotificationReadResult{}, fmt.Errorf("mark all notifications read: %w", err)
	}
	return result, nil
}

func commentRequestHash(userID, postID int64, body string, parentID *int64) (string, error) {
	// 指纹显式绑定 operation、数据库用户和实际路径；相同 key 因此不能跨用户或跨帖子误重放。
	canonicalBody, err := json.Marshal(struct {
		Body            string `json:"body"`
		ParentCommentID *int64 `json:"parent_comment_id"`
	}{Body: body, ParentCommentID: parentID})
	if err != nil {
		return "", err
	}
	contents := strings.Join([]string{
		"createComment",
		strconv.FormatInt(userID, 10),
		"/api/v1/posts/" + strconv.FormatInt(postID, 10) + "/comments",
		string(canonicalBody),
	}, "\n")
	sum := sha256.Sum256([]byte(contents))
	return hex.EncodeToString(sum[:]), nil
}
