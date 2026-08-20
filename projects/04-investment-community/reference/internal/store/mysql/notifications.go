package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func (store *Store) ListNotifications(ctx context.Context, query domain.NotificationListQuery) ([]domain.Notification, error) {
	afterID := int64(0)
	afterTime := time.Now().UTC().Add(100 * 365 * 24 * time.Hour)
	if query.After != nil {
		afterID = query.After.ID
		afterTime = query.After.CreatedAt.UTC()
	}
	rows, err := store.db.QueryContext(ctx, `
SELECT n.id,n.type,a.id,a.display_name,n.post_id,n.comment_id,n.read_at,n.created_at
FROM notifications n LEFT JOIN users a ON a.id=n.actor_user_id
WHERE n.user_id=? AND (?=FALSE OR n.read_at IS NULL)
  AND (?=0 OR n.created_at<? OR (n.created_at=? AND n.id<?))
ORDER BY n.created_at DESC,n.id DESC LIMIT ?`,
		query.UserID, query.UnreadOnly, afterID, afterTime, afterTime, afterID, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()
	notifications := make([]domain.Notification, 0)
	for rows.Next() {
		var notification domain.Notification
		var notificationType string
		var actorID sql.NullInt64
		var actorName sql.NullString
		var commentID sql.NullInt64
		var readAt sql.NullTime
		if err := rows.Scan(&notification.ID, &notificationType, &actorID, &actorName, &notification.PostID,
			&commentID, &readAt, &notification.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		notification.Type = domain.NotificationType(notificationType)
		if actorID.Valid {
			notification.Actor = &domain.PublicUser{ID: actorID.Int64, DisplayName: actorName.String}
		}
		if commentID.Valid {
			notification.CommentID = &commentID.Int64
		}
		if readAt.Valid {
			value := readAt.Time.UTC()
			notification.ReadAt = &value
		}
		notification.CreatedAt = notification.CreatedAt.UTC()
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}
	return notifications, nil
}

func (store *Store) MarkAllNotificationsRead(ctx context.Context, userID int64) (domain.NotificationReadResult, error) {
	readAt := time.Now().UTC().Truncate(time.Microsecond)
	result, err := store.db.ExecContext(ctx,
		"UPDATE notifications SET read_at=? WHERE user_id=? AND read_at IS NULL", readAt, userID)
	if err != nil {
		return domain.NotificationReadResult{}, fmt.Errorf("mark notifications read: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return domain.NotificationReadResult{}, fmt.Errorf("read marked notification count: %w", err)
	}
	return domain.NotificationReadResult{ReadCount: count, ReadAt: readAt}, nil
}
