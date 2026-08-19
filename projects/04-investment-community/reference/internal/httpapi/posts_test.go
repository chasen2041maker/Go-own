package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestPostsHTTPCreateListDetailUpdateDeleteAndPublicAuthorShape(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 123000000, time.UTC)
	post := domain.Post{ID: 10, Circle: domain.PostCircle{ID: 7, Slug: "risk-lab", Name: "风险实验室"}, Author: domain.PublicUser{ID: 42, DisplayName: "Alice"}, Title: "标题", Body: "正文", Securities: []domain.Security{{ID: 3, Code: "NOVA", Name: "新星", Exchange: "XSEA"}}, Visibility: domain.VisibilityVisible, Version: 1, ModerationVersion: 1, CreatedAt: now, UpdatedAt: now}
	calls := 0
	application := &fakePostsApplication{
		create: func(_ context.Context, input usecase.CreatePostInput) (domain.Post, error) {
			if input.UserID != 42 || input.IdempotencyKey != "post-key-1" || input.CircleID != 7 {
				t.Fatalf("CreatePost() input = %#v", input)
			}
			return post, nil
		},
		list: func(_ context.Context, input usecase.PostListInput) (usecase.PostPage, error) {
			calls++
			if input.UserID != 42 || input.CircleID != 7 || input.SecurityID != 3 || input.Limit != 1 {
				t.Fatalf("ListPosts() input = %#v", input)
			}
			if calls == 1 {
				return usecase.PostPage{Items: []domain.Post{post}, Next: &domain.PostCursor{CreatedAt: now, ID: 10}}, nil
			}
			if input.After == nil || input.After.ID != 10 {
				t.Fatalf("ListPosts() after = %#v", input.After)
			}
			return usecase.PostPage{}, nil
		},
		get: func(_ context.Context, userID, postID int64) (domain.Post, error) { return post, nil },
		update: func(_ context.Context, input usecase.UpdatePostInput) (domain.Post, error) {
			if input.UserID != 42 || input.PostID != 10 || input.Version != 1 || input.Title == nil {
				t.Fatalf("UpdatePost() input = %#v", input)
			}
			post.Version = 2
			return post, nil
		},
		remove: func(_ context.Context, userID, postID int64) error { return nil },
	}
	router := newPostsTestRouter(t, application, mustCursorCodec(t, now), 42)

	created := authenticatedJSONRequest(router, http.MethodPost, "/api/v1/posts", `{"circle_id":7,"title":"标题","body":"正文","security_ids":[3]}`, "post-key-1")
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	var public map[string]any
	if err := json.NewDecoder(created.Body).Decode(&public); err != nil {
		t.Fatal(err)
	}
	author := public["author"].(map[string]any)
	if _, ok := author["email"]; ok {
		t.Fatalf("public author leaked email: %#v", author)
	}
	if len(author) != 2 {
		t.Fatalf("public author = %#v", author)
	}

	first := performAuthenticatedRequest(router, http.MethodGet, "/api/v1/posts?circle_id=7&security_id=3&limit=1", "", "")
	if first.Code != http.StatusOK {
		t.Fatalf("list = %d %s", first.Code, first.Body.String())
	}
	var page postListResponse
	if err := json.NewDecoder(first.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page = %#v", page)
	}
	second := performAuthenticatedRequest(router, http.MethodGet, "/api/v1/posts?circle_id=7&security_id=3&limit=1&cursor="+url.QueryEscape(*page.Page.NextCursor), "", "")
	if second.Code != http.StatusOK {
		t.Fatalf("second = %d %s", second.Code, second.Body.String())
	}
	changed := performAuthenticatedRequest(router, http.MethodGet, "/api/v1/posts?circle_id=8&security_id=3&limit=1&cursor="+url.QueryEscape(*page.Page.NextCursor), "", "")
	if changed.Code != http.StatusBadRequest || decodeError(t, changed).Code != "invalid_cursor" {
		t.Fatalf("changed = %d %s", changed.Code, changed.Body.String())
	}

	if detail := performAuthenticatedRequest(router, http.MethodGet, "/api/v1/posts/10", "", ""); detail.Code != http.StatusOK {
		t.Fatalf("detail = %d", detail.Code)
	}
	updated := authenticatedJSONRequest(router, http.MethodPatch, "/api/v1/posts/10", `{"version":1,"title":"新标题"}`, "")
	if updated.Code != http.StatusOK {
		t.Fatalf("update = %d %s", updated.Code, updated.Body.String())
	}
	deleted := performAuthenticatedRequest(router, http.MethodDelete, "/api/v1/posts/10", "", "")
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("delete = %d %q", deleted.Code, deleted.Body.String())
	}
}

