package httpapi

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func TestCursorCodecRoundTripsStablePositions(t *testing.T) {
	now := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	codec := mustCursorCodec(t, now)

	securityBinding := securityCursorBinding{Query: "NO", Exchange: "XSEA", Limit: 2}
	securityToken, err := codec.EncodeSecurity(securityBinding, domain.SecurityCursor{Code: "NOVA", ID: 7})
	if err != nil {
		t.Fatalf("EncodeSecurity() error = %v", err)
	}
	securityPosition, err := codec.DecodeSecurity(securityToken, securityBinding)
	if err != nil || securityPosition.Code != "NOVA" || securityPosition.ID != 7 {
		t.Fatalf("DecodeSecurity() = %#v, %v", securityPosition, err)
	}

	createdAt := now.Add(-time.Microsecond)
	circleBinding := circleCursorBinding{Limit: 20}
	circleToken, err := codec.EncodeCircle(circleBinding, domain.CircleCursor{CreatedAt: createdAt, ID: 9})
	if err != nil {
		t.Fatalf("EncodeCircle() error = %v", err)
	}
	circlePosition, err := codec.DecodeCircle(circleToken, circleBinding)
	if err != nil || !circlePosition.CreatedAt.Equal(createdAt) || circlePosition.ID != 9 {
		t.Fatalf("DecodeCircle() = %#v, %v", circlePosition, err)
	}

	postBinding := postCursorBinding{CircleID: 7, SecurityID: 3, Limit: 20}
	postToken, err := codec.EncodePost(postBinding, domain.PostCursor{CreatedAt: createdAt, ID: 11})
	if err != nil {
		t.Fatalf("EncodePost() error = %v", err)
	}
	postPosition, err := codec.DecodePost(postToken, postBinding)
	if err != nil || !postPosition.CreatedAt.Equal(createdAt) || postPosition.ID != 11 {
		t.Fatalf("DecodePost() = %#v, %v", postPosition, err)
	}
	if _, err := codec.DecodePost(postToken, postCursorBinding{CircleID: 8, SecurityID: 3, Limit: 20}); !errors.Is(err, errInvalidCursor) {
		t.Fatalf("DecodePost(changed filter) error = %v", err)
	}

	commentBinding := commentCursorBinding{PostID: 11, Limit: 20}
	commentToken, err := codec.EncodeComment(commentBinding, domain.CommentCursor{CreatedAt: createdAt, ID: 15})
	if err != nil {
		t.Fatalf("EncodeComment() error = %v", err)
	}
	commentPosition, err := codec.DecodeComment(commentToken, commentBinding)
	if err != nil || commentPosition.ID != 15 || !commentPosition.CreatedAt.Equal(createdAt) {
		t.Fatalf("DecodeComment() = %#v, %v", commentPosition, err)
	}
	if _, err := codec.DecodeComment(commentToken, commentCursorBinding{PostID: 12, Limit: 20}); !errors.Is(err, errInvalidCursor) {
		t.Fatalf("DecodeComment(changed post) error = %v", err)
	}

	notificationBinding := notificationCursorBinding{UserID: 42, UnreadOnly: true, Limit: 10}
	notificationToken, err := codec.EncodeNotification(notificationBinding, domain.NotificationCursor{CreatedAt: createdAt, ID: 20})
	if err != nil {
		t.Fatalf("EncodeNotification() error = %v", err)
	}
	notificationPosition, err := codec.DecodeNotification(notificationToken, notificationBinding)
	if err != nil || notificationPosition.ID != 20 || !notificationPosition.CreatedAt.Equal(createdAt) {
		t.Fatalf("DecodeNotification() = %#v, %v", notificationPosition, err)
	}
	if _, err := codec.DecodeNotification(notificationToken, notificationCursorBinding{UserID: 43, UnreadOnly: true, Limit: 10}); !errors.Is(err, errInvalidCursor) {
		t.Fatalf("DecodeNotification(changed user) error = %v", err)
	}
}

func TestCursorCodecRejectsTamperingChangedFiltersWrongKindAndExpiry(t *testing.T) {
	now := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	codec := mustCursorCodec(t, now)
	binding := securityCursorBinding{Query: "NO", Exchange: "XSEA", Limit: 2}
	token, err := codec.EncodeSecurity(binding, domain.SecurityCursor{Code: "NOVA", ID: 7})
	if err != nil {
		t.Fatalf("EncodeSecurity() error = %v", err)
	}

	parts := strings.Split(token, ".")
	tampered := parts[0][:len(parts[0])-1] + string(alternateBase64Byte(parts[0][len(parts[0])-1])) + "." + parts[1]
	tests := []struct {
		name    string
		token   string
		binding securityCursorBinding
	}{
		{name: "tampered", token: tampered, binding: binding},
		{name: "changed query", token: token, binding: securityCursorBinding{Query: "TI", Exchange: "XSEA", Limit: 2}},
		{name: "changed exchange", token: token, binding: securityCursorBinding{Query: "NO", Exchange: "XNOVA", Limit: 2}},
		{name: "changed limit", token: token, binding: securityCursorBinding{Query: "NO", Exchange: "XSEA", Limit: 3}},
		{name: "empty", token: "", binding: binding},
		{name: "oversized", token: strings.Repeat("a", 2049), binding: binding},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := codec.DecodeSecurity(test.token, test.binding); !errors.Is(err, errInvalidCursor) {
				t.Fatalf("DecodeSecurity() error = %v, want errInvalidCursor", err)
			}
		})
	}

	if _, err := codec.DecodeCircle(token, circleCursorBinding{Limit: 2}); !errors.Is(err, errInvalidCursor) {
		t.Fatalf("DecodeCircle(security token) error = %v, want errInvalidCursor", err)
	}
	codec.now = func() time.Time { return now.Add(31 * time.Minute) }
	if _, err := codec.DecodeSecurity(token, binding); !errors.Is(err, errInvalidCursor) {
		t.Fatalf("DecodeSecurity(expired) error = %v, want errInvalidCursor", err)
	}
}

func mustCursorCodec(t *testing.T, now time.Time) *CursorCodec {
	t.Helper()
	codec, err := NewCursorCodec(strings.Repeat("s", 32), 30*time.Minute)
	if err != nil {
		t.Fatalf("NewCursorCodec() error = %v", err)
	}
	codec.now = func() time.Time { return now }
	return codec
}

func alternateBase64Byte(value byte) byte {
	if value == 'A' {
		return 'B'
	}
	return 'A'
}
