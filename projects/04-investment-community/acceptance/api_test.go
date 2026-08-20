//go:build acceptance

package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCommunityGovernanceJourney(t *testing.T) {
	baseURL := os.Getenv("COMMUNITY_ACCEPTANCE_BASE_URL")
	if baseURL == "" {
		t.Fatal("COMMUNITY_ACCEPTANCE_BASE_URL is required for acceptance tests")
	}

	// 黑盒测试只持有 HTTP 地址，不导入 reference/internal；这样 starter 也能用同一契约验收。
	client := newJourneyClient(t, baseURL)
	client.runCommunityGovernanceJourney()
}

func TestFailureDiagnosticRedactsAuthenticationAndContent(t *testing.T) {
	contents := []byte(`{"access_token":"sentinel-token","body":"sentinel-body","error":{"code":"conflict","request_id":"req-safe"}}`)
	got := safeResponseSummary(contents)
	for _, secret := range []string{"sentinel-token", "sentinel-body", "access_token"} {
		if strings.Contains(got, secret) {
			t.Fatalf("diagnostic leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "conflict") || !strings.Contains(got, "req-safe") {
		t.Fatalf("diagnostic omitted safe error fields: %s", got)
	}
}

type journeyClient struct {
	t       *testing.T
	baseURL string
	http    *http.Client
	called  map[string]bool
}

type authResult struct {
	AccessToken string `json:"access_token"`
	User        struct {
		ID int64 `json:"id"`
	} `json:"user"`
}

type identified struct {
	ID int64 `json:"id"`
}

type catalogItem struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Slug string `json:"slug"`
}

type catalogPage struct {
	Items []catalogItem `json:"items"`
}

type postResult struct {
	ID                int64 `json:"id"`
	Version           int64 `json:"version"`
	ModerationVersion int64 `json:"moderation_version"`
}

type notificationPage struct {
	Items []struct {
		Type string `json:"type"`
	} `json:"items"`
}

type auditPage struct {
	Items []struct {
		Action     string `json:"action"`
		TargetType string `json:"target_type"`
		TargetID   int64  `json:"target_id"`
	} `json:"items"`
}

type decisionResult struct {
	Target struct {
		ModerationVersion int64 `json:"moderation_version"`
	} `json:"target"`
}

type adminReportPage struct {
	Items []struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	} `json:"items"`
}

func newJourneyClient(t *testing.T, baseURL string) *journeyClient {
	t.Helper()
	return &journeyClient{
		t:       t,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
		called:  make(map[string]bool),
	}
}

