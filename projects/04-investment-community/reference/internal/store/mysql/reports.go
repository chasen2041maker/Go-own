package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func (store *Store) CreateReport(ctx context.Context, input domain.CreateReportParams) (domain.ReportReceipt, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ReportReceipt{}, false, fmt.Errorf("begin create report transaction: %w", err)
	}
	defer tx.Rollback()

	// 查重必须早于目标可见性检查。目标稍后被隐藏时，网络重试仍应返回原举报而不是 404。
	receipt, found, err := findExistingReport(ctx, tx, input.ReporterID, input.TargetType, input.TargetID, true)
	if err != nil {
		return domain.ReportReceipt{}, false, err
	}
	if found {
		if err := tx.Commit(); err != nil {
			return domain.ReportReceipt{}, false, fmt.Errorf("commit report replay: %w", err)
		}
		return receipt, true, nil
	}

	authorID, err := lockReportTarget(ctx, tx, input.TargetType, input.TargetID)
	if err != nil {
		return domain.ReportReceipt{}, false, err
	}
	if authorID == input.ReporterID {
		return domain.ReportReceipt{}, false, domain.ErrSelfReportForbidden
	}
	var result sql.Result
	if input.TargetType == domain.ContentTypePost {
		result, err = tx.ExecContext(ctx, `INSERT INTO reports (reporter_id,post_id,reason_code,details) VALUES (?,?,?,?)`, input.ReporterID, input.TargetID, input.Reason, input.Details)
	} else {
		result, err = tx.ExecContext(ctx, `INSERT INTO reports (reporter_id,comment_id,reason_code,details) VALUES (?,?,?,?)`, input.ReporterID, input.TargetID, input.Reason, input.Details)
	}
	if err != nil {
		if duplicateKey(err) {
			replayed, ok, findErr := findExistingReport(ctx, store.db, input.ReporterID, input.TargetType, input.TargetID, false)
			if findErr != nil || !ok {
				return domain.ReportReceipt{}, false, fmt.Errorf("replay concurrent report: %w", findErr)
			}
			return replayed, true, nil
		}
		return domain.ReportReceipt{}, false, fmt.Errorf("insert report: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.ReportReceipt{}, false, fmt.Errorf("report id: %w", err)
	}
	receipt, _, err = findReportReceiptByID(ctx, tx, id)
	if err != nil {
		return domain.ReportReceipt{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.ReportReceipt{}, false, fmt.Errorf("commit create report: %w", err)
	}
	return receipt, false, nil
}

func findExistingReport(ctx context.Context, queryer postQueryer, reporterID int64, targetType domain.ContentType, targetID int64, lock bool) (domain.ReportReceipt, bool, error) {
	column := "post_id"
	if targetType == domain.ContentTypeComment {
		column = "comment_id"
	}
	statement := `SELECT id,status,created_at FROM reports WHERE reporter_id=? AND ` + column + `=?`
	if lock {
		statement += " FOR UPDATE"
	}
	var receipt domain.ReportReceipt
	var status string
	err := queryer.QueryRowContext(ctx, statement, reporterID, targetID).Scan(&receipt.ID, &status, &receipt.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ReportReceipt{}, false, nil
	}
	if err != nil {
		return domain.ReportReceipt{}, false, fmt.Errorf("find existing report: %w", err)
	}
	receipt.TargetType = targetType
	receipt.TargetID = targetID
	receipt.Status = externalReportStatus(status)
	receipt.CreatedAt = receipt.CreatedAt.UTC()
	return receipt, true, nil
}

func findReportReceiptByID(ctx context.Context, queryer postQueryer, id int64) (domain.ReportReceipt, bool, error) {
	var receipt domain.ReportReceipt
	var postID, commentID sql.NullInt64
	var status string
	err := queryer.QueryRowContext(ctx, "SELECT id,post_id,comment_id,status,created_at FROM reports WHERE id=?", id).Scan(&receipt.ID, &postID, &commentID, &status, &receipt.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ReportReceipt{}, false, nil
	}
	if err != nil {
		return domain.ReportReceipt{}, false, fmt.Errorf("find report receipt: %w", err)
	}
	if postID.Valid {
		receipt.TargetType = domain.ContentTypePost
		receipt.TargetID = postID.Int64
	} else {
		receipt.TargetType = domain.ContentTypeComment
		receipt.TargetID = commentID.Int64
	}
	receipt.Status = externalReportStatus(status)
	receipt.CreatedAt = receipt.CreatedAt.UTC()
	return receipt, true, nil
}

func lockReportTarget(ctx context.Context, tx *sql.Tx, targetType domain.ContentType, targetID int64) (int64, error) {
	var authorID int64
	var visibility string
	var deleted sql.NullTime
	statement := "SELECT author_id,visibility,deleted_at FROM posts WHERE id=? FOR UPDATE"
	notFound := domain.ErrPostNotFound
	if targetType == domain.ContentTypeComment {
		statement = "SELECT author_id,visibility,deleted_at FROM comments WHERE id=? FOR UPDATE"
		notFound = domain.ErrCommentNotFound
	}
	err := tx.QueryRowContext(ctx, statement, targetID).Scan(&authorID, &visibility, &deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, notFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock report target: %w", err)
	}
	if deleted.Valid || visibility != "visible" {
		return 0, notFound
	}
	return authorID, nil
}

func (store *Store) ListReports(ctx context.Context, query domain.ReportListQuery) ([]domain.AdminReport, error) {
	afterID := int64(0)
	afterTime := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if query.After != nil {
		afterID = query.After.ID
		afterTime = query.After.CreatedAt.UTC()
	}
	dbStatus := ""
	if query.Status == domain.ReportStatusIgnored {
		dbStatus = "dismissed"
	} else if query.Status != "" {
		dbStatus = string(query.Status)
	}
	rows, err := store.db.QueryContext(ctx, `
SELECT r.id,ru.id,ru.display_name,
CASE WHEN r.post_id IS NOT NULL THEN 'post' ELSE 'comment' END,
COALESCE(r.post_id,r.comment_id),COALESCE(p.visibility,c.visibility),COALESCE(p.moderation_version,c.moderation_version),
CASE WHEN p.deleted_at IS NOT NULL OR c.deleted_at IS NOT NULL THEN TRUE ELSE FALSE END,
p.title,CASE WHEN p.id IS NOT NULL THEN IF(p.deleted_at IS NULL,LEFT(p.body,500),NULL) ELSE IF(c.deleted_at IS NULL,LEFT(c.body,500),NULL) END,
r.reason_code,r.details,r.status,r.resolution_action,hu.id,hu.display_name,r.created_at,r.handled_at
FROM reports r JOIN users ru ON ru.id=r.reporter_id
LEFT JOIN posts p ON p.id=r.post_id LEFT JOIN comments c ON c.id=r.comment_id LEFT JOIN users hu ON hu.id=r.handled_by
WHERE (?='' OR r.status=?) AND (?='' OR (?='post' AND r.post_id IS NOT NULL) OR (?='comment' AND r.comment_id IS NOT NULL))
AND (?=0 OR r.created_at<? OR (r.created_at=? AND r.id<?))
ORDER BY r.created_at DESC,r.id DESC LIMIT ?`, dbStatus, dbStatus, query.TargetType, query.TargetType, query.TargetType, afterID, afterTime, afterTime, afterID, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("query admin reports: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AdminReport, 0)
	for rows.Next() {
		var item domain.AdminReport
		var targetType, visibility, reason, status string
		var title, excerpt sql.NullString
		var action sql.NullString
		var handlerID sql.NullInt64
		var handlerName sql.NullString
		var handledAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Reporter.ID, &item.Reporter.DisplayName, &targetType, &item.Target.ID, &visibility, &item.Target.ModerationVersion, &item.Target.Deleted, &title, &excerpt, &reason, &item.Details, &status, &action, &handlerID, &handlerName, &item.CreatedAt, &handledAt); err != nil {
			return nil, fmt.Errorf("scan admin report: %w", err)
		}
		item.Target.TargetType = domain.ContentType(targetType)
		item.Target.Visibility = domain.Visibility(visibility)
		if title.Valid {
			item.Target.Title = &title.String
		}
		if excerpt.Valid {
			item.Target.Excerpt = &excerpt.String
		}
		item.Reason = domain.ReportReason(reason)
		item.Status = externalReportStatus(status)
		if action.Valid {
			value := externalReportDecision(action.String)
			item.Decision = &value
		}
		if handlerID.Valid {
			item.DecidedBy = &domain.PublicUser{ID: handlerID.Int64, DisplayName: handlerName.String}
		}
		if handledAt.Valid {
			value := handledAt.Time.UTC()
			item.DecidedAt = &value
		}
		item.CreatedAt = item.CreatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin reports: %w", err)
	}
	return items, nil
}

func externalReportStatus(value string) domain.ReportStatus {
	if value == "dismissed" {
		return domain.ReportStatusIgnored
	}
	return domain.ReportStatus(value)
}
func externalReportDecision(value string) domain.ReportDecision {
	if value == "dismiss" {
		return domain.ReportDecisionIgnore
	}
	return domain.ReportDecision(value)
}
