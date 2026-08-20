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

func TestDecisionHTTPUsesAuthenticatedAdminAndRequestID(t *testing.T) {
	auth := adminAuth()
	application := &fakeGovernanceApplication{decide: func(_ context.Context, input usecase.DecideReportInput) (domain.AdminReport, error) {
		if input.Admin.ID != 7 || input.ReportID != 9 || input.Decision != domain.ReportDecisionHide || input.Note != "违规" || input.RequestID != "governance-1" {
			t.Fatalf("input = %+v", input)
		}
		now := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
		decision := domain.ReportDecisionHide
		return domain.AdminReport{ID: 9, Reporter: domain.PublicUser{ID: 3, DisplayName: "举报者"},
			Target: domain.ContentSnapshot{TargetType: domain.ContentTypePost, ID: 5, Visibility: domain.VisibilityHidden, ModerationVersion: 2},
			Reason: domain.ReportReasonSpam, Status: domain.ReportStatusResolved, Decision: &decision,
			DecidedBy: &domain.PublicUser{ID: 7, DisplayName: "管理员"}, CreatedAt: now, DecidedAt: &now}, nil
	}}
	handler := governanceTestHandler(t, auth, application)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reports/9/decision", strings.NewReader(`{"decision":"hide","note":"违规"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set(HeaderRequestID, "governance-1")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestRestoreHTTPRejectsInvalidTargetTypeBeforeApplication(t *testing.T) {
	application := &fakeGovernanceApplication{}
	handler := governanceTestHandler(t, adminAuth(), application)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/video/3/restore", strings.NewReader(`{"expected_moderation_version":2}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || decodeError(t, response).Code != "invalid_request" {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestAuditHTTPDoesNotExposeAdminEmail(t *testing.T) {
	application := &fakeGovernanceApplication{list: func(_ context.Context, input usecase.AuditListInput) (usecase.AuditPage, error) {
		if input.Admin.ID != 7 || input.Action != domain.AuditActionContentHidden || input.TargetType != domain.ContentTypePost || input.AdminID != 8 || input.Limit != 20 {
			t.Fatalf("input = %+v", input)
		}
		reportID := int64(9)
		note := "违规"
		return usecase.AuditPage{Items: []domain.AuditLog{{ID: 1, Admin: domain.PublicUser{ID: 8, DisplayName: "审核员"}, Action: domain.AuditActionContentHidden, TargetType: domain.ContentTypePost, TargetID: 5, ReportID: &reportID, Note: &note, CreatedAt: time.Now()}}}, nil
	}}
	handler := governanceTestHandler(t, adminAuth(), application)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?action=content_hidden&target_type=post&admin_id=8&limit=20", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	admin := body["items"].([]any)[0].(map[string]any)["admin"].(map[string]any)
	if _, exists := admin["email"]; exists {
		t.Fatal("audit response leaked admin email")
	}
}

func adminAuth() *fakeAuthApplication {
	return &fakeAuthApplication{authenticate: func(context.Context, string) (domain.User, error) {
		return domain.User{ID: 7, DisplayName: "管理员", Role: domain.RoleAdmin, Status: domain.UserStatusActive}, nil
	}}
}

func governanceTestHandler(t *testing.T, auth AuthApplication, application GovernanceApplication) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	registerGovernanceRoutes(mux, auth, application, mustCursorCodec(t, time.Now()))
	return WithRequestID(mux)
}

type fakeGovernanceApplication struct {
	decide  func(context.Context, usecase.DecideReportInput) (domain.AdminReport, error)
	restore func(context.Context, usecase.RestoreContentInput) (domain.RestoredContent, error)
	list    func(context.Context, usecase.AuditListInput) (usecase.AuditPage, error)
}

func (application *fakeGovernanceApplication) DecideReport(ctx context.Context, input usecase.DecideReportInput) (domain.AdminReport, error) {
	if application.decide == nil {
		return domain.AdminReport{}, nil
	}
	return application.decide(ctx, input)
}
func (application *fakeGovernanceApplication) RestoreContent(ctx context.Context, input usecase.RestoreContentInput) (domain.RestoredContent, error) {
	if application.restore == nil {
		return domain.RestoredContent{}, nil
	}
	return application.restore(ctx, input)
}
func (application *fakeGovernanceApplication) ListAuditLogs(ctx context.Context, input usecase.AuditListInput) (usecase.AuditPage, error) {
	if application.list == nil {
		return usecase.AuditPage{}, nil
	}
	return application.list(ctx, input)
}
