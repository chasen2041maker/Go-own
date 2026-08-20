package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

type GovernanceApplication interface {
	DecideReport(context.Context, usecase.DecideReportInput) (domain.AdminReport, error)
	RestoreContent(context.Context, usecase.RestoreContentInput) (domain.RestoredContent, error)
	ListAuditLogs(context.Context, usecase.AuditListInput) (usecase.AuditPage, error)
}

type decisionRequest struct {
	Decision domain.ReportDecision `json:"decision"`
	Note     string                `json:"note"`
}

type restoreContentRequest struct {
	ExpectedModerationVersion int64 `json:"expected_moderation_version"`
}

type restoredContentResponse struct {
	TargetType        domain.ContentType `json:"target_type"`
	TargetID          int64              `json:"target_id"`
	Visibility        domain.Visibility  `json:"visibility"`
	ModerationVersion int64              `json:"moderation_version"`
	RestoredAt        time.Time          `json:"restored_at"`
}

type auditLogResponse struct {
	ID         int64              `json:"id"`
	Admin      publicUserResponse `json:"admin"`
	Action     domain.AuditAction `json:"action"`
	TargetType domain.ContentType `json:"target_type"`
	TargetID   int64              `json:"target_id"`
	ReportID   *int64             `json:"report_id"`
	Note       *string            `json:"note"`
	CreatedAt  time.Time          `json:"created_at"`
}

type auditLogListResponse struct {
	Items []auditLogResponse `json:"items"`
	Page  cursorPageResponse `json:"page"`
}

type governanceHandler struct {
	application GovernanceApplication
	cursors     *CursorCodec
}

func registerGovernanceRoutes(mux *http.ServeMux, auth AuthApplication, application GovernanceApplication, cursors *CursorCodec) {
	handler := governanceHandler{application: application, cursors: cursors}
	admin := func(next http.HandlerFunc) http.Handler { return authenticate(auth, RequireAdmin(next)) }
	mux.Handle("/api/v1/admin/reports/{reportId}/decision", requireMethod(http.MethodPost, admin(handler.decide)))
	mux.Handle("/api/v1/admin/content/{targetType}/{targetId}/restore", requireMethod(http.MethodPost, admin(handler.restore)))
	mux.Handle("/api/v1/admin/audit-logs", requireMethod(http.MethodGet, admin(handler.listAudits)))
}

func (handler governanceHandler) decide(writer http.ResponseWriter, request *http.Request) {
	admin, _ := CurrentUserFromContext(request.Context())
	reportID, failure := parsePathID(request.PathValue("reportId"), "reportId")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	var body decisionRequest
	if failure := decodeJSON(writer, request, &body); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	// author_deleted 是系统收口值，不是管理员动作；在 HTTP 边界固定为协议级 400。
	if body.Decision != domain.ReportDecisionIgnore && body.Decision != domain.ReportDecisionHide {
		WriteError(writer, request, http.StatusBadRequest, "invalid_request", "decision 必须是 ignore 或 hide", nil)
		return
	}
	report, err := handler.application.DecideReport(request.Context(), usecase.DecideReportInput{
		Admin: admin, ReportID: reportID, Decision: body.Decision, Note: body.Note,
		RequestID: RequestIDFromContext(request.Context()),
	})
	if err != nil {
		writeGovernanceError(writer, request, err)
		return
	}
	WriteJSON(writer, http.StatusOK, publicAdminReport(report))
}

func (handler governanceHandler) restore(writer http.ResponseWriter, request *http.Request) {
	admin, _ := CurrentUserFromContext(request.Context())
	targetType := domain.ContentType(request.PathValue("targetType"))
	if targetType != domain.ContentTypePost && targetType != domain.ContentTypeComment {
		WriteError(writer, request, http.StatusBadRequest, "invalid_request", "targetType 必须是 post 或 comment", nil)
		return
	}
	targetID, failure := parsePathID(request.PathValue("targetId"), "targetId")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	var body restoreContentRequest
	if failure := decodeJSON(writer, request, &body); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	result, err := handler.application.RestoreContent(request.Context(), usecase.RestoreContentInput{
		Admin: admin, TargetType: targetType, TargetID: targetID,
		ExpectedModerationVersion: body.ExpectedModerationVersion,
		RequestID:                 RequestIDFromContext(request.Context()),
	})
	if err != nil {
		writeGovernanceError(writer, request, err)
		return
	}
	WriteJSON(writer, http.StatusOK, restoredContentResponse{
		TargetType: result.TargetType, TargetID: result.TargetID, Visibility: result.Visibility,
		ModerationVersion: result.ModerationVersion, RestoredAt: result.RestoredAt.UTC(),
	})
}