func (client *journeyClient) runCommunityGovernanceJourney() {
	client.t.Helper()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	password := "LocalJourney!" + suffix
	alice := client.register("alice_"+suffix+"@example.test", "Alice "+suffix, password)
	bob := client.register("bob_"+suffix+"@example.test", "Bob "+suffix, password)
	alice = client.login("alice_"+suffix+"@example.test", password)
	client.request("getCurrentUser", http.MethodGet, "/api/v1/me", alice.AccessToken, "", nil, http.StatusOK, nil)
	adminPassword := os.Getenv("COMMUNITY_ACCEPTANCE_ADMIN_PASSWORD")
	if adminPassword == "" {
		client.t.Fatal("COMMUNITY_ACCEPTANCE_ADMIN_PASSWORD is required for acceptance tests")
	}
	adminEmail := os.Getenv("COMMUNITY_ACCEPTANCE_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@starharbor.example.test"
	}
	admin := client.login(adminEmail, adminPassword)

	var securities catalogPage
	client.request("listSecurities", http.MethodGet, "/api/v1/securities?limit=20", alice.AccessToken, "", nil, http.StatusOK, &securities)
	securityID := catalogID(client.t, securities.Items, "AURR", false)
	var circles catalogPage
	client.request("listCircles", http.MethodGet, "/api/v1/circles?limit=20", alice.AccessToken, "", nil, http.StatusOK, &circles)
	circleID := catalogID(client.t, circles.Items, "long-horizon", true)

	for _, token := range []string{alice.AccessToken, bob.AccessToken} {
		client.request("setCircleMembership", http.MethodPut, fmt.Sprintf("/api/v1/circles/%d/membership", circleID), token, "",
			map[string]any{"joined": true}, http.StatusOK, nil)
	}
	var post postResult
	client.request("createPost", http.MethodPost, "/api/v1/posts", alice.AccessToken, "post-"+suffix,
		map[string]any{"circle_id": circleID, "title": "虚构长期观察 " + suffix, "body": "仅用于黑盒验收的虚构观点。", "security_ids": []int64{securityID}},
		http.StatusCreated, &post)
	var updated postResult
	client.request("updatePost", http.MethodPatch, fmt.Sprintf("/api/v1/posts/%d", post.ID), alice.AccessToken, "",
		map[string]any{"version": post.Version, "title": "虚构长期观察（已更新） " + suffix}, http.StatusOK, &updated)
	if updated.Version != post.Version+1 {
		client.t.Fatalf("updated post version = %d, want %d", updated.Version, post.Version+1)
	}
	client.request("listPosts", http.MethodGet, fmt.Sprintf("/api/v1/posts?circle_id=%d&limit=20", circleID), alice.AccessToken, "", nil, http.StatusOK, nil)

	var bobComment identified
	client.request("createComment", http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), bob.AccessToken, "comment-bob-"+suffix,
		map[string]any{"body": "Bob 的虚构评论"}, http.StatusCreated, &bobComment)
	var aliceComment identified
	client.request("createComment", http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), alice.AccessToken, "comment-alice-"+suffix,
		map[string]any{"body": "Alice 的虚构顶级评论"}, http.StatusCreated, &aliceComment)
	client.request("createComment", http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), bob.AccessToken, "reply-"+suffix,
		map[string]any{"body": "Bob 的虚构回复", "parent_comment_id": aliceComment.ID}, http.StatusCreated, nil)
	client.request("listComments", http.MethodGet, fmt.Sprintf("/api/v1/posts/%d/comments?limit=20", post.ID), alice.AccessToken, "", nil, http.StatusOK, nil)
	client.requireNotificationTypes(alice.AccessToken, "comment", "reply")
	client.request("markAllNotificationsRead", http.MethodPut, "/api/v1/notifications/read", alice.AccessToken, "", nil, http.StatusOK, nil)

	var report identified
	client.request("createReport", http.MethodPost, "/api/v1/reports", bob.AccessToken, "", map[string]any{
		"target_type": "post", "target_id": post.ID, "reason": "misleading", "details": "黑盒验收虚构举报",
	}, http.StatusCreated, &report)
	var reports adminReportPage
	client.request("listAdminReports", http.MethodGet, "/api/v1/admin/reports?status=pending&limit=100", admin.AccessToken, "", nil, http.StatusOK, &reports)
	if !containsPendingReport(reports.Items, report.ID) {
		client.t.Fatalf("pending admin report %d not found", report.ID)
	}
	var decision decisionResult
	client.request("decideAdminReport", http.MethodPost, fmt.Sprintf("/api/v1/admin/reports/%d/decision", report.ID), admin.AccessToken, "",
		map[string]any{"decision": "hide", "note": "黑盒验收隐藏"}, http.StatusOK, &decision)
	if decision.Target.ModerationVersion <= post.ModerationVersion {
		client.t.Fatalf("hide moderation_version = %d, want > %d", decision.Target.ModerationVersion, post.ModerationVersion)
	}
	client.request("getPost", http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", post.ID), alice.AccessToken, "", nil, http.StatusNotFound, nil)
	client.requireNotificationTypes(alice.AccessToken, "content_hidden")
	client.requireAudit(admin.AccessToken, post.ID, "content_hidden")

	var restored postResult
	client.request("restoreAdminContent", http.MethodPost, fmt.Sprintf("/api/v1/admin/content/post/%d/restore", post.ID), admin.AccessToken, "",
		map[string]any{"expected_moderation_version": decision.Target.ModerationVersion}, http.StatusOK, &restored)
	if restored.ModerationVersion != decision.Target.ModerationVersion+1 {
		client.t.Fatalf("restore moderation_version = %d, want %d", restored.ModerationVersion, decision.Target.ModerationVersion+1)
	}
	client.request("getPost", http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", post.ID), alice.AccessToken, "", nil, http.StatusOK, nil)
	client.requireNotificationTypes(alice.AccessToken, "content_restored")
	client.requireAudit(admin.AccessToken, post.ID, "content_restored")

	// 删除放在恢复与审计断言之后，既覆盖作者删除操作，也不破坏治理旅程的可观察终态。
	client.request("deleteComment", http.MethodDelete, fmt.Sprintf("/api/v1/comments/%d", bobComment.ID), bob.AccessToken, "", nil, http.StatusNoContent, nil)
	client.request("deletePost", http.MethodDelete, fmt.Sprintf("/api/v1/posts/%d", post.ID), alice.AccessToken, "", nil, http.StatusNoContent, nil)
	client.requireEveryOperation()
}

func containsPendingReport(items []struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}, reportID int64) bool {
	for _, item := range items {
		if item.ID == reportID && item.Status == "pending" {
			return true
		}
	}
	return false
}

