package platform

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfigRejectsMissingJWTSecretWithoutLeakingConfigValues(t *testing.T) {
	environment := validEnvironment()
	delete(environment, "JWT_SECRET")

	_, err := LoadConfig(mapLookup(environment))
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want missing JWT_SECRET error")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("LoadConfig() error = %q, want JWT_SECRET variable name", err)
	}
	if strings.Contains(err.Error(), environment["DATABASE_DSN"]) {
		t.Fatalf("LoadConfig() error leaked DATABASE_DSN: %q", err)
	}
}

func TestLoadConfigUsesSafeHTTPDefaults(t *testing.T) {
	config, err := LoadConfig(mapLookup(validEnvironment()))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got, want := config.HTTPAddress, "127.0.0.1:8084"; got != want {
		t.Errorf("HTTPAddress = %q, want %q", got, want)
	}
	if got, want := config.ReadHeaderTimeout, 5*time.Second; got != want {
		t.Errorf("ReadHeaderTimeout = %s, want %s", got, want)
	}
	if got, want := config.ReadTimeout, 10*time.Second; got != want {
		t.Errorf("ReadTimeout = %s, want %s", got, want)
	}
	if got, want := config.WriteTimeout, 15*time.Second; got != want {
		t.Errorf("WriteTimeout = %s, want %s", got, want)
	}
	if got, want := config.IdleTimeout, 60*time.Second; got != want {
		t.Errorf("IdleTimeout = %s, want %s", got, want)
	}
	if got, want := config.ShutdownTimeout, 10*time.Second; got != want {
		t.Errorf("ShutdownTimeout = %s, want %s", got, want)
	}
	if got, want := config.ReadinessTimeout, 2*time.Second; got != want {
		t.Errorf("ReadinessTimeout = %s, want %s", got, want)
	}
}

func TestLoadConfigParsesTimeoutOverrides(t *testing.T) {
	environment := validEnvironment()
	environment["HTTP_ADDR"] = "0.0.0.0:9090"
	environment["HTTP_READ_HEADER_TIMEOUT"] = "3s"
	environment["HTTP_READ_TIMEOUT"] = "4s"
	environment["HTTP_WRITE_TIMEOUT"] = "5s"
	environment["HTTP_IDLE_TIMEOUT"] = "30s"
	environment["HTTP_SHUTDOWN_TIMEOUT"] = "7s"
	environment["READINESS_TIMEOUT"] = "750ms"

	config, err := LoadConfig(mapLookup(environment))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got, want := config.HTTPAddress, environment["HTTP_ADDR"]; got != want {
		t.Errorf("HTTPAddress = %q, want %q", got, want)
	}
	if got, want := config.ReadHeaderTimeout, 3*time.Second; got != want {
		t.Errorf("ReadHeaderTimeout = %s, want %s", got, want)
	}
	if got, want := config.ReadTimeout, 4*time.Second; got != want {
		t.Errorf("ReadTimeout = %s, want %s", got, want)
	}
	if got, want := config.WriteTimeout, 5*time.Second; got != want {
		t.Errorf("WriteTimeout = %s, want %s", got, want)
	}
	if got, want := config.IdleTimeout, 30*time.Second; got != want {
		t.Errorf("IdleTimeout = %s, want %s", got, want)
	}
	if got, want := config.ShutdownTimeout, 7*time.Second; got != want {
		t.Errorf("ShutdownTimeout = %s, want %s", got, want)
	}
	if got, want := config.ReadinessTimeout, 750*time.Millisecond; got != want {
		t.Errorf("ReadinessTimeout = %s, want %s", got, want)
	}
}

func TestLoadConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name        string
		variable    string
		value       string
		wantInError string
	}{
		{name: "empty database DSN", variable: "DATABASE_DSN", value: "", wantInError: "DATABASE_DSN"},
		{name: "short JWT secret", variable: "JWT_SECRET", value: "too-short", wantInError: "JWT_SECRET"},
		{name: "empty JWT issuer", variable: "JWT_ISSUER", value: "", wantInError: "JWT_ISSUER"},
		{name: "empty JWT audience", variable: "JWT_AUDIENCE", value: "", wantInError: "JWT_AUDIENCE"},
		{name: "invalid HTTP address", variable: "HTTP_ADDR", value: "localhost", wantInError: "HTTP_ADDR"},
		{name: "invalid duration", variable: "HTTP_READ_TIMEOUT", value: "later", wantInError: "HTTP_READ_TIMEOUT"},
		{name: "non-positive duration", variable: "READINESS_TIMEOUT", value: "0s", wantInError: "READINESS_TIMEOUT"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := validEnvironment()
			environment[test.variable] = test.value

			_, err := LoadConfig(mapLookup(environment))
			if err == nil {
				t.Fatalf("LoadConfig() error = nil, want error containing %q", test.wantInError)
			}
			if !strings.Contains(err.Error(), test.wantInError) {
				t.Fatalf("LoadConfig() error = %q, want %q", err, test.wantInError)
			}
			if test.value != "" && strings.Contains(err.Error(), test.value) {
				t.Fatalf("LoadConfig() error leaked rejected value: %q", err)
			}
		})
	}
}

func validEnvironment() map[string]string {
	return map[string]string{
		"DATABASE_DSN": "community:private-password@tcp(localhost:3306)/community",
		"JWT_SECRET":   "0123456789abcdef0123456789abcdef",
		"JWT_ISSUER":   "investment-community",
		"JWT_AUDIENCE": "investment-community-api",
	}
}

func mapLookup(environment map[string]string) func(string) string {
	return func(name string) string {
		return environment[name]
	}
}