func (handler governanceHandler) listAudits(writer http.ResponseWriter, request *http.Request) {
	admin, _ := CurrentUserFromContext(request.Context())
	values, failure := parseKnownQuery(request, map[string]struct{}{"cursor": {}, "limit": {}, "action": {}, "target_type": {}, "admin_id": {}})
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	limit, cursor, failure := parseSimplePage(values)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	actionRaw, failure := singleQueryValue(values, "action")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	targetRaw, failure := singleQueryValue(values, "target_type")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	adminRaw, failure := singleQueryValue(values, "admin_id")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	var adminID int64
	if adminRaw != "" {
		adminID, failure = parsePathID(adminRaw, "admin_id")
		if failure != nil {
			WriteError(writer, request, failure.status, failure.code, failure.message, nil)
			return
		}
	}
	binding := auditCursorBinding{ViewerID: admin.ID, Action: domain.AuditAction(actionRaw), TargetType: domain.ContentType(targetRaw), AdminID: adminID, Limit: limit}
	var after *domain.AuditCursor
	if cursor != "" {
		position, err := handler.cursors.DecodeAudit(cursor, binding)
		if err != nil {
			WriteError(writer, request, http.StatusBadRequest, "invalid_cursor", "分页游标无效", nil)
			return
		}
		after = &position
	}
	page, err := handler.application.ListAuditLogs(request.Context(), usecase.AuditListInput{
		Admin: admin, Action: binding.Action, TargetType: binding.TargetType, AdminID: binding.AdminID, After: after, Limit: limit,
	})
	if err != nil {
		writeGovernanceError(writer, request, err)
		return
	}
	response := auditLogListResponse{Items: make([]auditLogResponse, 0, len(page.Items)), Page: cursorPageResponse{HasMore: page.Next != nil}}
	for _, item := range page.Items {
		response.Items = append(response.Items, auditLogResponse{ID: item.ID, Admin: publicUserResponse{ID: item.Admin.ID, DisplayName: item.Admin.DisplayName}, Action: item.Action, TargetType: item.TargetType, TargetID: item.TargetID, ReportID: item.ReportID, Note: item.Note, CreatedAt: item.CreatedAt.UTC()})
	}
	if page.Next != nil {
		token, err := handler.cursors.EncodeAudit(binding, *page.Next)
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		response.Page.NextCursor = &token
	}
	WriteJSON(writer, http.StatusOK, response)
}

func writeGovernanceError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "请求字段未通过校验", []FieldViolation{{Field: validation.Field, Reason: validation.Reason}})
	case errors.Is(err, domain.ErrForbidden):
		WriteError(writer, request, http.StatusForbidden, "forbidden", "没有执行此操作的权限", nil)
	case errors.Is(err, domain.ErrReportNotFound), errors.Is(err, domain.ErrPostNotFound), errors.Is(err, domain.ErrCommentNotFound):
		WriteError(writer, request, http.StatusNotFound, "not_found", "目标不存在", nil)
	case errors.Is(err, domain.ErrReportAlreadyDecided):
		WriteError(writer, request, http.StatusConflict, "report_already_decided", "举报已由其他决策处理", nil)
	case errors.Is(err, domain.ErrModerationVersionConflict):
		WriteError(writer, request, http.StatusConflict, "moderation_version_conflict", "内容治理版本已变化", nil)
	case errors.Is(err, domain.ErrContentNotRestorable):
		WriteError(writer, request, http.StatusConflict, "content_not_restorable", "内容当前不可恢复", nil)
	default:
		writeInternalError(writer, request)
	}
}
