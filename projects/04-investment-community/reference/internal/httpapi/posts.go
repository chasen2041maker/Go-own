package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

type PostsApplication interface {
	CreatePost(context.Context, usecase.CreatePostInput) (domain.Post, error)
	ListPosts(context.Context, usecase.PostListInput) (usecase.PostPage, error)
	GetPost(context.Context, int64, int64) (domain.Post, error)
	UpdatePost(context.Context, usecase.UpdatePostInput) (domain.Post, error)
	DeletePost(context.Context, int64, int64) error
}

type createPostRequest struct {
	CircleID    int64   `json:"circle_id"`
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	SecurityIDs []int64 `json:"security_ids"`
}

type optionalString struct {
	Set   bool
	Value string
}

func (value *optionalString) UnmarshalJSON(data []byte) error {
	value.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("value cannot be null")
	}
	return json.Unmarshal(data, &value.Value)
}

type optionalInt64Slice struct {
	Set   bool
	Value []int64
}

func (value *optionalInt64Slice) UnmarshalJSON(data []byte) error {
	value.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("value cannot be null")
	}
	return json.Unmarshal(data, &value.Value)
}

type updatePostRequest struct {
	Version     *int64             `json:"version"`
	Title       optionalString     `json:"title"`
	Body        optionalString     `json:"body"`
	SecurityIDs optionalInt64Slice `json:"security_ids"`
}

type publicUserResponse struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
}
type postCircleResponse struct {
	ID   int64  `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}
type postResponse struct {
	ID                int64              `json:"id"`
	Circle            postCircleResponse `json:"circle"`
	Author            publicUserResponse `json:"author"`
	Title             string             `json:"title"`
	Body              string             `json:"body"`
	Securities        []securityResponse `json:"securities"`
	CommentCount      int64              `json:"comment_count"`
	Visibility        domain.Visibility  `json:"visibility"`
	Version           int64              `json:"version"`
	ModerationVersion int64              `json:"moderation_version"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}
type postListResponse struct {
	Items []postResponse     `json:"items"`
	Page  cursorPageResponse `json:"page"`
}

type postsHandler struct {
	application PostsApplication
	cursors     *CursorCodec
}

func registerPostRoutes(mux *http.ServeMux, auth AuthApplication, application PostsApplication, cursors *CursorCodec) {
	handler := postsHandler{application: application, cursors: cursors}
	mux.Handle("/api/v1/posts", authenticate(auth, http.HandlerFunc(handler.collection)))
	mux.Handle("/api/v1/posts/{postId}", authenticate(auth, http.HandlerFunc(handler.item)))
}

func (handler postsHandler) collection(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		handler.list(writer, request)
	case http.MethodPost:
		handler.create(writer, request)
	default:
		writer.Header().Set("Allow", "GET, POST")
		WriteError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持", nil)
	}
}

func (handler postsHandler) item(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		handler.get(writer, request)
	case http.MethodPatch:
		handler.update(writer, request)
	case http.MethodDelete:
		handler.remove(writer, request)
	default:
		writer.Header().Set("Allow", "GET, PATCH, DELETE")
		WriteError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持", nil)
	}
}

