package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

type GovernanceRepository interface {
	DecideReport(context.Context, domain.DecideReportParams) (domain.AdminReport, error)
	RestoreContent(context.Context, domain.RestoreContentParams) (domain.RestoredContent, error)
	ListAuditLogs(context.Context, domain.AuditListQuery) ([]domain.AuditLog, error)
}

type DecideReportInput struct {
	Admin     domain.User
	ReportID  int64
	Decision  domain.ReportDecision
	Note      string
	RequestID string
}

type RestoreContentInput struct {
	Admin                     domain.User
	TargetType                domain.ContentType
	TargetID                  int64
	ExpectedModerationVersion int64
	RequestID                 string
}

type AuditListInput struct {
	Admin      domain.User
	Action     domain.AuditAction
	TargetType domain.ContentType
	AdminID    int64
	After      *domain.AuditCursor
	Limit      int
}

type AuditPage struct {
	Items []domain.AuditLog
	Next  *domain.AuditCursor
}

type GovernanceService struct{ repository GovernanceRepository }

func NewGovernanceService(repository GovernanceRepository) (*GovernanceService, error) {
	if repository == nil {
		return nil, errors.New("governance service repository is required")
	}
	return &GovernanceService{repository: repository}, nil
}

func (service *GovernanceService) DecideReport(ctx context.Context, input DecideReportInput) (domain.AdminReport, error) {
	if err := requireActiveAdmin(input.Admin); err != nil {
		return domain.AdminReport{}, err
	}
	if err := validatePositiveID("report_id", input.ReportID); err != nil {
		return domain.AdminReport{}, err
	}
	if input.Decision != domain.ReportDecisionIgnore && input.Decision != domain.ReportDecisionHide {
		return domain.AdminReport{}, &domain.ValidationError{Field: "decision", Reason: "必须是 ignore 或 hide"}
	}
	note, err := normalizeGovernanceNote(input.Note)
	if err != nil {
		return domain.AdminReport{}, err
	}
	if err := validateRequestID(input.RequestID); err != nil {
		return domain.AdminReport{}, err
	}
	report, err := service.repository.DecideReport(ctx, domain.DecideReportParams{
		AdminID: input.Admin.ID, ReportID: input.ReportID, Decision: input.Decision,
		Note: note, RequestID: input.RequestID,
	})
	if err != nil {
		return domain.AdminReport{}, fmt.Errorf("decide report: %w", err)
	}
	return report, nil
}

func (service *GovernanceService) RestoreContent(ctx context.Context, input RestoreContentInput) (domain.RestoredContent, error) {
	if err := requireActiveAdmin(input.Admin); err != nil {
		return domain.RestoredContent{}, err
	}
	if input.TargetType != domain.ContentTypePost && input.TargetType != domain.ContentTypeComment {
		return domain.RestoredContent{}, &domain.ValidationError{Field: "target_type", Reason: "必须是 post 或 comment"}
	}
	if err := validatePositiveID("target_id", input.TargetID); err != nil {
		return domain.RestoredContent{}, err
	}
	if err := validatePositiveID("expected_moderation_version", input.ExpectedModerationVersion); err != nil {
		return domain.RestoredContent{}, err
	}
	if err := validateRequestID(input.RequestID); err != nil {
		return domain.RestoredContent{}, err
	}
	result, err := service.repository.RestoreContent(ctx, domain.RestoreContentParams{
		AdminID: input.Admin.ID, TargetType: input.TargetType, TargetID: input.TargetID,
		ExpectedModerationVersion: input.ExpectedModerationVersion, RequestID: input.RequestID,
	})
	if err != nil {
		return domain.RestoredContent{}, fmt.Errorf("restore content: %w", err)
	}
	return result, nil
}

func (service *GovernanceService) ListAuditLogs(ctx context.Context, input AuditListInput) (AuditPage, error) {
	if err := requireActiveAdmin(input.Admin); err != nil {
		return AuditPage{}, err
	}
	if input.Action != "" && input.Action != domain.AuditActionReportIgnored && input.Action != domain.AuditActionContentHidden && input.Action != domain.AuditActionContentRestored {
		return AuditPage{}, &domain.ValidationError{Field: "action", Reason: "审计动作无效"}
	}
	if input.TargetType != "" && input.TargetType != domain.ContentTypePost && input.TargetType != domain.ContentTypeComment {
		return AuditPage{}, &domain.ValidationError{Field: "target_type", Reason: "必须是 post 或 comment"}
	}
	if input.AdminID < 0 {
		return AuditPage{}, &domain.ValidationError{Field: "admin_id", Reason: "必须是正 int64"}
	}
	if input.After != nil && (input.After.ID <= 0 || input.After.CreatedAt.IsZero()) {
		return AuditPage{}, &domain.ValidationError{Field: "cursor", Reason: "分页位置无效"}
	}
	limit, err := validatePageLimit(input.Limit)
	if err != nil {
		return AuditPage{}, err
	}
	items, err := service.repository.ListAuditLogs(ctx, domain.AuditListQuery{
		Action: input.Action, TargetType: input.TargetType, AdminID: input.AdminID,
		After: input.After, Limit: limit + 1,
	})
	if err != nil {
		return AuditPage{}, fmt.Errorf("list audit logs: %w", err)
	}
	page := AuditPage{Items: items}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.Next = &domain.AuditCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return page, nil
}

func requireActiveAdmin(user domain.User) error {
	if user.ID <= 0 || user.Role != domain.RoleAdmin || user.Status != domain.UserStatusActive {
		return domain.ErrForbidden
	}
	return nil
}

func normalizeGovernanceNote(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 1000 {
		return "", &domain.ValidationError{Field: "note", Reason: "最多 1000 个字符"}
	}
	return value, nil
}

func validateRequestID(value string) error {
	if len(value) < 1 || len(value) > 64 {
		return &domain.ValidationError{Field: "request_id", Reason: "请求追踪标识无效"}
	}
	return nil
}
