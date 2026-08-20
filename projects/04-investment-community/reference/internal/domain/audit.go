package domain

import (
	"errors"
	"time"
)

var (
	ErrReportAlreadyDecided      = errors.New("report already decided")
	ErrContentNotRestorable      = errors.New("content not restorable")
	ErrModerationVersionConflict = errors.New("moderation version conflict")
)

type AuditAction string

const (
	AuditActionReportIgnored   AuditAction = "report_ignored"
	AuditActionContentHidden   AuditAction = "content_hidden"
	AuditActionContentRestored AuditAction = "content_restored"
)

type AuditLog struct {
	ID         int64
	Admin      PublicUser
	Action     AuditAction
	TargetType ContentType
	TargetID   int64
	ReportID   *int64
	Note       *string
	CreatedAt  time.Time
}

type AuditCursor struct {
	CreatedAt time.Time
	ID        int64
}

type AuditListQuery struct {
	Action     AuditAction
	TargetType ContentType
	AdminID    int64
	After      *AuditCursor
	Limit      int
}

type DecideReportParams struct {
	AdminID   int64
	ReportID  int64
	Decision  ReportDecision
	Note      string
	RequestID string
}

type RestoreContentParams struct {
	AdminID                   int64
	TargetType                ContentType
	TargetID                  int64
	ExpectedModerationVersion int64
	RequestID                 string
}

type RestoredContent struct {
	TargetType        ContentType
	TargetID          int64
	Visibility        Visibility
	ModerationVersion int64
	RestoredAt        time.Time
}
