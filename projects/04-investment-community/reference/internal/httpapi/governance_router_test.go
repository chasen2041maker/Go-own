package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

func TestFullRouterRegistersGovernanceRoutes(t *testing.T) {
	called := false
	governance := &fakeGovernanceApplication{decide: func(_ context.Context, input usecase.DecideReportInput) (domain.AdminReport, error) {
		called = true
		if input.Admin.ID != 7 || input.ReportID != 9 || input.Decision != domain.ReportDecisionIgnore {
			t.Fatalf("input = %+v", input)
		}
		return domain.AdminReport{ID: 9}, nil
	}}
	router := NewRouterWithGovernance(nil, time.Second, adminAuth(), nil, nil, nil, nil, governance, mustCursorCodec(t, time.Now()))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reports/9/decision", strings.NewReader(`{"decision":"ignore"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !called {
		t.Fatalf("status = %d, called = %v, body = %s", response.Code, called, response.Body.String())
	}
}

func TestFullRouterMapsGovernanceErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		error      error
		wantStatus int
		wantCode   string
	}{
		{name: "report already decided", path: "/api/v1/admin/reports/9/decision", body: `{"decision":"hide"}`, error: domain.ErrReportAlreadyDecided, wantStatus: http.StatusConflict, wantCode: "report_already_decided"},
		{name: "moderation version changed", path: "/api/v1/admin/content/post/5/restore", body: `{"expected_moderation_version":1}`, error: domain.ErrModerationVersionConflict, wantStatus: http.StatusConflict, wantCode: "moderation_version_conflict"},
		{name: "content cannot be restored", path: "/api/v1/admin/content/post/5/restore", body: `{"expected_moderation_version":1}`, error: domain.ErrContentNotRestorable, wantStatus: http.StatusConflict, wantCode: "content_not_restorable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			governance := &fakeGovernanceApplication{
				decide: func(context.Context, usecase.DecideReportInput) (domain.AdminReport, error) {
					return domain.AdminReport{}, fmt.Errorf("repository: %w", test.error)
				},
				restore: func(context.Context, usecase.RestoreContentInput) (domain.RestoredContent, error) {
					return domain.RestoredContent{}, fmt.Errorf("repository: %w", test.error)
				},
			}
			router := NewRouterWithGovernance(nil, time.Second, adminAuth(), nil, nil, nil, nil, governance, mustCursorCodec(t, time.Now()))
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer token")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if got := decodeError(t, response).Code; got != test.wantCode {
				t.Fatalf("error code = %q, want %q", got, test.wantCode)
			}
		})
	}
}

func TestAuditCursorIsBoundToViewerAndFiltersInFullRouter(t *testing.T) {
	now := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	listCalls := 0
	governance := &fakeGovernanceApplication{list: func(_ context.Context, input usecase.AuditListInput) (usecase.AuditPage, error) {
		listCalls++
		return usecase.AuditPage{Next: &domain.AuditCursor{CreatedAt: now, ID: 11}}, nil
	}}
	router := NewRouterWithGovernance(nil, time.Second, adminAuth(), nil, nil, nil, nil, governance, mustCursorCodec(t, now))
	first := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?action=content_hidden&target_type=post&admin_id=7&limit=20", nil)
	first.Header.Set("Authorization", "Bearer token")
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first status = %d, body = %s", firstResponse.Code, firstResponse.Body.String())
	}
	var page auditLogListResponse
	if err := json.NewDecoder(firstResponse.Body).Decode(&page); err != nil {
		t.Fatalf("decode first page: %v", err)
	}
	if page.Page.NextCursor == nil {
		t.Fatal("next cursor is nil")
	}

	// 游标必须连同筛选条件签名，否则客户端可把同一位置挪到另一查询，造成跳页或重复。
	secondPath := "/api/v1/admin/audit-logs?action=content_restored&target_type=post&admin_id=7&limit=20&cursor=" + url.QueryEscape(*page.Page.NextCursor)
	second := httptest.NewRequest(http.MethodGet, secondPath, nil)
	second.Header.Set("Authorization", "Bearer token")
	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, second)

	if secondResponse.Code != http.StatusBadRequest || decodeError(t, secondResponse).Code != "invalid_cursor" {
		t.Fatalf("status = %d, body = %s", secondResponse.Code, secondResponse.Body.String())
	}
	if listCalls != 1 {
		t.Fatalf("list calls = %d, want 1", listCalls)
	}

	otherAdminAuth := &fakeAuthApplication{authenticate: func(context.Context, string) (domain.User, error) {
		return domain.User{ID: 8, DisplayName: "另一管理员", Role: domain.RoleAdmin, Status: domain.UserStatusActive}, nil
	}}
	otherAdminRouter := NewRouterWithGovernance(nil, time.Second, otherAdminAuth, nil, nil, nil, nil, governance, mustCursorCodec(t, now))
	otherAdmin := httptest.NewRequest(http.MethodGet,
		"/api/v1/admin/audit-logs?action=content_hidden&target_type=post&admin_id=7&limit=20&cursor="+url.QueryEscape(*page.Page.NextCursor), nil)
	otherAdmin.Header.Set("Authorization", "Bearer token")
	otherAdminResponse := httptest.NewRecorder()
	otherAdminRouter.ServeHTTP(otherAdminResponse, otherAdmin)
	if otherAdminResponse.Code != http.StatusBadRequest || decodeError(t, otherAdminResponse).Code != "invalid_cursor" {
		t.Fatalf("other admin status = %d, body = %s", otherAdminResponse.Code, otherAdminResponse.Body.String())
	}
	if listCalls != 1 {
		t.Fatalf("list calls after viewer mismatch = %d, want 1", listCalls)
	}
}
