// Package domain 保存不依赖 HTTP、数据库或密码库的业务词汇和规则。
package domain

import (
	"errors"
	"fmt"
)

var (
	ErrEmailTaken         = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrForbidden          = errors.New("forbidden")
)

// ValidationError 只描述可公开的字段规则；底层数据库和密码库错误不能放进这里。
type ValidationError struct {
	Field  string
	Reason string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("validate %s: %s", err.Field, err.Reason)
}
