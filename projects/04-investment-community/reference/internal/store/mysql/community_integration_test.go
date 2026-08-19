//go:build integration

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
	"go-own/projects/04-investment-community/reference/migrations"
)

func TestSecurityStoreSearchesActivePrefixAndPaginatesByCodeAndID(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	suffix := integrationSuffix()
	prefix := "Q" + suffix[:8]
	exchange := "X" + suffix[:8]
	otherExchange := "Y" + suffix[:8]
	insertSecurity := func(market, code, name, status string) int64 {
		t.Helper()
		result, err := database.ExecContext(ctx,
			"INSERT INTO securities (market, code, name, status) VALUES (?, ?, ?, ?)", market, code, name, status)
		if err != nil {
			t.Fatalf("insert security %s: %v", code, err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("security id: %v", err)
		}
		return id
	}
	ids := []int64{
		insertSecurity(exchange, prefix+"A", "测试星云甲", "active"),
		insertSecurity(exchange, prefix+"B", "测试星云乙", "active"),
		insertSecurity(exchange, prefix+"C", "测试星云丙", "active"),
		insertSecurity(exchange, prefix+"D", "测试星云停用", "inactive"),
		insertSecurity(otherExchange, prefix+"E", "测试星云异所", "active"),
		insertSecurity(otherExchange, prefix+"B", "测试星云同码", "active"),
	}
	t.Cleanup(func() { deleteIntegrationIDs(database, "securities", ids) })

	service := mustIntegrationCommunityService(t, store)
	first, err := service.ListSecurities(ctx, usecase.SecurityListInput{Query: prefix, Exchange: exchange, Limit: 2})
	if err != nil {
		t.Fatalf("ListSecurities(first) error = %v", err)
	}
	if len(first.Items) != 2 || first.Items[0].Code != prefix+"A" || first.Items[1].Code != prefix+"B" || first.Next == nil {
		t.Fatalf("first page = %#v", first)
	}
	second, err := service.ListSecurities(ctx, usecase.SecurityListInput{
		Query: prefix, Exchange: exchange, Limit: 2, After: first.Next,
	})
	if err != nil {
		t.Fatalf("ListSecurities(second) error = %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].Code != prefix+"C" || second.Next != nil {
		t.Fatalf("second page = %#v", second)
	}
	byName, err := service.ListSecurities(ctx, usecase.SecurityListInput{Query: "测试星云", Exchange: exchange, Limit: 10})
	if err != nil || len(byName.Items) != 3 {
		t.Fatalf("name-prefix search = %#v, error = %v", byName, err)
	}
	tieFirst, err := service.ListSecurities(ctx, usecase.SecurityListInput{Query: prefix + "B", Limit: 1})
	if err != nil || len(tieFirst.Items) != 1 || tieFirst.Next == nil {
		t.Fatalf("same-code first page = %#v, error = %v", tieFirst, err)
	}
	tieSecond, err := service.ListSecurities(ctx, usecase.SecurityListInput{Query: prefix + "B", Limit: 1, After: tieFirst.Next})
	if err != nil || len(tieSecond.Items) != 1 || tieSecond.Items[0].ID == tieFirst.Items[0].ID {
		t.Fatalf("same-code second page = %#v, error = %v", tieSecond, err)
	}
}

func TestCircleStoreListsActiveCirclesWithMembershipAndStablePagination(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	suffix := integrationSuffix()
	userID := insertIntegrationUser(t, ctx, database, suffix)
	// 目录没有测试专用筛选；用未来时间让本测试的三条夹具稳定排在共享测试库现有数据之前。
	createdAt := time.Date(2099, 8, 19, 12, 0, 0, 123000000, time.UTC)
	insertCircle := func(index int, status string, at time.Time) int64 {
		t.Helper()
		result, err := database.ExecContext(ctx, `
INSERT INTO circles (slug, name, description, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`, fmt.Sprintf("circle-%s-%d", suffix, index), fmt.Sprintf("集成圈子%s-%d", suffix, index), "虚构圈子", status, at, at)
		if err != nil {
			t.Fatalf("insert circle: %v", err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	oldestID := insertCircle(1, "active", createdAt.Add(-time.Minute))
	firstTieID := insertCircle(2, "active", createdAt)
	secondTieID := insertCircle(3, "active", createdAt)
	archivedID := insertCircle(4, "archived", createdAt.Add(time.Minute))
	circleIDs := []int64{oldestID, firstTieID, secondTieID, archivedID}
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM circle_memberships WHERE user_id = ?", userID)
		deleteIntegrationIDs(database, "circles", circleIDs)
		deleteIntegrationIDs(database, "users", []int64{userID})
	})
	if _, err := database.ExecContext(ctx, "INSERT INTO circle_memberships (circle_id, user_id) VALUES (?, ?)", secondTieID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	service := mustIntegrationCommunityService(t, store)
	first, err := service.ListCircles(ctx, usecase.CircleListInput{UserID: userID, Limit: 2})
	if err != nil {
		t.Fatalf("ListCircles(first) error = %v", err)
	}
	if len(first.Items) != 2 || first.Items[0].ID != secondTieID || first.Items[1].ID != firstTieID ||
		!first.Items[0].IsMember || first.Items[0].MemberCount != 1 || first.Next == nil {
		t.Fatalf("first page = %#v", first)
	}
	second, err := service.ListCircles(ctx, usecase.CircleListInput{UserID: userID, Limit: 2, After: first.Next})
	if err != nil {
		t.Fatalf("ListCircles(second) error = %v", err)
	}
	if len(second.Items) < 1 || second.Items[0].ID != oldestID {
		t.Fatalf("second page = %#v", second)
	}
}

func TestConcurrentMembershipJoinAndRepeatedLeaveConvergeOnOneRow(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	suffix := integrationSuffix()
	userID := insertIntegrationUser(t, ctx, database, suffix)
	result, err := database.ExecContext(ctx,
		"INSERT INTO circles (slug, name, description, status) VALUES (?, ?, '', 'active')",
		"membership-"+suffix, "成员集成圈子"+suffix)
	if err != nil {
		t.Fatalf("insert circle: %v", err)
	}
	circleID, _ := result.LastInsertId()
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM circle_memberships WHERE circle_id = ?", circleID)
		deleteIntegrationIDs(database, "circles", []int64{circleID})
		deleteIntegrationIDs(database, "users", []int64{userID})
	})
	for _, joined := range []bool{true, false} {
		if _, err := store.SetCircleMembership(ctx, math.MaxInt64, userID, joined); !errors.Is(err, domain.ErrCircleNotFound) {
			t.Fatalf("SetCircleMembership(missing, joined=%t) error = %v, want ErrCircleNotFound", joined, err)
		}
	}

	const workers = 8
	start := make(chan struct{})
	results := make(chan domain.CircleMembership, workers)
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			membership, err := store.SetCircleMembership(ctx, circleID, userID, true)
			if err != nil {
				errors <- err
				return
			}
			results <- membership
		}()
	}
	close(start)
	group.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Fatalf("concurrent SetCircleMembership() error = %v", err)
	}
	var joinedAt time.Time
	for membership := range results {
		if !membership.Joined || membership.JoinedAt == nil {
			t.Fatalf("membership = %#v", membership)
		}
		if joinedAt.IsZero() {
			joinedAt = *membership.JoinedAt
		} else if !joinedAt.Equal(*membership.JoinedAt) {
			t.Fatalf("joined_at changed: %s != %s", joinedAt, *membership.JoinedAt)
		}
	}
	var count int
	if err := database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM circle_memberships WHERE circle_id = ? AND user_id = ?", circleID, userID).Scan(&count); err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if count != 1 {
		t.Fatalf("membership rows = %d, want 1", count)
	}

	for range 2 {
		membership, err := store.SetCircleMembership(ctx, circleID, userID, false)
		if err != nil || membership.Joined || membership.JoinedAt != nil {
			t.Fatalf("leave result = %#v, error = %v", membership, err)
		}
	}
	if err := database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM circle_memberships WHERE circle_id = ? AND user_id = ?", circleID, userID).Scan(&count); err != nil {
		t.Fatalf("count memberships after leave: %v", err)
	}
	if count != 0 {
		t.Fatalf("membership rows after leave = %d, want 0", count)
	}
}

