package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func TestDecisionRejectsNonAdminBeforeRepository(t *testing.T) {
	repository := &fakeGovernanceRepository{}
	service, err := NewGovernanceService(repository)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.DecideReport(context.Background(), DecideReportInput{
		Admin:    domain.User{ID: 7, Role: domain.RoleUser, Status: domain.UserStatusActive},
		ReportID: 9, Decision: domain.ReportDecisionHide, RequestID: "req-7",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("DecideReport error = %v, want forbidden", err)
	}
	if repository.decideCalls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.decideCalls)
	}
}

func TestDecisionValidatesInputAndForwardsServerIdentity(t *testing.T) {
	repository := &fakeGovernanceRepository{report: domain.AdminReport{ID: 9}}
	service, err := NewGovernanceService(repository)
	if err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: 7, Role: domain.RoleAdmin, Status: domain.UserStatusActive}

	_, err = service.DecideReport(context.Background(), DecideReportInput{
		Admin: admin, ReportID: 9, Decision: domain.ReportDecisionHide,
		Note: "  违反社区规则  ", RequestID: "req-7",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := repository.decision; got.AdminID != 7 || got.ReportID != 9 || got.Note != "违反社区规则" || got.RequestID != "req-7" {
		t.Fatalf("decision params = %+v", got)
	}

	_, err = service.DecideReport(context.Background(), DecideReportInput{Admin: admin, ReportID: 9, Decision: domain.ReportDecisionAuthorDeleted, RequestID: "req-8"})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || validation.Field != "decision" {
		t.Fatalf("invalid decision error = %v", err)
	}
}

func TestRestoreRejectsInvalidTargetTypeBeforeRepository(t *testing.T) {
	repository := &fakeGovernanceRepository{}
	service, err := NewGovernanceService(repository)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.RestoreContent(context.Background(), RestoreContentInput{
		Admin:      domain.User{ID: 7, Role: domain.RoleAdmin, Status: domain.UserStatusActive},
		TargetType: "video", TargetID: 3, ExpectedModerationVersion: 2, RequestID: "req-9",
	})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || validation.Field != "target_type" {
		t.Fatalf("RestoreContent error = %v", err)
	}
	if repository.restoreCalls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.restoreCalls)
	}
}

func TestAuditListUsesStableExtraRowPagination(t *testing.T) {
	now := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	repository := &fakeGovernanceRepository{audits: []domain.AuditLog{
		{ID: 3, CreatedAt: now}, {ID: 2, CreatedAt: now.Add(-time.Second)}, {ID: 1, CreatedAt: now.Add(-2 * time.Second)},
	}}
	service, err := NewGovernanceService(repository)
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.ListAuditLogs(context.Background(), AuditListInput{
		Admin:  domain.User{ID: 7, Role: domain.RoleAdmin, Status: domain.UserStatusActive},
		Action: domain.AuditActionContentHidden, TargetType: domain.ContentTypePost, AdminID: 8, Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Next == nil || page.Next.ID != 2 || repository.auditQuery.Limit != 3 {
		t.Fatalf("page = %+v, query = %+v", page, repository.auditQuery)
	}
}

type fakeGovernanceRepository struct {
	decideCalls  int
	restoreCalls int
	report       domain.AdminReport
	decision     domain.DecideReportParams
	restore      domain.RestoreContentParams
	audits       []domain.AuditLog
	auditQuery   domain.AuditListQuery
}

func (repository *fakeGovernanceRepository) DecideReport(_ context.Context, input domain.DecideReportParams) (domain.AdminReport, error) {
	repository.decideCalls++
	repository.decision = input
	return repository.report, nil
}

func (repository *fakeGovernanceRepository) RestoreContent(_ context.Context, input domain.RestoreContentParams) (domain.RestoredContent, error) {
	repository.restoreCalls++
	repository.restore = input
	return domain.RestoredContent{}, nil
}

func (repository *fakeGovernanceRepository) ListAuditLogs(_ context.Context, query domain.AuditListQuery) ([]domain.AuditLog, error) {
	repository.auditQuery = query
	return repository.audits, nil
}