func TestPostsHTTPRejectsMissingKeyForgedAuthorInvalidPatchAndMapsDomainErrors(t *testing.T) {
	application := &fakePostsApplication{create: func(context.Context, usecase.CreatePostInput) (domain.Post, error) {
		return domain.Post{}, domain.ErrMembershipRequired
	}, update: func(context.Context, usecase.UpdatePostInput) (domain.Post, error) {
		return domain.Post{}, domain.ErrVersionConflict
	}, get: func(context.Context, int64, int64) (domain.Post, error) { return domain.Post{}, domain.ErrPostNotFound }, remove: func(context.Context, int64, int64) error { return domain.ErrContentNotEditable }, list: func(context.Context, usecase.PostListInput) (usecase.PostPage, error) { return usecase.PostPage{}, nil }}
	router := newPostsTestRouter(t, application, mustCursorCodec(t, time.Now()), 42)

	tests := []struct {
		method, path, body, key string
		status                  int
		code                    string
	}{
		{http.MethodPost, "/api/v1/posts", `{"circle_id":7,"title":"标题","body":"正文","security_ids":[3]}`, "", 400, "invalid_request"},
		{http.MethodPost, "/api/v1/posts", `{"circle_id":7,"title":"标题","body":"正文","security_ids":[3],"author_id":99}`, "key", 400, "invalid_request"},
		{http.MethodPost, "/api/v1/posts", `{"circle_id":7,"title":"标题","body":"正文","security_ids":[3]}`, "key", 403, "membership_required"},
		{http.MethodPatch, "/api/v1/posts/10", `{"version":1}`, "", 400, "invalid_request"},
		{http.MethodPatch, "/api/v1/posts/10", `{"version":1,"title":"新标题"}`, "", 409, "version_conflict"},
		{http.MethodGet, "/api/v1/posts/10", "", "", 404, "not_found"},
		{http.MethodDelete, "/api/v1/posts/10", "", "", 409, "content_not_editable"},
	}
	for _, test := range tests {
		var response *httptest.ResponseRecorder
		if test.method == http.MethodPost || test.method == http.MethodPatch {
			response = authenticatedJSONRequest(router, test.method, test.path, test.body, test.key)
		} else {
			response = performAuthenticatedRequest(router, test.method, test.path, "", "")
		}
		if response.Code != test.status || decodeError(t, response).Code != test.code {
			t.Fatalf("%s %s = %d %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}

type fakePostsApplication struct {
	create func(context.Context, usecase.CreatePostInput) (domain.Post, error)
	list   func(context.Context, usecase.PostListInput) (usecase.PostPage, error)
	get    func(context.Context, int64, int64) (domain.Post, error)
	update func(context.Context, usecase.UpdatePostInput) (domain.Post, error)
	remove func(context.Context, int64, int64) error
}

func (a *fakePostsApplication) CreatePost(c context.Context, i usecase.CreatePostInput) (domain.Post, error) {
	return a.create(c, i)
}
func (a *fakePostsApplication) ListPosts(c context.Context, i usecase.PostListInput) (usecase.PostPage, error) {
	return a.list(c, i)
}
func (a *fakePostsApplication) GetPost(c context.Context, u, p int64) (domain.Post, error) {
	return a.get(c, u, p)
}
func (a *fakePostsApplication) UpdatePost(c context.Context, i usecase.UpdatePostInput) (domain.Post, error) {
	return a.update(c, i)
}
func (a *fakePostsApplication) DeletePost(c context.Context, u, p int64) error {
	return a.remove(c, u, p)
}

func newPostsTestRouter(t *testing.T, posts PostsApplication, codec *CursorCodec, userID int64) http.Handler {
	t.Helper()
	auth := &fakeAuthApplication{authenticate: func(context.Context, string) (domain.User, error) {
		return domain.User{ID: userID, Status: domain.UserStatusActive, Role: domain.RoleUser}, nil
	}}
	return NewRouterWithCommunityAndPosts(nil, time.Second, auth, nil, posts, codec)
}

func authenticatedJSONRequest(handler http.Handler, method, path, body, key string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer current-token")
	request.Header.Set("Content-Type", "application/json")
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