func TestMembershipForeignKeysRejectUnknownIDs(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suffix := integrationSuffix()
	userID := insertIntegrationUser(t, ctx, database, suffix)
	result, err := database.ExecContext(ctx,
		"INSERT INTO circles (slug, name, description, status) VALUES (?, ?, '', 'active')",
		"fk-membership-"+suffix, "外键圈子"+suffix)
	if err != nil {
		t.Fatalf("insert circle: %v", err)
	}
	circleID, _ := result.LastInsertId()
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM circle_memberships WHERE circle_id = ?", circleID)
		deleteIntegrationIDs(database, "circles", []int64{circleID})
		deleteIntegrationIDs(database, "users", []int64{userID})
	})

	if _, err := store.SetCircleMembership(ctx, circleID, math.MaxInt64, true); err == nil {
		t.Fatal("joining with an unknown user bypassed the membership foreign key")
	}
	if _, err := store.SetCircleMembership(ctx, math.MaxInt64, userID, true); !errors.Is(err, domain.ErrCircleNotFound) {
		t.Fatalf("joining an unknown circle error = %v, want ErrCircleNotFound", err)
	}
}

func TestConcurrentMembershipJoinAndLeaveNeverMisreportCircleMissing(t *testing.T) {
	database, store := openCommunityIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	suffix := integrationSuffix()
	userID := insertIntegrationUser(t, ctx, database, suffix)
	result, err := database.ExecContext(ctx,
		"INSERT INTO circles (slug, name, description, status) VALUES (?, ?, '', 'active')",
		"mixed-membership-"+suffix, "混合并发圈子"+suffix)
	if err != nil {
		t.Fatalf("insert circle: %v", err)
	}
	circleID, _ := result.LastInsertId()
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM circle_memberships WHERE circle_id = ?", circleID)
		deleteIntegrationIDs(database, "circles", []int64{circleID})
		deleteIntegrationIDs(database, "users", []int64{userID})
	})

	const workers = 20
	start := make(chan struct{})
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for index := range workers {
		joined := index%2 == 0
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			membership, err := store.SetCircleMembership(ctx, circleID, userID, joined)
			if err != nil {
				errors <- err
				return
			}
			if membership.Joined != joined || (joined && membership.JoinedAt == nil) || (!joined && membership.JoinedAt != nil) {
				errors <- fmt.Errorf("membership result does not match requested state: %#v", membership)
			}
		}()
	}
	close(start)
	group.Wait()
	close(errors)
	for err := range errors {
		t.Fatalf("mixed membership operation error = %v", err)
	}
}

