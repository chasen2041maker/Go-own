package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	driver "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

const governanceTransactionAttempts = 3

type governanceTarget struct {
	targetType        domain.ContentType
	targetID          int64
	postID            int64
	authorID          int64
	visibility        domain.Visibility
	moderationVersion int64
	deleted           bool
}

func (store *Store) DecideReport(ctx context.Context, input domain.DecideReportParams) (domain.AdminReport, error) {
	var lastErr error
	for attempt := 0; attempt < governanceTransactionAttempts; attempt++ {
		report, err := store.decideReportOnce(ctx, input)
		if !retryableGovernanceError(err) {
			return report, err
		}
		lastErr = err
	}
	return domain.AdminReport{}, fmt.Errorf("decide report after retries: %w", lastErr)
}

func (store *Store) decideReportOnce(ctx context.Context, input domain.DecideReportParams) (domain.AdminReport, error) {
	targetType, targetID, err := store.reportTarget(ctx, input.ReportID)
	if err != nil {
		return domain.AdminReport{}, err
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AdminReport{}, fmt.Errorf("begin decision transaction: %w", err)
	}
	defer tx.Rollback()

	// 所有治理入口先锁内容，再按举报 ID 锁举报；统一顺序避免两个管理员形成锁环。
	target, err := lockGovernanceTarget(ctx, tx, targetType, targetID)
	if err != nil {
		return domain.AdminReport{}, err
	}
	reports, err := lockTargetReports(ctx, tx, targetType, targetID)
	if err != nil {
		return domain.AdminReport{}, err
	}
	lockedReport, found := reports[input.ReportID]
	if !found {
		return domain.AdminReport{}, domain.ErrReportNotFound
	}
	status, action := lockedReport[0], lockedReport[1]
	if status != "pending" {
		if externalReportDecision(action) != input.Decision {
			return domain.AdminReport{}, domain.ErrReportAlreadyDecided
		}
		if err := tx.Commit(); err != nil {
			return domain.AdminReport{}, fmt.Errorf("commit decision replay: %w", err)
		}
		return store.findAdminReportByID(ctx, store.db, input.ReportID)
	}

	now := time.Now().UTC()
	switch input.Decision {
	case domain.ReportDecisionIgnore:
		if _, err := tx.ExecContext(ctx, `UPDATE reports SET status='dismissed',resolution_action='dismiss',handled_by=?,handled_at=?,resolution_note=?,updated_at=? WHERE id=? AND status='pending'`, input.AdminID, now, input.Note, now, input.ReportID); err != nil {
			return domain.AdminReport{}, fmt.Errorf("dismiss report: %w", err)
		}
		if err := insertGovernanceAudit(ctx, tx, input, target, "report_dismissed", "pending", "dismissed", now); err != nil {
			return domain.AdminReport{}, err
		}
	case domain.ReportDecisionHide:
		if target.deleted || target.visibility != domain.VisibilityVisible {
			return domain.AdminReport{}, domain.ErrReportAlreadyDecided
		}
		table := "posts"
		if target.targetType == domain.ContentTypeComment {
			table = "comments"
		}
		if _, err := tx.ExecContext(ctx, `UPDATE `+table+` SET visibility='hidden',moderation_version=moderation_version+1,updated_at=? WHERE id=?`, now, target.targetID); err != nil {
			return domain.AdminReport{}, fmt.Errorf("hide content: %w", err)
		}
		column := "post_id"
		if target.targetType == domain.ContentTypeComment {
			column = "comment_id"
		}
		if _, err := tx.ExecContext(ctx, `UPDATE reports SET status='resolved',resolution_action='hide',handled_by=?,handled_at=?,resolution_note=?,updated_at=? WHERE `+column+`=? AND status='pending'`, input.AdminID, now, input.Note, now, target.targetID); err != nil {
			return domain.AdminReport{}, fmt.Errorf("resolve target reports: %w", err)
		}
		var commentID any
		if target.targetType == domain.ContentTypeComment {
			commentID = target.targetID
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO notifications (user_id,type,actor_user_id,post_id,comment_id,created_at) VALUES (?,'content_hidden',NULL,?,?,?)`, target.authorID, target.postID, commentID, now); err != nil {
			return domain.AdminReport{}, fmt.Errorf("insert hidden notification: %w", err)
		}
		if err := insertGovernanceAudit(ctx, tx, input, target, "content_hidden", "visible", "hidden", now); err != nil {
			return domain.AdminReport{}, err
		}
	default:
		return domain.AdminReport{}, &domain.ValidationError{Field: "decision", Reason: "必须是 ignore 或 hide"}
	}
	if err := tx.Commit(); err != nil {
		return domain.AdminReport{}, fmt.Errorf("commit report decision: %w", err)
	}
	return store.findAdminReportByID(ctx, store.db, input.ReportID)
}

func (store *Store) RestoreContent(ctx context.Context, input domain.RestoreContentParams) (domain.RestoredContent, error) {
	var lastErr error
	for attempt := 0; attempt < governanceTransactionAttempts; attempt++ {
		result, err := store.restoreContentOnce(ctx, input)
		if !retryableGovernanceError(err) {
			return result, err
		}
		lastErr = err
	}
	return domain.RestoredContent{}, fmt.Errorf("restore content after retries: %w", lastErr)
}

func (store *Store) restoreContentOnce(ctx context.Context, input domain.RestoreContentParams) (domain.RestoredContent, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.RestoredContent{}, fmt.Errorf("begin restore transaction: %w", err)
	}
	defer tx.Rollback()
	target, err := lockGovernanceTarget(ctx, tx, input.TargetType, input.TargetID)
	if err != nil {
		return domain.RestoredContent{}, err
	}
	if target.deleted {
		return domain.RestoredContent{}, domain.ErrContentNotRestorable
	}
	if target.visibility == domain.VisibilityVisible {
		if target.moderationVersion == input.ExpectedModerationVersion+1 {
			restoredAt, found, err := findLatestRestoreAudit(ctx, tx, input.TargetType, input.TargetID)
			if err != nil {
				return domain.RestoredContent{}, err
			}
			if found {
				if err := tx.Commit(); err != nil {
					return domain.RestoredContent{}, fmt.Errorf("commit restore replay: %w", err)
				}
				return domain.RestoredContent{TargetType: input.TargetType, TargetID: input.TargetID, Visibility: domain.VisibilityVisible, ModerationVersion: target.moderationVersion, RestoredAt: restoredAt}, nil
			}
		}
		if target.moderationVersion > input.ExpectedModerationVersion+1 {
			return domain.RestoredContent{}, domain.ErrModerationVersionConflict
		}
		return domain.RestoredContent{}, domain.ErrContentNotRestorable
	}
	if target.moderationVersion != input.ExpectedModerationVersion {
		return domain.RestoredContent{}, domain.ErrModerationVersionConflict
	}

	now := time.Now().UTC()
	table := "posts"
	if input.TargetType == domain.ContentTypeComment {
		table = "comments"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE `+table+` SET visibility='visible',moderation_version=moderation_version+1,updated_at=? WHERE id=?`, now, input.TargetID); err != nil {
		return domain.RestoredContent{}, fmt.Errorf("restore content: %w", err)
	}
	var commentID any
	if input.TargetType == domain.ContentTypeComment {
		commentID = input.TargetID
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications (user_id,type,actor_user_id,post_id,comment_id,created_at) VALUES (?,'content_restored',NULL,?,?,?)`, target.authorID, target.postID, commentID, now); err != nil {
		return domain.RestoredContent{}, fmt.Errorf("insert restore notification: %w", err)
	}
	audit := domain.DecideReportParams{AdminID: input.AdminID, Note: "", RequestID: input.RequestID}
	if err := insertGovernanceAudit(ctx, tx, audit, target, "content_restored", "hidden", "visible", now); err != nil {
		return domain.RestoredContent{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.RestoredContent{}, fmt.Errorf("commit content restore: %w", err)
	}
	return domain.RestoredContent{TargetType: input.TargetType, TargetID: input.TargetID, Visibility: domain.VisibilityVisible, ModerationVersion: target.moderationVersion + 1, RestoredAt: now}, nil
}

func (store *Store) ListAuditLogs(ctx context.Context, query domain.AuditListQuery) ([]domain.AuditLog, error) {
	afterID := int64(0)
	afterTime := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if query.After != nil {
		afterID, afterTime = query.After.ID, query.After.CreatedAt.UTC()
	}
	dbAction := string(query.Action)
	if query.Action == domain.AuditActionReportIgnored {
		dbAction = "report_dismissed"
	}
	rows, err := store.db.QueryContext(ctx, `
SELECT a.id,u.id,u.display_name,a.action,
CASE WHEN a.post_id IS NOT NULL THEN 'post' ELSE 'comment' END,
COALESCE(a.post_id,a.comment_id),a.report_id,a.reason,a.created_at
FROM admin_audit_logs a JOIN users u ON u.id=a.admin_user_id
WHERE (?='' OR a.action=?)
AND (?='' OR (?='post' AND a.post_id IS NOT NULL) OR (?='comment' AND a.comment_id IS NOT NULL))
AND (?=0 OR a.admin_user_id=?)
AND (?=0 OR a.created_at<? OR (a.created_at=? AND a.id<?))
ORDER BY a.created_at DESC,a.id DESC LIMIT ?`, dbAction, dbAction, query.TargetType, query.TargetType, query.TargetType,
		query.AdminID, query.AdminID, afterID, afterTime, afterTime, afterID, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AuditLog, 0)
	for rows.Next() {
		var item domain.AuditLog
		var action, targetType, reason string
		var reportID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Admin.ID, &item.Admin.DisplayName, &action, &targetType, &item.TargetID, &reportID, &reason, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		item.Action = externalAuditAction(action)
		item.TargetType = domain.ContentType(targetType)
		if reportID.Valid {
			item.ReportID = &reportID.Int64
		}
		if reason != "" {
			item.Note = &reason
		}
		item.CreatedAt = item.CreatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}
	return items, nil
}

func (store *Store) reportTarget(ctx context.Context, reportID int64) (domain.ContentType, int64, error) {
	var postID, commentID sql.NullInt64
	if err := store.db.QueryRowContext(ctx, "SELECT post_id,comment_id FROM reports WHERE id=?", reportID).Scan(&postID, &commentID); errors.Is(err, sql.ErrNoRows) {
		return "", 0, domain.ErrReportNotFound
	} else if err != nil {
		return "", 0, fmt.Errorf("find report target: %w", err)
	}
	if postID.Valid {
		return domain.ContentTypePost, postID.Int64, nil
	}
	return domain.ContentTypeComment, commentID.Int64, nil
}

func lockGovernanceTarget(ctx context.Context, tx *sql.Tx, targetType domain.ContentType, targetID int64) (governanceTarget, error) {
	target := governanceTarget{targetType: targetType, targetID: targetID, postID: targetID}
	var deleted sql.NullTime
	statement := "SELECT author_id,visibility,moderation_version,deleted_at FROM posts WHERE id=? FOR UPDATE"
	notFound := domain.ErrPostNotFound
	if targetType == domain.ContentTypeComment {
		statement = "SELECT author_id,post_id,visibility,moderation_version,deleted_at FROM comments WHERE id=? FOR UPDATE"
		notFound = domain.ErrCommentNotFound
	}
	var err error
	if targetType == domain.ContentTypePost {
		err = tx.QueryRowContext(ctx, statement, targetID).Scan(&target.authorID, &target.visibility, &target.moderationVersion, &deleted)
	} else {
		err = tx.QueryRowContext(ctx, statement, targetID).Scan(&target.authorID, &target.postID, &target.visibility, &target.moderationVersion, &deleted)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return governanceTarget{}, notFound
	}
	if err != nil {
		return governanceTarget{}, fmt.Errorf("lock governance target: %w", err)
	}
	target.deleted = deleted.Valid
	return target, nil
}

func lockTargetReports(ctx context.Context, tx *sql.Tx, targetType domain.ContentType, targetID int64) (map[int64][2]string, error) {
	column := "post_id"
	if targetType == domain.ContentTypeComment {
		column = "comment_id"
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,status,COALESCE(resolution_action,'') FROM reports WHERE `+column+`=? ORDER BY id FOR UPDATE`, targetID)
	if err != nil {
		return nil, fmt.Errorf("lock target reports: %w", err)
	}
	defer rows.Close()
	result := make(map[int64][2]string)
	for rows.Next() {
		var id int64
		var status, action string
		if err := rows.Scan(&id, &status, &action); err != nil {
			return nil, fmt.Errorf("scan locked report: %w", err)
		}
		result[id] = [2]string{status, action}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locked reports: %w", err)
	}
	return result, nil
}

func insertGovernanceAudit(ctx context.Context, tx *sql.Tx, input domain.DecideReportParams, target governanceTarget, action, before, after string, createdAt time.Time) error {
	var reportID any
	if input.ReportID > 0 {
		reportID = input.ReportID
	}
	var postID, commentID any
	if target.targetType == domain.ContentTypePost {
		postID = target.targetID
	} else {
		commentID = target.targetID
	}
	// 审计和状态变化共用事务；审计失败必须让治理事实一起回滚。
	_, err := tx.ExecContext(ctx, `INSERT INTO admin_audit_logs (admin_user_id,action,report_id,post_id,comment_id,before_status,after_status,reason,request_id,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, input.AdminID, action, reportID, postID, commentID, before, after, input.Note, input.RequestID, createdAt)
	if err != nil {
		return fmt.Errorf("insert governance audit: %w", err)
	}
	return nil
}

func findLatestRestoreAudit(ctx context.Context, queryer postQueryer, targetType domain.ContentType, targetID int64) (time.Time, bool, error) {
	column := "post_id"
	if targetType == domain.ContentTypeComment {
		column = "comment_id"
	}
	var createdAt time.Time
	err := queryer.QueryRowContext(ctx, `SELECT created_at FROM admin_audit_logs WHERE action='content_restored' AND `+column+`=? ORDER BY id DESC LIMIT 1`, targetID).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("find restore audit: %w", err)
	}
	return createdAt.UTC(), true, nil
}

func retryableGovernanceError(err error) bool {
	var mysqlErr *driver.MySQLError
	return errors.As(err, &mysqlErr) && (mysqlErr.Number == 1213 || mysqlErr.Number == 1205)
}

func externalAuditAction(value string) domain.AuditAction {
	if value == "report_dismissed" {
		return domain.AuditActionReportIgnored
	}
	return domain.AuditAction(value)
}
