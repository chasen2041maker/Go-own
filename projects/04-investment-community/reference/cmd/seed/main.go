package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/platform"
)

const seedTimeout = 30 * time.Second
const seedConfirmation = "fictional-development-data"

func main() {
	if err := runSeed(os.Getenv); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
	slog.Info("fictional community seed applied")
}

func runSeed(getenv func(string) string) error {
	if getenv == nil {
		return errors.New("seed environment reader is required")
	}
	dsn := strings.TrimSpace(getenv("DATABASE_DSN"))
	confirmation := strings.TrimSpace(getenv("SEED_CONFIRM"))
	adminPassword := getenv("SEED_ADMIN_PASSWORD")
	userPassword := getenv("SEED_USER_PASSWORD")
	if dsn == "" {
		return errors.New("DATABASE_DSN is required")
	}
	if confirmation != seedConfirmation {
		return errors.New("SEED_CONFIRM must explicitly allow fictional development data")
	}
	if adminPassword == "" || userPassword == "" {
		return errors.New("SEED_ADMIN_PASSWORD and SEED_USER_PASSWORD are required")
	}

	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return errors.New("open seed database")
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		return errors.New("connect to seed database")
	}
	return applySeedAtomically(ctx, database, platform.NewPasswordHasher(), defaultSeedData(adminPassword, userPassword))
}

func applySeedAtomically(ctx context.Context, database *sql.DB, hasher seedPasswordHasher, data seedData) error {
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("begin seed transaction")
	}
	defer transaction.Rollback()
	// 用户密码、虚构证券和圈子是同一个演示快照；中途失败不能留下半套 Seed。
	if err := applySeed(ctx, transaction, hasher, data); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return errors.New("commit seed transaction")
	}
	return nil
}

func defaultSeedData(adminPassword, userPassword string) seedData {
	return seedData{
		Users: []seedUser{
			{Email: "admin@starharbor.example.test", DisplayName: "星港管理员", Password: adminPassword, Role: domain.RoleAdmin},
			{Email: "learner@starharbor.example.test", DisplayName: "星港学习者", Password: userPassword, Role: domain.RoleUser},
		},
		// 全部代码、名称与交易所均为教学虚构，不映射真实公司或证券。
		Securities: []seedSecurity{
			{Exchange: "XSEA", Code: "AURR", Name: "曙环资源", Status: domain.SecurityStatusActive},
			{Exchange: "XSEA", Code: "NOVA", Name: "星湾科技", Status: domain.SecurityStatusActive},
			{Exchange: "XNOVA", Code: "TIDE", Name: "潮汐工坊", Status: domain.SecurityStatusActive},
			{Exchange: "XSEA", Code: "MIST", Name: "雾原样本", Status: domain.SecurityStatusInactive},
		},
		Circles: []seedCircle{
			{Slug: "long-horizon", Name: "长期观察", Description: "讨论长期视角下的虚构案例"},
			{Slug: "risk-lab", Name: "风险实验室", Description: "用虚构情境练习风险识别"},
		},
	}
}
