package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

type ReportsApplication interface {
	CreateReport(context.Context, usecase.CreateReportInput) (usecase.CreateReportResult, error)
	ListReports(context.Context, usecase.ReportListInput) (usecase.ReportPage, error)
}

type createReportRequest struct {
	TargetType domain.ContentType  `json:"target_type"`
	TargetID   int64               `json:"target_id"`
	Reason     domain.ReportReason `json:"reason"`
	Details    string              `json:"details"`
}
type reportReceiptResponse struct {
	ID         int64               `json:"id"`
	TargetType domain.ContentType  `json:"target_type"`
	TargetID   int64               `json:"target_id"`
	Status     domain.ReportStatus `json:"status"`
	CreatedAt  time.Time           `json:"created_at"`
}
type contentSnapshotResponse struct {
	TargetType        domain.ContentType `json:"target_type"`
	ID                int64              `json:"id"`
	Visibility        domain.Visibility  `json:"visibility"`
	ModerationVersion int64              `json:"moderation_version"`
	Deleted           bool               `json:"deleted"`
	Title             *string            `json:"title"`
	Excerpt           *string            `json:"excerpt"`
}
type adminReportResponse struct {
	ID            int64                   `json:"id"`
	Reporter      publicUserResponse      `json:"reporter"`
	Target        contentSnapshotResponse `json:"target"`
	Reason        domain.ReportReason     `json:"reason"`
	Details       *string                 `json:"details"`
	Status        domain.ReportStatus     `json:"status"`
	DecidedAction *domain.ReportDecision  `json:"decided_action"`
	DecidedBy     *publicUserResponse     `json:"decided_by"`
	CreatedAt     time.Time               `json:"created_at"`
	DecidedAt     *time.Time              `json:"decided_at"`
}
type adminReportListResponse struct {
	Items []adminReportResponse `json:"items"`
	Page  cursorPageResponse    `json:"page"`
}

type reportsHandler struct {
	application ReportsApplication
	cursors     *CursorCodec
}

func registerReportRoutes(mux *http.ServeMux, auth AuthApplication, application ReportsApplication, cursors *CursorCodec) {
	handler := reportsHandler{application: application, cursors: cursors}
	mux.Handle("/api/v1/reports", requireMethod(http.MethodPost, authenticate(auth, http.HandlerFunc(handler.create))))
	mux.Handle("/api/v1/admin/reports", requireMethod(http.MethodGet, authenticate(auth, RequireAdmin(http.HandlerFunc(handler.list)))))
}

func (handler reportsHandler) create(writer http.ResponseWriter, request *http.Request) {
	user, _ := CurrentUserFromContext(request.Context())
	var body createReportRequest
	if failure := decodeJSON(writer, request, &body); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	result, err := handler.application.CreateReport(request.Context(), usecase.CreateReportInput{ReporterID: user.ID, TargetType: body.TargetType, TargetID: body.TargetID, Reason: body.Reason, Details: body.Details})
	if err != nil {
		writeReportError(writer, request, err)
		return
	}
	status := http.StatusCreated
	if result.Existing {
		status = http.StatusOK
	}
	WriteJSON(writer, status, publicReportReceipt(result.Report))
}

func (handler reportsHandler) list(writer http.ResponseWriter, request *http.Request) {
	admin, _ := CurrentUserFromContext(request.Context())
	values, failure := parseKnownQuery(request, map[string]struct{}{"cursor": {}, "limit": {}, "status": {}, "target_type": {}})
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	limit, cursor, failure := parseSimplePage(values)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	statusRaw, failure := singleQueryValue(values, "status")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	targetRaw, failure := singleQueryValue(values, "target_type")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	binding := reportCursorBinding{AdminID: admin.ID, Status: domain.ReportStatus(statusRaw), TargetType: domain.ContentType(targetRaw), Limit: limit}
	var after *domain.ReportCursor
	if cursor != "" {
		decoded, err := handler.cursors.DecodeReport(cursor, binding)
		if err != nil {
			WriteError(writer, request, http.StatusBadRequest, "invalid_cursor", "分页游标无效", nil)
			return
		}
		after = &decoded
	}
	page, err := handler.application.ListReports(request.Context(), usecase.ReportListInput{Admin: admin, Status: binding.Status, TargetType: binding.TargetType, After: after, Limit: limit})
	if err != nil {
		writeReportError(writer, request, err)
		return
	}
	response := adminReportListResponse{Items: make([]adminReportResponse, 0, len(page.Items)), Page: cursorPageResponse{HasMore: page.Next != nil}}
	for _, item := range page.Items {
		response.Items = append(response.Items, publicAdminReport(item))
	}
	if page.Next != nil {
		token, err := handler.cursors.EncodeReport(binding, *page.Next)
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		response.Page.NextCursor = &token
	}
	WriteJSON(writer, http.StatusOK, response)
}

func publicReportReceipt(report domain.ReportReceipt) reportReceiptResponse {
	return reportReceiptResponse{ID: report.ID, TargetType: report.TargetType, TargetID: report.TargetID, Status: report.Status, CreatedAt: report.CreatedAt.UTC()}
}
func publicAdminReport(report domain.AdminReport) adminReportResponse {
	var details *string
	if report.Details != "" {
		details = &report.Details
	}
	var decidedBy *publicUserResponse
	if report.DecidedBy != nil {
		decidedBy = &publicUserResponse{ID: report.DecidedBy.ID, DisplayName: report.DecidedBy.DisplayName}
	}
	return adminReportResponse{ID: report.ID, Reporter: publicUserResponse{ID: report.Reporter.ID, DisplayName: report.Reporter.DisplayName}, Target: contentSnapshotResponse{TargetType: report.Target.TargetType, ID: report.Target.ID, Visibility: report.Target.Visibility, ModerationVersion: report.Target.ModerationVersion, Deleted: report.Target.Deleted, Title: report.Target.Title, Excerpt: report.Target.Excerpt}, Reason: report.Reason, Details: details, Status: report.Status, DecidedAction: report.Decision, DecidedBy: decidedBy, CreatedAt: report.CreatedAt.UTC(), DecidedAt: report.DecidedAt}
}

func writeReportError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "请求字段未通过校验", []FieldViolation{{Field: validation.Field, Reason: validation.Reason}})
	case errors.Is(err, domain.ErrSelfReportForbidden):
		WriteError(writer, request, http.StatusUnprocessableEntity, "self_report_forbidden", "不能举报自己的内容", nil)
	case errors.Is(err, domain.ErrPostNotFound), errors.Is(err, domain.ErrCommentNotFound), errors.Is(err, domain.ErrReportNotFound):
		WriteError(writer, request, http.StatusNotFound, "not_found", "目标不存在", nil)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(writer, request, http.StatusForbidden, "forbidden", "没有执行此操作的权限", nil)
	default:
		writeInternalError(writer, request)
	}
}
