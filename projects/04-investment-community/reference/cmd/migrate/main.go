package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("数据库迁移失败", "error", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return fmt.Errorf("DATABASE_DSN 不能为空")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("创建数据库连接池: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("连接数据库: %w", err)
	}
	if err := migrations.Apply(ctx, db); err != nil {
		return err
	}

	slog.Info("数据库迁移完成")
	return nil
}
