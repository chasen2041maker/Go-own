package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/internal/httpapi"
	"go-own/projects/04-investment-community/reference/internal/platform"
)

func main() {
	config, err := platform.LoadConfig(os.Getenv)
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	database, err := sql.Open("mysql", config.DatabaseDSN)
	if err != nil {
		slog.Error("create database pool", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	// 个人项目也必须显式限制连接池；无上限连接会在数据库变慢时放大故障。
	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)
	database.SetConnMaxIdleTime(1 * time.Minute)

	// API 可以先启动，再由 /readyz 诚实反映数据库状态；迁移由独立命令执行。
	router := httpapi.NewRouter(database, config.ReadinessTimeout)
	server := platform.NewHTTPServer(config, router)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("starting API", "address", config.HTTPAddress)
	if err := platform.Serve(ctx, server, config.ShutdownTimeout); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}
