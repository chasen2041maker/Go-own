package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

type InteractionsApplication interface {
	CreateComment(context.Context, usecase.CreateCommentInput) (domain.Comment, error)
	ListComments(context.Context, usecase.CommentListInput) (usecase.CommentPage, error)
	DeleteComment(context.Context, int64, int64) error
	ListNotifications(context.Context, usecase.NotificationListInput) (usecase.NotificationPage, error)
	MarkAllNotificationsRead(context.Context, int64) (domain.NotificationReadResult, error)
}

type createCommentRequest struct {
	Body            string `json:"body"`
	ParentCommentID *int64 `json:"parent_comment_id"`
}

type commentResponse struct {
	ID                int64              `json:"id"`
	PostID            int64              `json:"post_id"`
	ParentCommentID   *int64             `json:"parent_comment_id"`
	Author            publicUserResponse `json:"author"`
	Body              string             `json:"body"`
	ModerationVersion int64              `json:"moderation_version"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type commentListResponse struct {
	Items []commentResponse  `json:"items"`
	Page  cursorPageResponse `json:"page"`
}

type notificationResponse struct {
	ID        int64                   `json:"id"`
	Type      domain.NotificationType `json:"type"`
	Actor     *publicUserResponse     `json:"actor"`
	PostID    int64                   `json:"post_id"`
	CommentID *int64                  `json:"comment_id"`
	ReadAt    *time.Time              `json:"read_at"`
	CreatedAt time.Time               `json:"created_at"`
}

type notificationListResponse struct {
	Items []notificationResponse `json:"items"`
	Page  cursorPageResponse     `json:"page"`
}

type markNotificationsReadResponse struct {
	ReadCount int64     `json:"read_count"`
	ReadAt    time.Time `json:"read_at"`
}

type interactionsHandler struct {
	application InteractionsApplication
	cursors     *CursorCodec
}

func registerInteractionRoutes(mux *http.ServeMux, auth AuthApplication, application InteractionsApplication, cursors *CursorCodec) {
	handler := interactionsHandler{application: application, cursors: cursors}
	mux.Handle("/api/v1/posts/{postId}/comments", authenticate(auth, http.HandlerFunc(handler.comments)))
	mux.Handle("/api/v1/comments/{commentId}", authenticate(auth, http.HandlerFunc(handler.comment)))
	mux.Handle("/api/v1/notifications", authenticate(auth, http.HandlerFunc(handler.notifications)))
	mux.Handle("/api/v1/notifications/read", authenticate(auth, http.HandlerFunc(handler.readNotifications)))
}

func (handler interactionsHandler) comments(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		handler.listComments(writer, request)
	case http.MethodPost:
		handler.createComment(writer, request)
	default:
		writer.Header().Set("Allow", "GET, POST")
		WriteError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持", nil)
	}
}

func (handler interactionsHandler) comment(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodDelete {
		writer.Header().Set("Allow", http.MethodDelete)
		WriteError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持", nil)
		return
	}
	user, _ := CurrentUserFromContext(request.Context())
	commentID, failure := parsePathID(request.PathValue("commentId"), "commentId")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	if err := handler.application.DeleteComment(request.Context(), user.ID, commentID); err != nil {
		writeInteractionError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler interactionsHandler) createComment(writer http.ResponseWriter, request *http.Request) {
	user, _ := CurrentUserFromContext(request.Context())
	postID, failure := parsePostID(request)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	key, failure := parseIdempotencyKey(request.Header.Values("Idempotency-Key"))
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	var body createCommentRequest
	if failure := decodeJSON(writer, request, &body); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	comment, err := handler.application.CreateComment(request.Context(), usecase.CreateCommentInput{
		UserID: user.ID, PostID: postID, ParentCommentID: body.ParentCommentID, Body: body.Body, IdempotencyKey: key,
	})
	if err != nil {
		writeInteractionError(writer, request, err)
		return
	}
	WriteJSON(writer, http.StatusCreated, publicComment(comment))
}

func (handler interactionsHandler) listComments(writer http.ResponseWriter, request *http.Request) {
	user, _ := CurrentUserFromContext(request.Context())
	postID, failure := parsePostID(request)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	values, failure := parseKnownQuery(request, map[string]struct{}{"cursor": {}, "limit": {}})
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	limit, cursor, failure := parseSimplePage(values)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	binding := commentCursorBinding{PostID: postID, Limit: limit}
	var after *domain.CommentCursor
	if cursor != "" {
		decoded, err := handler.cursors.DecodeComment(cursor, binding)
		if err != nil {
			WriteError(writer, request, http.StatusBadRequest, "invalid_cursor", "分页游标无效", nil)
			return
		}
		after = &decoded
	}
	page, err := handler.application.ListComments(request.Context(), usecase.CommentListInput{UserID: user.ID, PostID: postID, Limit: limit, After: after})
	if err != nil {
		writeInteractionError(writer, request, err)
		return
	}
	items := make([]commentResponse, 0, len(page.Items))
	for _, comment := range page.Items {
		items = append(items, publicComment(comment))
	}
	next := ""
	if page.Next != nil {
		next, err = handler.cursors.EncodeComment(binding, *page.Next)
		if err != nil {
			writeInternalError(writer, request)
			return
		}
	}
	WriteJSON(writer, http.StatusOK, commentListResponse{Items: items, Page: cursorPage(next)})
}

func (handler interactionsHandler) notifications(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		WriteError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持", nil)
		return
	}
	user, _ := CurrentUserFromContext(request.Context())
	values, failure := parseKnownQuery(request, map[string]struct{}{"cursor": {}, "limit": {}, "unread_only": {}})
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	limit, cursor, failure := parseSimplePage(values)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	unreadOnly := false
	if raw, present := values["unread_only"]; present {
		if len(raw) != 1 {
			WriteError(writer, request, http.StatusBadRequest, "invalid_request", "unread_only 只能出现一次", nil)
			return
		}
		parsed, err := strconv.ParseBool(raw[0])
		if err != nil {
			WriteError(writer, request, http.StatusBadRequest, "invalid_request", "unread_only 必须是布尔值", nil)
			return
		}
		unreadOnly = parsed
	}
	binding := notificationCursorBinding{UserID: user.ID, UnreadOnly: unreadOnly, Limit: limit}
	var after *domain.NotificationCursor
	if cursor != "" {
		decoded, err := handler.cursors.DecodeNotification(cursor, binding)
		if err != nil {
			WriteError(writer, request, http.StatusBadRequest, "invalid_cursor", "分页游标无效", nil)
			return
		}
		after = &decoded
	}
	page, err := handler.application.ListNotifications(request.Context(), usecase.NotificationListInput{
		UserID: user.ID, UnreadOnly: unreadOnly, Limit: limit, After: after,
	})
	if err != nil {
		writeInteractionError(writer, request, err)
		return
	}
	items := make([]notificationResponse, 0, len(page.Items))
	for _, notification := range page.Items {
		items = append(items, publicNotification(notification))
	}
	next := ""
	if page.Next != nil {
		next, err = handler.cursors.EncodeNotification(binding, *page.Next)
		if err != nil {
			writeInternalError(writer, request)
			return
		}
	}
	WriteJSON(writer, http.StatusOK, notificationListResponse{Items: items, Page: cursorPage(next)})
}

func (handler interactionsHandler) readNotifications(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPut {
		writer.Header().Set("Allow", http.MethodPut)
		WriteError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持", nil)
		return
	}
	user, _ := CurrentUserFromContext(request.Context())
	result, err := handler.application.MarkAllNotificationsRead(request.Context(), user.ID)
	if err != nil {
		writeInteractionError(writer, request, err)
		return
	}
	WriteJSON(writer, http.StatusOK, markNotificationsReadResponse{ReadCount: result.ReadCount, ReadAt: result.ReadAt.UTC()})
}

func parseSimplePage(values url.Values) (int, string, *decodeFailure) {
	limit, failure := parsePageLimit(values)
	if failure != nil {
		return 0, "", failure
	}
	cursor, failure := singleQueryValue(values, "cursor")
	if failure != nil {
		return 0, "", failure
	}
	if _, present := values["cursor"]; present && cursor == "" {
		return 0, "", &decodeFailure{status: http.StatusBadRequest, code: "invalid_cursor", message: "分页游标无效"}
	}
	return limit, cursor, nil
}

func cursorPage(next string) cursorPageResponse {
	page := cursorPageResponse{HasMore: next != ""}
	if next != "" {
		page.NextCursor = &next
	}
	return page
}

func parsePathID(raw, name string) (int64, *decodeFailure) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != raw {
		return 0, invalidRequestFailure(name + " 必须是规范正 int64")
	}
	return id, nil
}

func publicComment(comment domain.Comment) commentResponse {
	return commentResponse{ID: comment.ID, PostID: comment.PostID, ParentCommentID: comment.ParentCommentID,
		Author: publicUserResponse{ID: comment.Author.ID, DisplayName: comment.Author.DisplayName}, Body: comment.Body,
		ModerationVersion: comment.ModerationVersion, CreatedAt: comment.CreatedAt.UTC(), UpdatedAt: comment.UpdatedAt.UTC()}
}

func publicNotification(notification domain.Notification) notificationResponse {
	var actor *publicUserResponse
	if notification.Actor != nil {
		actor = &publicUserResponse{ID: notification.Actor.ID, DisplayName: notification.Actor.DisplayName}
	}
	return notificationResponse{ID: notification.ID, Type: notification.Type, Actor: actor, PostID: notification.PostID,
		CommentID: notification.CommentID, ReadAt: notification.ReadAt, CreatedAt: notification.CreatedAt.UTC()}
}

func writeInteractionError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "请求字段未通过校验", []FieldViolation{{Field: validation.Field, Reason: validation.Reason}})
	case errors.Is(err, domain.ErrMembershipRequired):
		WriteError(writer, request, http.StatusForbidden, "membership_required", "需要先加入圈子", nil)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(writer, request, http.StatusForbidden, "forbidden", "没有执行此操作的权限", nil)
	case errors.Is(err, domain.ErrPostNotFound), errors.Is(err, domain.ErrCommentNotFound):
		WriteError(writer, request, http.StatusNotFound, "not_found", "内容不存在", nil)
	case errors.Is(err, domain.ErrIdempotencyConflict):
		WriteError(writer, request, http.StatusConflict, "idempotency_conflict", "幂等键已用于不同请求", nil)
	case errors.Is(err, domain.ErrContentNotEditable):
		WriteError(writer, request, http.StatusConflict, "content_not_editable", "内容当前不可编辑", nil)
	case errors.Is(err, domain.ErrParentCommentInvalid):
		WriteError(writer, request, http.StatusUnprocessableEntity, "parent_comment_invalid", "父评论不符合一级回复规则", nil)
	default:
		writeInternalError(writer, request)
	}
}
