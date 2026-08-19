package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// HTTPServer 是 Serve 所需的最小生命周期接口，使关闭逻辑无需真实端口也能测试。
type HTTPServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Close() error
}

// NewHTTPServer 显式应用全部网络超时；零值会让慢客户端无限占用连接。
func NewHTTPServer(config Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              config.HTTPAddress,
		Handler:           handler,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
	}
}

// Serve 持续运行到监听失败或 ctx 取消。关闭时另建有界上下文，因为信号上下文此时已经取消。
func Serve(ctx context.Context, server HTTPServer, shutdownTimeout time.Duration) error {
	if server == nil {
		return fmt.Errorf("serve HTTP: server is required")
	}
	if shutdownTimeout <= 0 {
		return fmt.Errorf("serve HTTP: shutdown timeout must be positive")
	}

	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErrors:
		return normalizeServeError(err)
	case <-ctx.Done():
	}

	// 先停止接收新请求，再给正在处理的请求一个固定收尾窗口。
	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		shutdownErr := fmt.Errorf("shutdown HTTP server: %w", err)
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(shutdownErr, fmt.Errorf("close HTTP server: %w", closeErr))
		}
		return shutdownErr
	}

	return normalizeServeError(<-serveErrors)
}

func normalizeServeError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("serve HTTP: %w", err)
}
