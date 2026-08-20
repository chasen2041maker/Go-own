package domain

import "time"

type NotificationType string

const (
	NotificationTypeComment         NotificationType = "comment"
	NotificationTypeReply           NotificationType = "reply"
	NotificationTypeContentHidden   NotificationType = "content_hidden"
	NotificationTypeContentRestored NotificationType = "content_restored"
)

// Notification.Actor 为 nil 时代表治理系统动作，避免向内容作者泄露管理员身份。
type Notification struct {
	ID        int64
	Type      NotificationType
	Actor     *PublicUser
	PostID    int64
	CommentID *int64
	ReadAt    *time.Time
	CreatedAt time.Time
}

type NotificationCursor struct {
	CreatedAt time.Time
	ID        int64
}

type NotificationListQuery struct {
	UserID     int64
	UnreadOnly bool
	After      *NotificationCursor
	Limit      int
}

type NotificationReadResult struct {
	ReadCount int64
	ReadAt    time.Time
}
