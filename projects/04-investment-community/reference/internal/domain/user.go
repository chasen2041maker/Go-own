package domain

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// User 中的 PasswordHash 只供认证用例使用，对外响应必须映射为单独 DTO。
type User struct {
	ID           int64
	Email        string
	PasswordHash string `json:"-"`
	DisplayName  string
	Role         Role
	Status       UserStatus
}

// NormalizeEmail 先按契约规范化，再要求 net/mail 只能解析出一个无显示名的裸地址。
func NormalizeEmail(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" || len(normalized) > 254 {
		return "", &ValidationError{Field: "email", Reason: "必须是长度不超过 254 字节的单个邮箱地址"}
	}
	address, err := mail.ParseAddress(normalized)
	if err != nil || address.Name != "" || address.Address != normalized {
		return "", &ValidationError{Field: "email", Reason: "必须是单个裸邮箱地址"}
	}
	return normalized, nil
}

// NormalizeDisplayName 以 Unicode 字符而不是字节计数，中文名字不会被误判为过长。
func NormalizeDisplayName(raw string) (string, error) {
	normalized := strings.TrimSpace(raw)
	if !utf8.ValidString(normalized) {
		return "", &ValidationError{Field: "display_name", Reason: "必须是有效的 UTF-8 文本"}
	}
	length := utf8.RuneCountInString(normalized)
	if length < 1 || length > 80 {
		return "", &ValidationError{Field: "display_name", Reason: "长度必须为 1 到 80 个字符"}
	}
	return normalized, nil
}

// ValidatePassword 用字符数表达最小强度，同时用字节数遵守 bcrypt 的 72 字节硬上限。
func ValidatePassword(password string) error {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 12 || len([]byte(password)) > 72 {
		return &ValidationError{Field: "password", Reason: "至少 12 个字符且 UTF-8 编码不超过 72 字节"}
	}
	return nil
}

// ValidateLoginPassword 只校验登录协议允许的字符数，不重新套用注册强度策略。
// 旧密码是否正确仍由 bcrypt 判断，但异常长度会在固定 dummy 工作后返回字段错误。
func ValidateLoginPassword(password string) error {
	if !utf8.ValidString(password) {
		return &ValidationError{Field: "password", Reason: "必须是有效的 UTF-8 文本"}
	}
	if length := utf8.RuneCountInString(password); length < 1 || length > 128 {
		return &ValidationError{Field: "password", Reason: "长度必须为 1 到 128 个字符"}
	}
	return nil
}

// WithoutPasswordHash 返回可跨越认证用例边界的用户快照。
// 仓储实体保留哈希用于密码校验，但 Handler、请求上下文和日志都不应再接触该字段。
func (user User) WithoutPasswordHash() User {
	user.PasswordHash = ""
	return user
}
