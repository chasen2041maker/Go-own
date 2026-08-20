package domain

import (
	"errors"
	"time"
)

var (
	ErrCommentNotFound      = errors.New("comment not found")
	ErrParentCommentInvalid = errors.New("parent comment must be a visible top-level comment in the same post")
)

// Comment 是评论接口的公开投影；作者刻意只使用 PublicUser，防止列表查询带出邮箱或密码哈希。
type Comment struct {
	ID                int64
	PostID            int64
	ParentCommentID   *int64
	Author            PublicUser
	Body              string
	ModerationVersion int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CommentCursor struct {
	CreatedAt time.Time
	ID        int64
}

type CommentListQuery struct {
	PostID int64
	After  *CommentCursor
	Limit  int
}

type CreateCommentParams struct {
	AuthorID        int64
	PostID          int64
	ParentCommentID *int64
	Body            string
	IdempotencyKey  string
	RequestHash     string
}