func (client *journeyClient) register(email, displayName, password string) authResult {
	var result authResult
	client.request("registerUser", http.MethodPost, "/api/v1/auth/register", "", "", map[string]any{
		"email": email, "display_name": displayName, "password": password,
	}, http.StatusCreated, &result)
	return result
}

func (client *journeyClient) login(email, password string) authResult {
	var result authResult
	client.request("loginUser", http.MethodPost, "/api/v1/auth/login", "", "", map[string]any{
		"email": email, "password": password,
	}, http.StatusOK, &result)
	return result
}

func (client *journeyClient) requireNotificationTypes(token string, wanted ...string) {
	client.t.Helper()
	var page notificationPage
	client.request("listNotifications", http.MethodGet, "/api/v1/notifications?limit=100", token, "", nil, http.StatusOK, &page)
	for _, want := range wanted {
		found := false
		for _, item := range page.Items {
			found = found || item.Type == want
		}
		if !found {
			client.t.Errorf("notification type %q not found", want)
		}
	}
}

func (client *journeyClient) requireAudit(token string, targetID int64, action string) {
	client.t.Helper()
	var page auditPage
	client.request("listAdminAuditLogs", http.MethodGet, "/api/v1/admin/audit-logs?limit=100", token, "", nil, http.StatusOK, &page)
	for _, item := range page.Items {
		if item.Action == action && item.TargetType == "post" && item.TargetID == targetID {
			return
		}
	}
	client.t.Errorf("audit action %q for post %d not found", action, targetID)
}

func (client *journeyClient) request(operationID, method, path, token, idempotencyKey string, body any, wantStatus int, destination any) {
	client.t.Helper()
	client.called[operationID] = true
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			client.t.Fatal(err)
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, client.baseURL+path, payload)
	if err != nil {
		client.t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response, err := client.http.Do(request)
	if err != nil {
		client.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		client.t.Fatal(err)
	}
	if response.StatusCode != wantStatus {
		client.t.Fatalf("%s %s status = %d, want %d; %s", method, path, response.StatusCode, wantStatus, safeResponseSummary(contents))
	}
	if got := response.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		client.t.Errorf("%s %s X-Content-Type-Options = %q", method, path, got)
	}
	if destination != nil && len(contents) > 0 {
		if err := json.Unmarshal(contents, destination); err != nil {
			client.t.Fatalf("decode %s %s: %v; %s", method, path, err, safeResponseSummary(contents))
		}
	}
}

func (client *journeyClient) requireEveryOperation() {
	client.t.Helper()
	want := []string{
		"registerUser", "loginUser", "getCurrentUser", "listSecurities", "listCircles", "setCircleMembership",
		"listPosts", "createPost", "getPost", "updatePost", "deletePost", "listComments", "createComment",
		"deleteComment", "createReport", "listNotifications", "markAllNotificationsRead", "listAdminReports",
		"decideAdminReport", "restoreAdminContent", "listAdminAuditLogs",
	}
	if len(client.called) != len(want) {
		client.t.Errorf("called operation count = %d, want %d: %#v", len(client.called), len(want), client.called)
	}
	for _, operationID := range want {
		if !client.called[operationID] {
			client.t.Errorf("operation %s was not called by journey", operationID)
		}
	}
}

func safeResponseSummary(contents []byte) string {
	var envelope struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if json.Unmarshal(contents, &envelope) == nil && envelope.Error.Code != "" {
		return fmt.Sprintf("error_code=%q request_id=%q", envelope.Error.Code, envelope.Error.RequestID)
	}
	return fmt.Sprintf("response_body_redacted bytes=%d", len(contents))
}

func catalogID(t *testing.T, items []catalogItem, value string, slug bool) int64 {
	t.Helper()
	for _, item := range items {
		if (!slug && item.Code == value) || (slug && item.Slug == value) {
			return item.ID
		}
	}
	t.Fatalf("catalog item %q not found", value)
	return 0
}
