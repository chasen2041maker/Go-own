package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestCreateCommentHTTPUsesAuthenticatedUserAndDoesNotExposeEmail(t *testing.T) {
	auth := &fakeAuthApplication{authenticate: func(context.Context, string) (domain.User, error) {
		return domain.User{ID: 42, DisplayName: "当前用户", Role: domain.RoleUser, Status: domain.UserStatusActive}, nil
	}}
	application := &fakeInteractionsApplication{create: func(_ context.Context, input usecase.CreateCommentInput) (domain.Comment, error) {
		if input.UserID != 42 || input.PostID != 7 || input.IdempotencyKey != "comment-key" || input.Body != "评论正文" {
			t.Fatalf("CreateComment input = %#v", input)
		}
		return domain.Comment{ID: 9, PostID: 7, Author: domain.PublicUser{ID: 42, DisplayName: "当前用户"},
			Body: input.Body, ModerationVersion: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}}
	mux := http.NewServeMux()
	registerInteractionRoutes(mux, auth, application, mustCursorCodec(t, time.Now()))
	handler := WithRequestID(mux)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/7/comments", strings.NewReader(`{"body":"评论正文"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Idempotency-Key", "comment-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	author := body["author"].(map[string]any)
	if _, exists := author["email"]; exists {
		t.Fatal("comment author leaked email")
	}
}

type fakeInteractionsApplication struct {
	create func(context.Context, usecase.CreateCommentInput) (domain.Comment, error)
}

func (application *fakeInteractionsApplication) CreateComment(ctx context.Context, input usecase.CreateCommentInput) (domain.Comment, error) {
	return application.create(ctx, input)
}
func (*fakeInteractionsApplication) ListComments(context.Context, usecase.CommentListInput) (usecase.CommentPage, error) {
	return usecase.CommentPage{}, nil
}
func (*fakeInteractionsApplication) DeleteComment(context.Context, int64, int64) error { return nil }
func (*fakeInteractionsApplication) ListNotifications(context.Context, usecase.NotificationListInput) (usecase.NotificationPage, error) {
	return usecase.NotificationPage{}, nil
}
func (*fakeInteractionsApplication) MarkAllNotificationsRead(context.Context, int64) (domain.NotificationReadResult, error) {
	return domain.NotificationReadResult{}, nil
}
