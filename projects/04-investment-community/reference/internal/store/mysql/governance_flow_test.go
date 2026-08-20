package mysql

import (
	"errors"
	"reflect"
	"testing"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func TestDecisionSnapshotIsLoadedBeforeCommit(t *testing.T) {
	events := make([]string, 0, 2)
	current := domain.AdminReport{ID: 9, Target: domain.ContentSnapshot{Visibility: domain.VisibilityHidden, ModerationVersion: 2}}

	result, err := snapshotAndCommitDecision(func() (domain.AdminReport, error) {
		events = append(events, "snapshot")
		return current, nil
	}, func() error {
		events = append(events, "commit")
		// 模拟提交后另一事务立即 restore；若快照在 commit 后读取，响应会错误看到 visible/version=3。
		current.Target.Visibility = domain.VisibilityVisible
		current.Target.ModerationVersion = 3
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events, []string{"snapshot", "commit"}) {
		t.Fatalf("events = %#v, want snapshot before commit", events)
	}
	if result.Target.Visibility != domain.VisibilityHidden || result.Target.ModerationVersion != 2 {
		t.Fatalf("result = %#v, want transaction snapshot", result)
	}
}

func TestDecisionSnapshotFailureDoesNotCommit(t *testing.T) {
	wantErr := errors.New("snapshot failed")
	commitCalls := 0
	_, err := snapshotAndCommitDecision(func() (domain.AdminReport, error) {
		return domain.AdminReport{}, wantErr
	}, func() error {
		commitCalls++
		return nil
	})
	if !errors.Is(err, wantErr) || commitCalls != 0 {
		t.Fatalf("error = %v, commit calls = %d", err, commitCalls)
	}
}
