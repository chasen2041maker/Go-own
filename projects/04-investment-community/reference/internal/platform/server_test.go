package platform

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPServerAppliesConfiguredTimeouts(t *testing.T) {
	config := Config{
		HTTPAddress:       "127.0.0.1:8084",
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      4 * time.Second,
		IdleTimeout:       5 * time.Second,
	}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})

	server := NewHTTPServer(config, handler)

	if got, want := server.Addr, config.HTTPAddress; got != want {
		t.Errorf("Addr = %q, want %q", got, want)
	}
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if got, want := response.Code, http.StatusNoContent; got != want {
		t.Errorf("installed Handler status = %d, want %d", got, want)
	}
	if got, want := server.ReadHeaderTimeout, config.ReadHeaderTimeout; got != want {
		t.Errorf("ReadHeaderTimeout = %s, want %s", got, want)
	}
	if got, want := server.ReadTimeout, config.ReadTimeout; got != want {
		t.Errorf("ReadTimeout = %s, want %s", got, want)
	}
	if got, want := server.WriteTimeout, config.WriteTimeout; got != want {
		t.Errorf("WriteTimeout = %s, want %s", got, want)
	}
	if got, want := server.IdleTimeout, config.IdleTimeout; got != want {
		t.Errorf("IdleTimeout = %s, want %s", got, want)
	}
}

func TestServeShutsDownAfterContextCancellation(t *testing.T) {
	server := newFakeHTTPServer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := Serve(ctx, server, time.Second); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if !server.shutdownCalled {
		t.Fatal("Serve() did not call Shutdown")
	}
	if !server.shutdownHadDeadline {
		t.Fatal("Shutdown context did not have a deadline")
	}
	if server.closeCalled {
		t.Fatal("Serve() called Close after successful Shutdown")
	}
}

func TestServeClosesServerWhenGracefulShutdownFails(t *testing.T) {
	server := newFakeHTTPServer()
	server.shutdownErr = errors.New("stuck request")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Serve(ctx, server, time.Second)
	if err == nil || !strings.Contains(err.Error(), "shutdown") {
		t.Fatalf("Serve() error = %v, want shutdown error", err)
	}
	if !server.closeCalled {
		t.Fatal("Serve() did not force Close after Shutdown failed")
	}
}

func TestServeReturnsUnexpectedListenError(t *testing.T) {
	want := errors.New("listen failed")
	server := newFakeHTTPServer()
	server.listenErr = want
	close(server.stop)

	err := Serve(context.Background(), server, time.Second)
	if !errors.Is(err, want) {
		t.Fatalf("Serve() error = %v, want %v", err, want)
	}
}

type fakeHTTPServer struct {
	stop                chan struct{}
	listenErr           error
	shutdownErr         error
	shutdownCalled      bool
	shutdownHadDeadline bool
	closeCalled         bool
}

func newFakeHTTPServer() *fakeHTTPServer {
	return &fakeHTTPServer{stop: make(chan struct{})}
}

func (server *fakeHTTPServer) ListenAndServe() error {
	<-server.stop
	if server.listenErr != nil {
		return server.listenErr
	}
	return http.ErrServerClosed
}

func (server *fakeHTTPServer) Shutdown(ctx context.Context) error {
	server.shutdownCalled = true
	_, server.shutdownHadDeadline = ctx.Deadline()
	if server.shutdownErr == nil {
		closeIfOpen(server.stop)
	}
	return server.shutdownErr
}

func (server *fakeHTTPServer) Close() error {
	server.closeCalled = true
	closeIfOpen(server.stop)
	return nil
}

func closeIfOpen(channel chan struct{}) {
	select {
	case <-channel:
	default:
		close(channel)
	}
}
