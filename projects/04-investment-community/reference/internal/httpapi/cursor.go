package httpapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

const (
	cursorVersion    = 1
	maximumCursorLen = 2048
)

var errInvalidCursor = errors.New("invalid cursor")

type securityCursorBinding struct {
	Query    string
	Exchange string
	Limit    int
}

type circleCursorBinding struct {
	Limit int
}

type postCursorBinding struct {
	CircleID   int64
	SecurityID int64
	Limit      int
}

type commentCursorBinding struct {
	PostID int64
	Limit  int
}

type notificationCursorBinding struct {
	UserID     int64
	UnreadOnly bool
	Limit      int
}

type cursorPayload struct {
	Version    int    `json:"v"`
	Kind       string `json:"k"`
	Query      string `json:"q,omitempty"`
	Exchange   string `json:"e,omitempty"`
	Limit      int    `json:"l"`
	Code       string `json:"c,omitempty"`
	CreatedAt  int64  `json:"t,omitempty"`
	CircleID   int64  `json:"o,omitempty"`
	SecurityID int64  `json:"s,omitempty"`
	PostID     int64  `json:"p,omitempty"`
	UserID     int64  `json:"u,omitempty"`
	UnreadOnly bool   `json:"n,omitempty"`
	ID         int64  `json:"i"`
	ExpiresAt  int64  `json:"x"`
}

// CursorCodec 只在 HTTP 边界把稳定位置变成不透明令牌；仓储无需知道签名和筛选绑定。
type CursorCodec struct {
	key []byte
	ttl time.Duration
	now func() time.Time
}

func NewCursorCodec(secret string, ttl time.Duration) (*CursorCodec, error) {
	if len(secret) < 32 {
		return nil, errors.New("cursor codec secret must contain at least 32 bytes")
	}
	if ttl < time.Second {
		return nil, errors.New("cursor codec ttl must be at least one second")
	}
	// 使用域分离后的派生键，避免游标签名与 JWT 直接共享同一密码学用途。
	derived := sha256.Sum256(append([]byte("investment-community/cursor/v1\x00"), []byte(secret)...))
	return &CursorCodec{key: derived[:], ttl: ttl, now: time.Now}, nil
}

func (codec *CursorCodec) EncodeSecurity(binding securityCursorBinding, position domain.SecurityCursor) (string, error) {
	if position.ID <= 0 || position.Code == "" || binding.Limit < 1 {
		return "", errInvalidCursor
	}
	return codec.encode(cursorPayload{
		Version: cursorVersion, Kind: "securities", Query: binding.Query, Exchange: binding.Exchange,
		Limit: binding.Limit, Code: position.Code, ID: position.ID,
	})
}

func (codec *CursorCodec) DecodeSecurity(token string, binding securityCursorBinding) (domain.SecurityCursor, error) {
	payload, err := codec.decode(token)
	if err != nil || payload.Kind != "securities" || payload.Query != binding.Query ||
		payload.Exchange != binding.Exchange || payload.Limit != binding.Limit || payload.Code == "" || payload.ID <= 0 {
		return domain.SecurityCursor{}, errInvalidCursor
	}
	return domain.SecurityCursor{Code: payload.Code, ID: payload.ID}, nil
}

func (codec *CursorCodec) EncodeCircle(binding circleCursorBinding, position domain.CircleCursor) (string, error) {
	if position.ID <= 0 || position.CreatedAt.IsZero() || binding.Limit < 1 {
		return "", errInvalidCursor
	}
	return codec.encode(cursorPayload{
		Version: cursorVersion, Kind: "circles", Limit: binding.Limit,
		CreatedAt: position.CreatedAt.UTC().UnixMicro(), ID: position.ID,
	})
}

func (codec *CursorCodec) DecodeCircle(token string, binding circleCursorBinding) (domain.CircleCursor, error) {
	payload, err := codec.decode(token)
	if err != nil || payload.Kind != "circles" || payload.Limit != binding.Limit || payload.CreatedAt <= 0 || payload.ID <= 0 {
		return domain.CircleCursor{}, errInvalidCursor
	}
	return domain.CircleCursor{CreatedAt: time.UnixMicro(payload.CreatedAt).UTC(), ID: payload.ID}, nil
}

func (codec *CursorCodec) EncodePost(binding postCursorBinding, position domain.PostCursor) (string, error) {
	if position.ID <= 0 || position.CreatedAt.IsZero() || binding.CircleID < 0 || binding.SecurityID < 0 || binding.Limit < 1 {
		return "", errInvalidCursor
	}
	return codec.encode(cursorPayload{Version: cursorVersion, Kind: "posts", CircleID: binding.CircleID,
		SecurityID: binding.SecurityID, Limit: binding.Limit, CreatedAt: position.CreatedAt.UTC().UnixMicro(), ID: position.ID})
}

