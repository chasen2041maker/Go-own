package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestListCirclesReturnsPublicShapeAndCurrentMembership(t *testing.T) {
	createdAt := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	community := &fakeCommunityApplication{listCircles: func(_ context.Context, input usecase.CircleListInput) (usecase.CirclePage, error) {
		if input.UserID != 42 || input.Limit != 20 || input.After != nil {
			t.Fatalf("ListCircles() input = %#v", input)
		}
		return usecase.CirclePage{Items: []domain.Circle{{
			ID: 3, Slug: "risk-lab", Name: "风险实验室", Description: "研究虚构情境中的风险",
			MemberCount: 5, IsMember: true, CreatedAt: createdAt,
		}}}, nil
	}}
	router := newCommunityTestRouter(t, community, mustCursorCodec(t, createdAt), 42)

	response := performAuthenticatedRequest(router, http.MethodGet, "/api/v1/circles", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items := body["items"].([]any)
	circle := items[0].(map[string]any)
	if circle["is_member"] != true || circle["member_count"] != float64(5) {
		t.Fatalf("circle = %#v", circle)
	}
	if _, exists := circle["status"]; exists {
		t.Fatalf("public circle leaked status: %#v", circle)
	}
}

func TestListCirclesPaginatesWithLimitBoundCursor(t *testing.T) {
	createdAt := time.Date(2026, 8, 19, 10, 0, 0, 123000000, time.UTC)
	calls := 0
	community := &fakeCommunityApplication{listCircles: func(_ context.Context, input usecase.CircleListInput) (usecase.CirclePage, error) {
		calls++
		if input.UserID != 42 || input.Limit != 1 {
			t.Fatalf("ListCircles() input = %#v", input)
		}
		if calls == 1 {
			if input.After != nil {
				t.Fatalf("first page after = %#v", input.After)
			}
			return usecase.CirclePage{
				Items: []domain.Circle{{ID: 9, Slug: "risk-lab", Name: "风险实验室", CreatedAt: createdAt}},
				Next:  &domain.CircleCursor{CreatedAt: createdAt, ID: 9},
			}, nil
		}
		if input.After == nil || input.After.ID != 9 || !input.After.CreatedAt.Equal(createdAt) {
			t.Fatalf("second page after = %#v", input.After)
		}
		return usecase.CirclePage{Items: []domain.Circle{{
			ID: 8, Slug: "long-horizon", Name: "长期观察", CreatedAt: createdAt.Add(-time.Second),
		}}}, nil
	}}
	router := newCommunityTestRouter(t, community, mustCursorCodec(t, createdAt), 42)

	first := performAuthenticatedRequest(router, http.MethodGet, "/api/v1/circles?limit=1", "", "")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d; body = %s", first.Code, first.Body.String())
	}
	var firstBody circleListResponse
	if err := json.NewDecoder(first.Body).Decode(&firstBody); err != nil {
		t.Fatalf("decode first page: %v", err)
	}
	if !firstBody.Page.HasMore || firstBody.Page.NextCursor == nil {
		t.Fatalf("first page = %#v", firstBody)
	}

	second := performAuthenticatedRequest(router, http.MethodGet,
		"/api/v1/circles?limit=1&cursor="+url.QueryEscape(*firstBody.Page.NextCursor), "", "")
	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d; body = %s", second.Code, second.Body.String())
	}
	changedLimit := performAuthenticatedRequest(router, http.MethodGet,
		"/api/v1/circles?limit=2&cursor="+url.QueryEscape(*firstBody.Page.NextCursor), "", "")
	if changedLimit.Code != http.StatusBadRequest || decodeError(t, changedLimit).Code != "invalid_cursor" {
		t.Fatalf("changed-limit response = %d %s", changedLimit.Code, changedLimit.Body.String())
	}
	if calls != 2 {
		t.Fatalf("application calls = %d, want 2", calls)
	}
}

