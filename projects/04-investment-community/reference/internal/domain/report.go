package domain

import (
	"errors"
	"time"
)

var (
	ErrReportNotFound      = errors.New("report not found")
	ErrSelfReportForbidden = errors.New("cannot report own content")
)

type ContentType string

const (
	ContentTypePost    ContentType = "post"
	ContentTypeComment ContentType = "comment"
)

type ReportReason string

const (
	ReportReasonSpam       ReportReason = "spam"
	ReportReasonHarassment ReportReason = "harassment"
	ReportReasonMisleading ReportReason = "misleading"
	ReportReasonIllegal    ReportReason = "illegal"
	ReportReasonOther      ReportReason = "other"
)

type ReportStatus string

const (
	ReportStatusPending  ReportStatus = "pending"
	ReportStatusIgnored  ReportStatus = "ignored"
	ReportStatusResolved ReportStatus = "resolved"
)

type ReportDecision string

const (
	ReportDecisionIgnore        ReportDecision = "ignore"
	ReportDecisionHide          ReportDecision = "hide"
	ReportDecisionAuthorDeleted ReportDecision = "author_deleted"
)

type ReportReceipt struct {
	ID         int64
	TargetType ContentType
	TargetID   int64
	Status     ReportStatus
	CreatedAt  time.Time
}

type ContentSnapshot struct {
	TargetType        ContentType
	ID                int64
	Visibility        Visibility
	ModerationVersion int64
	Deleted           bool
	Title             *string
	Excerpt           *string
}

type AdminReport struct {
	ID        int64
	Reporter  PublicUser
	Target    ContentSnapshot
	Reason    ReportReason
	Details   string
	Status    ReportStatus
	Decision  *ReportDecision
	DecidedBy *PublicUser
	CreatedAt time.Time
	DecidedAt *time.Time
}

type ReportCursor struct {
	CreatedAt time.Time
	ID        int64
}
type ReportListQuery struct {
	Status     ReportStatus
	TargetType ContentType
	After      *ReportCursor
	Limit      int
}

type CreateReportParams struct {
	ReporterID int64
	TargetType ContentType
	TargetID   int64
	Reason     ReportReason
	Details    string
}