func (handler postsHandler) create(writer http.ResponseWriter, request *http.Request) {
	key, failure := parseIdempotencyKey(request.Header.Values("Idempotency-Key"))
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	var body createPostRequest
	if failure := decodeJSON(writer, request, &body); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	user, ok := CurrentUserFromContext(request.Context())
	if !ok {
		writeUnauthenticated(writer, request)
		return
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	post, err := handler.application.CreatePost(request.Context(), usecase.CreatePostInput{UserID: user.ID, IdempotencyKey: key,
		CircleID: body.CircleID, Title: body.Title, Body: body.Body, SecurityIDs: body.SecurityIDs})
	if err != nil {
		writePostApplicationError(writer, request, err)
		return
	}
	WriteJSON(writer, http.StatusCreated, publicPost(post))
}

func (handler postsHandler) list(writer http.ResponseWriter, request *http.Request) {
	query, failure := parsePostListQuery(request)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	user, ok := CurrentUserFromContext(request.Context())
	if !ok {
		writeUnauthenticated(writer, request)
		return
	}
	if handler.application == nil || handler.cursors == nil {
		writeInternalError(writer, request)
		return
	}
	input := usecase.PostListInput{UserID: user.ID, CircleID: query.circleID, SecurityID: query.securityID, Limit: query.limit}
	binding := postCursorBinding{CircleID: query.circleID, SecurityID: query.securityID, Limit: query.limit}
	if query.cursor != "" {
		position, err := handler.cursors.DecodePost(query.cursor, binding)
		if err != nil {
			WriteError(writer, request, http.StatusBadRequest, "invalid_cursor", "分页游标无效", nil)
			return
		}
		input.After = &position
	}
	page, err := handler.application.ListPosts(request.Context(), input)
	if err != nil {
		writePostApplicationError(writer, request, err)
		return
	}
	response := postListResponse{Items: make([]postResponse, 0, len(page.Items)), Page: cursorPageResponse{HasMore: page.Next != nil}}
	for _, post := range page.Items {
		response.Items = append(response.Items, publicPost(post))
	}
	if page.Next != nil {
		token, err := handler.cursors.EncodePost(binding, *page.Next)
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		response.Page.NextCursor = &token
	}
	WriteJSON(writer, http.StatusOK, response)
}

func (handler postsHandler) get(writer http.ResponseWriter, request *http.Request) {
	postID, failure := parsePostID(request)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	user, ok := CurrentUserFromContext(request.Context())
	if !ok {
		writeUnauthenticated(writer, request)
		return
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	post, err := handler.application.GetPost(request.Context(), user.ID, postID)
	if err != nil {
		writePostApplicationError(writer, request, err)
		return
	}
	WriteJSON(writer, http.StatusOK, publicPost(post))
}

func (handler postsHandler) update(writer http.ResponseWriter, request *http.Request) {
	postID, failure := parsePostID(request)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	var body updatePostRequest
	if failure := decodeJSON(writer, request, &body); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	if body.Version == nil || (!body.Title.Set && !body.Body.Set && !body.SecurityIDs.Set) {
		WriteError(writer, request, http.StatusBadRequest, "invalid_request", "version 和至少一个修改字段为必填", nil)
		return
	}
	user, ok := CurrentUserFromContext(request.Context())
	if !ok {
		writeUnauthenticated(writer, request)
		return
	}
	input := usecase.UpdatePostInput{UserID: user.ID, PostID: postID, Version: *body.Version}
	if body.Title.Set {
		input.Title = &body.Title.Value
	}
	if body.Body.Set {
		input.Body = &body.Body.Value
	}
	if body.SecurityIDs.Set {
		input.SecurityIDs = &body.SecurityIDs.Value
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	post, err := handler.application.UpdatePost(request.Context(), input)
	if err != nil {
		writePostApplicationError(writer, request, err)
		return
	}
	WriteJSON(writer, http.StatusOK, publicPost(post))
}

func (handler postsHandler) remove(writer http.ResponseWriter, request *http.Request) {
	postID, failure := parsePostID(request)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	user, ok := CurrentUserFromContext(request.Context())
	if !ok {
		writeUnauthenticated(writer, request)
		return
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	if err := handler.application.DeletePost(request.Context(), user.ID, postID); err != nil {
		writePostApplicationError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

type parsedPostListQuery struct {
	circleID   int64
	securityID int64
	limit      int
	cursor     string
}

func parsePostListQuery(request *http.Request) (parsedPostListQuery, *decodeFailure) {
	values, failure := parseKnownQuery(request, map[string]struct{}{"circle_id": {}, "security_id": {}, "limit": {}, "cursor": {}})
	if failure != nil {
		return parsedPostListQuery{}, failure
	}
	circleID, failure := optionalQueryID(values, "circle_id")
	if failure != nil {
		return parsedPostListQuery{}, failure
	}
	securityID, failure := optionalQueryID(values, "security_id")
	if failure != nil {
		return parsedPostListQuery{}, failure
	}
	limit, failure := parsePageLimit(values)
	if failure != nil {
		return parsedPostListQuery{}, failure
	}
	cursor, failure := singleQueryValue(values, "cursor")
	if failure != nil {
		return parsedPostListQuery{}, failure
	}
	if _, present := values["cursor"]; present && cursor == "" {
		return parsedPostListQuery{}, &decodeFailure{status: 400, code: "invalid_cursor", message: "分页游标无效"}
	}
	return parsedPostListQuery{circleID: circleID, securityID: securityID, limit: limit, cursor: cursor}, nil
}

func optionalQueryID(values url.Values, name string) (int64, *decodeFailure) {
	raw, failure := singleQueryValue(values, name)
	if failure != nil {
		return 0, failure
	}
	if _, present := values[name]; !present {
		return 0, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, invalidRequestFailure(name + " 必须是正 int64")
	}
	return id, nil
}

func parsePostID(request *http.Request) (int64, *decodeFailure) {
	id, err := strconv.ParseInt(request.PathValue("postId"), 10, 64)
	if err != nil || id <= 0 {
		return 0, invalidRequestFailure("postId 必须是正 int64")
	}
	return id, nil
}

func parseIdempotencyKey(values []string) (string, *decodeFailure) {
	if len(values) != 1 || len(values[0]) < 1 || len(values[0]) > 128 {
		return "", invalidRequestFailure("Idempotency-Key 必须是 1 到 128 个可见 ASCII 字符")
	}
	for index := range len(values[0]) {
		if values[0][index] < '!' || values[0][index] > '~' {
			return "", invalidRequestFailure("Idempotency-Key 只能包含可见 ASCII 字符")
		}
	}
	return values[0], nil
}

func publicPost(post domain.Post) postResponse {
	securities := make([]securityResponse, 0, len(post.Securities))
	for _, security := range post.Securities {
		securities = append(securities, securityResponse{ID: security.ID, Code: security.Code, Name: security.Name, Exchange: security.Exchange})
	}
	return postResponse{ID: post.ID, Circle: postCircleResponse{ID: post.Circle.ID, Slug: post.Circle.Slug, Name: post.Circle.Name},
		Author: publicUserResponse{ID: post.Author.ID, DisplayName: post.Author.DisplayName}, Title: post.Title, Body: post.Body,
		Securities: securities, CommentCount: post.CommentCount, Visibility: post.Visibility, Version: post.Version,
		ModerationVersion: post.ModerationVersion, CreatedAt: post.CreatedAt.UTC(), UpdatedAt: post.UpdatedAt.UTC()}
}

func writePostApplicationError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "请求字段未通过校验", []FieldViolation{{Field: validation.Field, Reason: validation.Reason}})
	case errors.Is(err, domain.ErrMembershipRequired):
		WriteError(writer, request, http.StatusForbidden, "membership_required", "需要先加入圈子", nil)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(writer, request, http.StatusForbidden, "forbidden", "没有执行此操作的权限", nil)
	case errors.Is(err, domain.ErrPostNotFound):
		WriteError(writer, request, http.StatusNotFound, "not_found", "帖子不存在", nil)
	case errors.Is(err, domain.ErrIdempotencyConflict):
		WriteError(writer, request, http.StatusConflict, "idempotency_conflict", "幂等键已用于不同请求", nil)
	case errors.Is(err, domain.ErrVersionConflict):
		WriteError(writer, request, http.StatusConflict, "version_conflict", "帖子版本已变化", nil)
	case errors.Is(err, domain.ErrContentNotEditable):
		WriteError(writer, request, http.StatusConflict, "content_not_editable", "帖子当前不可编辑", nil)
	default:
		writeInternalError(writer, request)
	}
}
