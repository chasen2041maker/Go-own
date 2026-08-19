package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"go-own/projects/04-investment-community/starter/internal/health"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/healthz", health.NewHandler())

	// 教学顺序：先完成路由装配，再设置边界超时，最后才开始监听。
	// 只监听回环地址，避免学习用 starter 被意外暴露到局域网。
	server := &http.Server{
		Addr:              "127.0.0.1:8081",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("starter API listening on http://%s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
