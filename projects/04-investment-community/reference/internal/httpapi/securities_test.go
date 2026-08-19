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

func TestListSecuritiesAuthenticatesAndPaginatesWithBoundCursor(t *testing.T) {
	now := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	codec := mustCursorCodec(t, now)
	calls := 0
	community := &fakeCommunityApplication{listSecurities: func(_ context.Context, input usecase.SecurityListInput) (usecase.SecurityPage, error) {
		calls++
		if input.Query != "NO" || input.Exchange != "XSEA" || input.Limit != 1 {
			t.Fatalf("ListSecurities() input = %#v", input)
		}
		if calls == 1 {
			if input.After != nil {
				t.Fatalf("first page after = %#v", input.After)
			}
			return usecase.SecurityPage{
				Items: []domain.Security{{ID: 7, Code: "NOVA", Name: "新星科技", Exchange: "XSEA"}},
				Next:  &domain.SecurityCursor{Code: "NOVA", ID: 7},
			}, nil
		}
		if input.After == nil || input.After.Code != "NOVA" || input.After.ID != 7 {
			t.Fatalf("next page after = %#v", input.After)
		}
		return usecase.SecurityPage{Items: []domain.Security{{ID: 8, Code: "NOVB", Name: "新湾材料", Exchange: "XSEA"}}}, nil
	}}
	router := newCommunityTestRouter(t, community, codec, 42)

	first := performAuthenticatedRequest(router, http.MethodGet, "/api/v1/securities?q=%20NO%20&exchange=%20XSEA%20&limit=1", "", "")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d; body = %s", first.Code, first.Body.String())
	}
	var firstBody securityListResponse
	if err := json.NewDecoder(first.Body).Decode(&firstBody); err != nil {
		t.Fatalf("decode first page: %v", err)
	}
	if len(firstBody.Items) != 1 || !firstBody.Page.HasMore || firstBody.Page.NextCursor == nil {
		t.Fatalf("first page = %#v", firstBody)
	}
	if firstBody.Items[0].Code != "NOVA" || firstBody.Items[0].Exchange != "XSEA" {
		t.Fatalf("first item = %#v", firstBody.Items[0])
	}

	secondPath := "/api/v1/securities?q=NO&exchange=XSEA&limit=1&cursor=" + url.QueryEscape(*firstBody.Page.NextCursor)
	second := performAuthenticatedRequest(router, http.MethodGet, secondPath, "", "")
	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d; body = %s", second.Code, second.Body.String())
	}
	var secondBody securityListResponse
	if err := json.NewDecoder(second.Body).Decode(&secondBody); err != nil {
		t.Fatalf("decode second page: %v", err)
	}
	if len(secondBody.Items) != 1 || secondBody.Page.HasMore || secondBody.Page.NextCursor != nil {
		t.Fatalf("second page = %#v", secondBody)
	}

	changed := performAuthenticatedRequest(router, http.MethodGet,
		"/api/v1/securities?q=TI&exchange=XSEA&limit=1&cursor="+url.QueryEscape(*firstBody.Page.NextCursor), "", "")
	if changed.Code != http.StatusBadRequest || decodeError(t, changed).Code != "invalid_cursor" {
		t.Fatalf("changed-filter response = %d %s", changed.Code, changed.Body.String())
	}
	if calls != 2 {
		t.Fatalf("application calls = %d, want 2", calls)
	}
}

func TestListSecuritiesRejectsUnauthenticatedInvalidQueryAndTamperedCursor(t *testing.T) {
	now := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	codec := mustCursorCodec(t, now)
	community := &fakeCommunityApplication{listSecurities: func(context.Context, usecase.SecurityListInput) (usecase.SecurityPage, error) {
		t.Fatal("invalid request must not call the application")
		return usecase.SecurityPage{}, nil
	}}
	router := newCommunityTestRouter(t, community, codec, 42)

	unauthenticated := httptest.NewRecorder()
	router.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/v1/securities", nil))
	if unauthenticated.Code != http.StatusUnauthorized || decodeError(t, unauthenticated).Code != "unauthenticated" {
		t.Fatalf("unauthenticated response = %d %s", unauthenticated.Code, unauthenticated.Body.String())
	}

	for _, path := range []string{
		"/api/v1/securities?limit=0",
		"/api/v1/securities?limit=101",
		"/api/v1/securities?limit=abc",
		"/api/v1/securities?unknown=value",
		"/api/v1/securities?cursor=not-a-signed-cursor",
	} {
		response := performAuthenticatedRequest(router, http.MethodGet, path, "", "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d; body = %s", path, response.Code, response.Body.String())
		}
	}
}

type fakeCommunityApplication struct {
	listSecurities func(context.Context, usecase.SecurityListInput) (usecase.SecurityPage, error)
	listCircles    func(context.Context, usecase.CircleListInput) (usecase.CirclePage, error)
	setMembership  func(context.Context, usecase.SetCircleMembershipInput) (domain.CircleMembership, error)
}

func (application *fakeCommunityApplication) ListSecurities(ctx context.Context, input usecase.SecurityListInput) (usecase.SecurityPage, error) {
	if application.listSecurities == nil {
		return usecase.SecurityPage{}, nil
	}
	return application.listSecurities(ctx, input)
}

func (application *fakeCommunityApplication) ListCircles(ctx context.Context, input usecase.CircleListInput) (usecase.CirclePage, error) {
	if application.listCircles == nil {
		return usecase.CirclePage{}, nil
	}
	return application.listCircles(ctx, input)
}

func (application *fakeCommunityApplication) SetCircleMembership(ctx context.Context, input usecase.SetCircleMembershipInput) (domain.CircleMembership, error) {
	if application.setMembership == nil {
		return domain.CircleMembership{}, nil
	}
	return application.setMembership(ctx, input)
}

func newCommunityTestRouter(t *testing.T, community CommunityApplication, codec *CursorCodec, userID int64) http.Handler {
	t.Helper()
	auth := &fakeAuthApplication{authenticate: func(context.Context, string) (domain.User, error) {
		return domain.User{ID: userID, Role: domain.RoleUser, Status: domain.UserStatusActive}, nil
	}}
	return NewRouterWithCommunity(nil, time.Second, auth, community, codec)
}

func performAuthenticatedRequest(handler http.Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, stringsReader(body))
	request.Header.Set("Authorization", "Bearer current-token")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func stringsReader(value string) *strings.Reader {
	return strings.NewReader(value)
}
