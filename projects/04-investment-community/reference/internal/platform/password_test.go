package platform

import (
	"strings"
	"testing"
)

func TestPasswordHasherHashesAndVerifiesWithoutStoringPlaintext(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "correct-horse"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == password || strings.Contains(hash, password) {
		t.Fatalf("Hash() exposed plaintext: %q", hash)
	}
	if err := hasher.Verify(hash, password); err != nil {
		t.Fatalf("Verify(correct password) error = %v", err)
	}
	if err := hasher.Verify(hash, "wrong-password"); err == nil {
		t.Fatal("Verify(wrong password) error = nil")
	}
}

func TestPasswordHasherRejectsLengthsOutsideBcryptByteRange(t *testing.T) {
	hasher := NewPasswordHasher()
	for _, password := range []string{strings.Repeat("p", 7), strings.Repeat("p", 73)} {
		if _, err := hasher.Hash(password); err == nil {
			t.Fatalf("Hash(%d bytes) error = nil", len(password))
		}
	}
}
