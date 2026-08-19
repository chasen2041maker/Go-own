// Package platform 保存进程级配置和 HTTP 生命周期代码。
package platform

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const minimumJWTSecretBytes = 32

// Config 是组装 API 前必须完成校验的进程配置。
// 密钥只保存在该结构中，错误信息不得回显其原始值。
type Config struct {
	HTTPAddress       string
	DatabaseDSN       string
	JWTSecret         string
	JWTIssuer         string
	JWTAudience       string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	ReadinessTimeout  time.Duration
}

// LoadConfig 通过传入的 getenv 读取配置，使测试无需修改进程环境；生产传 os.Getenv。
func LoadConfig(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, fmt.Errorf("config: getenv is required")
	}

	config := Config{
		HTTPAddress:       valueOrDefault(getenv("HTTP_ADDR"), "127.0.0.1:8084"),
		DatabaseDSN:       strings.TrimSpace(getenv("DATABASE_DSN")),
		JWTSecret:         getenv("JWT_SECRET"),
		JWTIssuer:         strings.TrimSpace(getenv("JWT_ISSUER")),
		JWTAudience:       strings.TrimSpace(getenv("JWT_AUDIENCE")),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		ReadinessTimeout:  2 * time.Second,
	}

	var err error
	if config.ReadHeaderTimeout, err = durationValue(getenv, "HTTP_READ_HEADER_TIMEOUT", config.ReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if config.ReadTimeout, err = durationValue(getenv, "HTTP_READ_TIMEOUT", config.ReadTimeout); err != nil {
		return Config{}, err
	}
	if config.WriteTimeout, err = durationValue(getenv, "HTTP_WRITE_TIMEOUT", config.WriteTimeout); err != nil {
		return Config{}, err
	}
	if config.IdleTimeout, err = durationValue(getenv, "HTTP_IDLE_TIMEOUT", config.IdleTimeout); err != nil {
		return Config{}, err
	}
	if config.ShutdownTimeout, err = durationValue(getenv, "HTTP_SHUTDOWN_TIMEOUT", config.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	if config.ReadinessTimeout, err = durationValue(getenv, "READINESS_TIMEOUT", config.ReadinessTimeout); err != nil {
		return Config{}, err
	}

	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

// Validate 在监听端口前拒绝不安全或不可用的配置。
func (config Config) Validate() error {
	if config.DatabaseDSN == "" {
		return fmt.Errorf("config: DATABASE_DSN is required")
	}
	if strings.TrimSpace(config.JWTSecret) != config.JWTSecret || len(config.JWTSecret) < minimumJWTSecretBytes {
		return fmt.Errorf("config: JWT_SECRET must contain at least %d bytes and no surrounding whitespace", minimumJWTSecretBytes)
	}
	if config.JWTIssuer == "" {
		return fmt.Errorf("config: JWT_ISSUER is required")
	}
	if config.JWTAudience == "" {
		return fmt.Errorf("config: JWT_AUDIENCE is required")
	}
	if err := validateAddress(config.HTTPAddress); err != nil {
		return err
	}

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{name: "HTTP_READ_HEADER_TIMEOUT", value: config.ReadHeaderTimeout},
		{name: "HTTP_READ_TIMEOUT", value: config.ReadTimeout},
		{name: "HTTP_WRITE_TIMEOUT", value: config.WriteTimeout},
		{name: "HTTP_IDLE_TIMEOUT", value: config.IdleTimeout},
		{name: "HTTP_SHUTDOWN_TIMEOUT", value: config.ShutdownTimeout},
		{name: "READINESS_TIMEOUT", value: config.ReadinessTimeout},
	}
	for _, timeout := range timeouts {
		if timeout.value <= 0 {
			return fmt.Errorf("config: %s must be a positive duration", timeout.name)
		}
	}
	return nil
}

func durationValue(getenv func(string) string, name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(getenv(name))
	if raw == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		// 只报告变量名，不回显可能敏感或恶意构造的原始值。
		return 0, fmt.Errorf("config: %s must be a positive Go duration", name)
	}
	return duration, nil
}

func validateAddress(address string) error {
	_, rawPort, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("config: HTTP_ADDR must use host:port form")
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("config: HTTP_ADDR must contain a port from 1 to 65535")
	}
	return nil
}

func valueOrDefault(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}