func TestSetCircleMembershipUsesJWTIdentityAndReturnsJoinThenLeaveState(t *testing.T) {
	joinedAt := time.Date(2026, 8, 19, 10, 30, 0, 0, time.UTC)
	community := &fakeCommunityApplication{setMembership: func(_ context.Context, input usecase.SetCircleMembershipInput) (domain.CircleMembership, error) {
		if input.UserID != 42 || input.CircleID != 7 {
			t.Fatalf("SetCircleMembership() input = %#v", input)
		}
		membership := domain.CircleMembership{CircleID: 7, UserID: 42, Joined: input.Joined}
		if input.Joined {
			membership.JoinedAt = &joinedAt
		}
		return membership, nil
	}}
	router := newCommunityTestRouter(t, community, mustCursorCodec(t, joinedAt), 42)

	joined := performAuthenticatedRequest(router, http.MethodPut, "/api/v1/circles/7/membership", `{"joined":true}`, "application/json")
	if joined.Code != http.StatusOK {
		t.Fatalf("join status = %d; body = %s", joined.Code, joined.Body.String())
	}
	var joinedBody circleMembershipResponse
	if err := json.NewDecoder(joined.Body).Decode(&joinedBody); err != nil {
		t.Fatalf("decode join: %v", err)
	}
	if joinedBody.UserID != 42 || !joinedBody.Joined || joinedBody.JoinedAt == nil || !joinedBody.JoinedAt.Equal(joinedAt) {
		t.Fatalf("join body = %#v", joinedBody)
	}

	left := performAuthenticatedRequest(router, http.MethodPut, "/api/v1/circles/7/membership", `{"joined":false}`, "application/json")
	if left.Code != http.StatusOK {
		t.Fatalf("leave status = %d; body = %s", left.Code, left.Body.String())
	}
	var leftBody circleMembershipResponse
	if err := json.NewDecoder(left.Body).Decode(&leftBody); err != nil {
		t.Fatalf("decode leave: %v", err)
	}
	if leftBody.Joined || leftBody.JoinedAt != nil {
		t.Fatalf("leave body = %#v", leftBody)
	}
}

func TestSetCircleMembershipRequiresStrictJSONAndValidPathID(t *testing.T) {
	calls := 0
	community := &fakeCommunityApplication{setMembership: func(context.Context, usecase.SetCircleMembershipInput) (domain.CircleMembership, error) {
		calls++
		return domain.CircleMembership{}, nil
	}}
	router := newCommunityTestRouter(t, community, mustCursorCodec(t, time.Now()), 42)

	tests := []struct {
		name        string
		path        string
		body        string
		contentType string
		wantStatus  int
		wantCode    string
	}{
		{name: "empty body", path: "/api/v1/circles/7/membership", body: "", contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_json"},
		{name: "missing joined", path: "/api/v1/circles/7/membership", body: `{}`, contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "null joined", path: "/api/v1/circles/7/membership", body: `{"joined":null}`, contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "forged user", path: "/api/v1/circles/7/membership", body: `{"joined":true,"user_id":99}`, contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "missing content type", path: "/api/v1/circles/7/membership", body: `{"joined":true}`, wantStatus: http.StatusUnsupportedMediaType, wantCode: "unsupported_media_type"},
		{name: "body over one MiB", path: "/api/v1/circles/7/membership", body: `{"joined":true,"padding":"` + strings.Repeat("x", (1<<20)+1) + `"}`, contentType: "application/json", wantStatus: http.StatusRequestEntityTooLarge, wantCode: "payload_too_large"},
		{name: "zero id", path: "/api/v1/circles/0/membership", body: `{"joined":true}`, contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "negative id", path: "/api/v1/circles/-1/membership", body: `{"joined":true}`, contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "decimal id", path: "/api/v1/circles/1.5/membership", body: `{"joined":true}`, contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "uuid id", path: "/api/v1/circles/not-an-id/membership", body: `{"joined":true}`, contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "overflow id", path: "/api/v1/circles/9223372036854775808/membership", body: `{"joined":true}`, contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performAuthenticatedRequest(router, http.MethodPut, test.path, test.body, test.contentType)
			if response.Code != test.wantStatus || decodeError(t, response).Code != test.wantCode {
				t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
			}
		})
	}
	if calls != 0 {
		t.Fatalf("application calls = %d, want 0", calls)
	}
}

func TestSetCircleMembershipMapsMissingCircleToNotFound(t *testing.T) {
	community := &fakeCommunityApplication{setMembership: func(context.Context, usecase.SetCircleMembershipInput) (domain.CircleMembership, error) {
		return domain.CircleMembership{}, domain.ErrCircleNotFound
	}}
	router := newCommunityTestRouter(t, community, mustCursorCodec(t, time.Now()), 42)

	response := performAuthenticatedRequest(router, http.MethodPut, "/api/v1/circles/999/membership", `{"joined":true}`, "application/json")
	if response.Code != http.StatusNotFound || decodeError(t, response).Code != "not_found" {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
