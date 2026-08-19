package domain

import (
	"strings"
	"testing"
)

func TestValidatePasswordUsesRuneMinimumAndBcryptByteMaximum(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "twelve ASCII characters", password: "password1234"},
		{name: "twelve multibyte characters", password: strings.Repeat("密", 12)},
		{name: "four multibyte characters are not twelve characters", password: "密码安全", wantErr: true},
		{name: "more than bcrypt seventy two bytes", password: strings.Repeat("密", 25), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidatePassword(test.password)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidatePassword() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestValidateLoginPasswordUsesContractCharacterLimit(t *testing.T) {
	for _, test := range []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "one character", password: "x"},
		{name: "empty", password: "", wantErr: true},
		{name: "one hundred twenty eight characters", password: strings.Repeat("密", 128)},
		{name: "one hundred twenty nine characters", password: strings.Repeat("x", 129), wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateLoginPassword(test.password)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateLoginPassword() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
