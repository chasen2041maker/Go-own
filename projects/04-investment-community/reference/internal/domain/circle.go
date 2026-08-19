package domain

import (
	"errors"
	"time"
)

var ErrCircleNotFound = errors.New("circle not found")

type CircleStatus string

const (
	CircleStatusActive   CircleStatus = "active"
	CircleStatusArchived CircleStatus = "archived"
)

// Circle 把当前用户的成员状态和公开统计一起返回，但不暴露内部 status 字段。
type Circle struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	MemberCount int64
	IsMember    bool
	CreatedAt   time.Time
}

type CircleCursor struct {
	CreatedAt time.Time
	ID        int64
}

type CircleListQuery struct {
	UserID int64
	After  *CircleCursor
	Limit  int
}

// CircleMembership 的 JoinedAt 在退出状态必须为 nil，避免伪造一个并不存在的关系时间。
type CircleMembership struct {
	CircleID int64
	UserID   int64
	Joined   bool
	JoinedAt *time.Time
}