func (codec *CursorCodec) DecodePost(token string, binding postCursorBinding) (domain.PostCursor, error) {
	payload, err := codec.decode(token)
	if err != nil || payload.Kind != "posts" || payload.CircleID != binding.CircleID || payload.SecurityID != binding.SecurityID ||
		payload.Limit != binding.Limit || payload.CreatedAt <= 0 || payload.ID <= 0 {
		return domain.PostCursor{}, errInvalidCursor
	}
	return domain.PostCursor{CreatedAt: time.UnixMicro(payload.CreatedAt).UTC(), ID: payload.ID}, nil
}

func (codec *CursorCodec) EncodeComment(binding commentCursorBinding, position domain.CommentCursor) (string, error) {
	if binding.PostID <= 0 || binding.Limit < 1 || position.ID <= 0 || position.CreatedAt.IsZero() {
		return "", errInvalidCursor
	}
	return codec.encode(cursorPayload{Version: cursorVersion, Kind: "comments", PostID: binding.PostID,
		Limit: binding.Limit, CreatedAt: position.CreatedAt.UTC().UnixMicro(), ID: position.ID})
}

func (codec *CursorCodec) DecodeComment(token string, binding commentCursorBinding) (domain.CommentCursor, error) {
	payload, err := codec.decode(token)
	if err != nil || payload.Kind != "comments" || payload.PostID != binding.PostID || payload.Limit != binding.Limit ||
		payload.CreatedAt <= 0 || payload.ID <= 0 {
		return domain.CommentCursor{}, errInvalidCursor
	}
	return domain.CommentCursor{CreatedAt: time.UnixMicro(payload.CreatedAt).UTC(), ID: payload.ID}, nil
}

func (codec *CursorCodec) EncodeNotification(binding notificationCursorBinding, position domain.NotificationCursor) (string, error) {
	if binding.UserID <= 0 || binding.Limit < 1 || position.ID <= 0 || position.CreatedAt.IsZero() {
		return "", errInvalidCursor
	}
	return codec.encode(cursorPayload{Version: cursorVersion, Kind: "notifications", UserID: binding.UserID,
		UnreadOnly: binding.UnreadOnly, Limit: binding.Limit, CreatedAt: position.CreatedAt.UTC().UnixMicro(), ID: position.ID})
}

func (codec *CursorCodec) DecodeNotification(token string, binding notificationCursorBinding) (domain.NotificationCursor, error) {
	payload, err := codec.decode(token)
	if err != nil || payload.Kind != "notifications" || payload.UserID != binding.UserID ||
		payload.UnreadOnly != binding.UnreadOnly || payload.Limit != binding.Limit || payload.CreatedAt <= 0 || payload.ID <= 0 {
		return domain.NotificationCursor{}, errInvalidCursor
	}
	return domain.NotificationCursor{CreatedAt: time.UnixMicro(payload.CreatedAt).UTC(), ID: payload.ID}, nil
}

func (codec *CursorCodec) encode(payload cursorPayload) (string, error) {
	if codec == nil || len(codec.key) == 0 || codec.now == nil {
		return "", errors.New("cursor codec is not configured")
	}
	payload.ExpiresAt = codec.now().Add(codec.ttl).Unix()
	contents, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode cursor payload: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(contents)
	signature := codec.sign(encoded)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (codec *CursorCodec) decode(token string) (cursorPayload, error) {
	if codec == nil || len(codec.key) == 0 || codec.now == nil || len(token) < 1 || len(token) > maximumCursorLen {
		return cursorPayload{}, errInvalidCursor
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return cursorPayload{}, errInvalidCursor
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, codec.sign(parts[0])) {
		return cursorPayload{}, errInvalidCursor
	}
	contents, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return cursorPayload{}, errInvalidCursor
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var payload cursorPayload
	if err := decoder.Decode(&payload); err != nil {
		return cursorPayload{}, errInvalidCursor
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return cursorPayload{}, errInvalidCursor
	}
	if payload.Version != cursorVersion || payload.ExpiresAt <= codec.now().Unix() {
		return cursorPayload{}, errInvalidCursor
	}
	return payload, nil
}

func (codec *CursorCodec) sign(encodedPayload string) []byte {
	mac := hmac.New(sha256.New, codec.key)
	_, _ = mac.Write([]byte(encodedPayload))
	return mac.Sum(nil)
}
