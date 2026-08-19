package domain

import (
	"errors"
	"time"
)

var (
	ErrPostNotFound        = errors.New("post not found")
	ErrMembershipRequired  = errors.New("circle membership required")
	ErrIdempotencyConflict = errors.New("idempotency key belongs to a different request")
	ErrVersionConflict     = errors.New("post version conflict")
	ErrContentNotEditable  = errors.New("content is not editable")
)

type Visibility string

const (
	VisibilityVisible Visibility = "visible"
	VisibilityHidden  Visibility = "hidden"
)

// PublicUser 是内容接口唯一允许公开的用户投影；有意不承载邮箱、角色或密码哈希。
type PublicUser struct {
	ID          int64
	DisplayName string
}

type PostCircle struct {
	ID   int64
	Slug string
	Name string
}

type Post struct {
	ID                int64
	Circle            PostCircle
	Author            PublicUser
	Title             string
	Body              string
	Securities        []Security
	CommentCount      int64
	Visibility        Visibility
	Version           int64
	ModerationVersion int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type PostCursor struct {
	CreatedAt time.Time
	ID        int64
}

type PostListQuery struct {
	CircleID   int64
	SecurityID int64
	After      *PostCursor
	Limit      int
}

type CreatePostParams struct {
	AuthorID       int64
	CircleID       int64
	Title          string
	Body           string
	SecurityIDs    []int64
	IdempotencyKey string
	RequestHash    string
}

type UpdatePostParams struct {
	ActorID           int64
	PostID            int64
	ExpectedVersion   int64
	Title             *string
	Body              *string
	SecurityIDs       []int64
	ReplaceSecurities bool
}
