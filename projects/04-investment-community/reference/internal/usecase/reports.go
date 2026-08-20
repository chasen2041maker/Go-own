package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

type ReportsRepository interface {
	CreateReport(context.Context, domain.CreateReportParams) (domain.ReportReceipt, bool, error)
	ListReports(context.Context, domain.ReportListQuery) ([]domain.AdminReport, error)
}

type CreateReportInput struct {
	ReporterID int64
	TargetType domain.ContentType
	TargetID   int64
	Reason     domain.ReportReason
	Details    string
}

type ReportListInput struct {
	Admin      domain.User
	Status     domain.ReportStatus
	TargetType domain.ContentType
	After      *domain.ReportCursor
	Limit      int
}

type ReportPage struct {
	Items []domain.AdminReport
	Next  *domain.ReportCursor
}
type CreateReportResult struct {
	Report   domain.ReportReceipt
	Existing bool
}

type ReportService struct{ repository ReportsRepository }

func NewReportService(repository ReportsRepository) (*ReportService, error) {
	if repository == nil {
		return nil, errors.New("report service repository is required")
	}
	return &ReportService{repository: repository}, nil
}

func (service *ReportService) CreateReport(ctx context.Context, input CreateReportInput) (CreateReportResult, error) {
	if err := validatePositiveID("reporter_id", input.ReporterID); err != nil {
		return CreateReportResult{}, err
	}
	if err := validatePositiveID("target_id", input.TargetID); err != nil {
		return CreateReportResult{}, err
	}
	if input.TargetType != domain.ContentTypePost && input.TargetType != domain.ContentTypeComment {
		return CreateReportResult{}, &domain.ValidationError{Field: "target_type", Reason: "必须是 post 或 comment"}
	}
	if !validReportReason(input.Reason) {
		return CreateReportResult{}, &domain.ValidationError{Field: "reason", Reason: "举报原因不在允许列表"}
	}
	details := strings.TrimSpace(input.Details)
	if !utf8.ValidString(details) || utf8.RuneCountInString(details) > 1000 {
		return CreateReportResult{}, &domain.ValidationError{Field: "details", Reason: "最多 1000 个字符"}
	}
	report, existing, err := service.repository.CreateReport(ctx, domain.CreateReportParams{
		ReporterID: input.ReporterID, TargetType: input.TargetType, TargetID: input.TargetID,
		Reason: input.Reason, Details: details,
	})
	if err != nil {
		return CreateReportResult{}, fmt.Errorf("create report: %w", err)
	}
	return CreateReportResult{Report: report, Existing: existing}, nil
}

func (service *ReportService) ListReports(ctx context.Context, input ReportListInput) (ReportPage, error) {
	if input.Admin.ID <= 0 || input.Admin.Role != domain.RoleAdmin || input.Admin.Status != domain.UserStatusActive {
		return ReportPage{}, domain.ErrForbidden
	}
	if input.Status != "" && input.Status != domain.ReportStatusPending && input.Status != domain.ReportStatusIgnored && input.Status != domain.ReportStatusResolved {
		return ReportPage{}, &domain.ValidationError{Field: "status", Reason: "举报状态无效"}
	}
	if input.TargetType != "" && input.TargetType != domain.ContentTypePost && input.TargetType != domain.ContentTypeComment {
		return ReportPage{}, &domain.ValidationError{Field: "target_type", Reason: "必须是 post 或 comment"}
	}
	limit, err := validatePageLimit(input.Limit)
	if err != nil {
		return ReportPage{}, err
	}
	if input.After != nil && (input.After.ID <= 0 || input.After.CreatedAt.IsZero()) {
		return ReportPage{}, &domain.ValidationError{Field: "cursor", Reason: "分页位置无效"}
	}
	items, err := service.repository.ListReports(ctx, domain.ReportListQuery{Status: input.Status, TargetType: input.TargetType, After: input.After, Limit: limit + 1})
	if err != nil {
		return ReportPage{}, fmt.Errorf("list reports: %w", err)
	}
	page := ReportPage{Items: items}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.Next = &domain.ReportCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return page, nil
}

func validReportReason(reason domain.ReportReason) bool {
	switch reason {
	case domain.ReportReasonSpam, domain.ReportReasonHarassment, domain.ReportReasonMisleading, domain.ReportReasonIllegal, domain.ReportReasonOther:
		return true
	default:
		return false
	}
}
