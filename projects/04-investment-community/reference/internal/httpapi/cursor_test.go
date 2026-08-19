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