func openCommunityIntegrationStore(t *testing.T) (*sql.DB, *Store) {
	t.Helper()
	dsn := os.Getenv("COMMUNITY_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNITY_TEST_DSN is not set")
	}
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := migrations.Apply(ctx, database); err != nil {
		t.Fatalf("migrations.Apply() error = %v", err)
	}
	store, err := New(database)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return database, store
}

func mustIntegrationCommunityService(t *testing.T, store *Store) *usecase.CommunityService {
	t.Helper()
	service, err := usecase.NewCommunityService(store)
	if err != nil {
		t.Fatalf("NewCommunityService() error = %v", err)
	}
	return service
}

func insertIntegrationUser(t *testing.T, ctx context.Context, database *sql.DB, suffix string) int64 {
	t.Helper()
	result, err := database.ExecContext(ctx, `
INSERT INTO users (email, password_hash, display_name, role, status)
VALUES (?, '$2a$10$integration-only-hash', ?, 'user', 'active')`, "member-"+suffix+"@example.test", "集成用户"+suffix)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

func integrationSuffix() string {
	value := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(value) < 8 {
		return fmt.Sprintf("%08s", value)
	}
	return value[len(value)-8:]
}

func deleteIntegrationIDs(database *sql.DB, table string, ids []int64) {
	for _, id := range ids {
		_, _ = database.Exec("DELETE FROM "+table+" WHERE id = ?", id) // #nosec G201 -- table 只来自测试内固定字符串。
	}
}
