package investmentcommunity_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestOpenAPIContainsEveryRegisteredOperation(t *testing.T) {
	root := projectRoot(t)
	want := map[string]string{
		"registerUser": "POST /api/v1/auth/register", "loginUser": "POST /api/v1/auth/login",
		"getCurrentUser": "GET /api/v1/me", "listSecurities": "GET /api/v1/securities",
		"listCircles": "GET /api/v1/circles", "setCircleMembership": "PUT /api/v1/circles/{circleId}/membership",
		"listPosts": "GET /api/v1/posts", "createPost": "POST /api/v1/posts",
		"getPost": "GET /api/v1/posts/{postId}", "updatePost": "PATCH /api/v1/posts/{postId}",
		"deletePost": "DELETE /api/v1/posts/{postId}", "listComments": "GET /api/v1/posts/{postId}/comments",
		"createComment": "POST /api/v1/posts/{postId}/comments", "deleteComment": "DELETE /api/v1/comments/{commentId}",
		"createReport": "POST /api/v1/reports", "listNotifications": "GET /api/v1/notifications",
		"markAllNotificationsRead": "PUT /api/v1/notifications/read", "listAdminReports": "GET /api/v1/admin/reports",
		"decideAdminReport":   "POST /api/v1/admin/reports/{reportId}/decision",
		"restoreAdminContent": "POST /api/v1/admin/content/{targetType}/{targetId}/restore",
		"listAdminAuditLogs":  "GET /api/v1/admin/audit-logs",
	}
	got := openAPIOperations(t, filepath.Join(root, "contracts", "openapi.yaml"))
	if len(got) != len(want) {
		t.Fatalf("OpenAPI operations = %d, want exactly %d", len(got), len(want))
	}
	for operation, route := range want {
		if got[operation] != route {
			t.Errorf("%s = %q, want %q", operation, got[operation], route)
		}
	}

	// 路由源码仍是运行事实源；同时检查路径字面量，防止契约新增操作却忘记装配 Handler。
	routeSources, err := filepath.Glob(filepath.Join(root, "reference", "internal", "httpapi", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var production strings.Builder
	for _, path := range routeSources {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		production.Write(contents)
	}
	for _, route := range want {
		path := strings.SplitN(route, " ", 2)[1]
		if !strings.Contains(production.String(), `"`+path+`"`) {
			t.Errorf("registered route source is missing %s", path)
		}
	}
}

func TestStarterDoesNotImportReference(t *testing.T) {
	root := projectRoot(t)
	for _, isolated := range []string{"starter", "acceptance"} {
		err := filepath.WalkDir(filepath.Join(root, isolated), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return walkErr
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(contents), "projects/04-investment-community/reference") {
				t.Errorf("%s imports reference implementation: %s", isolated, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestWorkflowRunsGofmtGate(t *testing.T) {
	workflow := filepath.Join(projectRoot(t), "..", "..", ".github", "workflows", "investment-community.yml")
	contents, err := os.ReadFile(workflow)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), `test -z "$(gofmt -l projects/04-investment-community)"`) {
		t.Fatal("default CI job is missing the Linux gofmt gate")
	}
}

func TestDeliveryFilesKeepLocalSecurityBoundary(t *testing.T) {
	root := projectRoot(t)
	dockerfile := mustRead(t, filepath.Join(root, "Dockerfile"))
	for _, want := range []string{"FROM swaggerapi/swagger-ui:v5.29.5 AS swagger-assets", "FROM nginxinc/nginx-unprivileged:1.29.1-alpine AS swagger"} {
		if !strings.Contains(dockerfile, want) {
			t.Errorf("Dockerfile missing %q", want)
		}
	}
	stagePattern := regexp.MustCompile(`(?ms)^FROM nginxinc/nginx-unprivileged:1\.29\.1-alpine AS swagger\r?$\n(.*?)(?:^FROM |\z)`)
	match := stagePattern.FindStringSubmatch(dockerfile)
	if match == nil || !regexp.MustCompile(`(?m)^USER 101\r?$`).MatchString(match[1]) ||
		!strings.Contains(match[1], "COPY projects/04-investment-community/swagger-initializer.js /usr/share/nginx/html/swagger-initializer.js") {
		t.Fatal("final swagger stage must copy its initializer and run explicitly as USER 101")
	}
	ignore := mustRead(t, filepath.Join(root, "Dockerfile.dockerignore"))
	for _, want := range []string{"**/.env", "**/.env.*", "!**/.env.example", "*.pem", "*.key", "*.p12", "*.pfx", "**/id_rsa"} {
		if !strings.Contains(ignore, want) {
			t.Errorf("Dockerfile.dockerignore missing %q", want)
		}
	}
	compose := mustRead(t, filepath.Join(root, "compose.yaml"))
	for _, want := range []string{"127.0.0.1:${COMMUNITY_MYSQL_PORT:-13385}:3306", "127.0.0.1:${COMMUNITY_API_PORT:-8084}:8084", "127.0.0.1:${COMMUNITY_SWAGGER_PORT:-8085}:8080"} {
		if !strings.Contains(compose, want) {
			t.Errorf("compose port is not loopback-bound: %s", want)
		}
	}
}

func TestLocalVerificationDocsUseOnePasswordAndDisposableSchema(t *testing.T) {
	root := projectRoot(t)
	readme := mustRead(t, filepath.Join(root, "README.md"))
	for _, want := range []string{
		"$env:COMMUNITY_TEST_DSN = ./projects/04-investment-community/scripts/create-integration-schema.ps1",
		"$env:COMMUNITY_ACCEPTANCE_ADMIN_PASSWORD = $env:COMMUNITY_ADMIN_PASSWORD",
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("project README is missing executable verification instruction %q", want)
		}
	}
	script := mustRead(t, filepath.Join(root, "scripts", "create-integration-schema.ps1"))
	for _, want := range []string{"investment_community_test", "GRANT ALL PRIVILEGES", "COMMUNITY_MYSQL_ROOT_PASSWORD", "COMMUNITY_MYSQL_PASSWORD"} {
		if !strings.Contains(script, want) {
			t.Errorf("integration schema helper missing %q", want)
		}
	}
	workflow := mustRead(t, filepath.Join(root, "..", "..", ".github", "workflows", "investment-community.yml"))
	if !strings.Contains(workflow, "docker image inspect") || !strings.Contains(workflow, ".Config.User") {
		t.Fatal("Compose cold-start CI must inspect the final Swagger image user")
	}
}

func TestLearningDocsStartSeededAPIBeforeAcceptance(t *testing.T) {
	root := projectRoot(t)
	for _, relative := range []string{filepath.Join("docs", "learning", "README.md"), filepath.Join("docs", "learning", "stage-08.md")} {
		contents := mustRead(t, filepath.Join(root, relative))
		composeIndex := strings.Index(contents, "docker compose -f ./projects/04-investment-community/compose.yaml up -d --build --wait")
		acceptanceIndex := strings.Index(contents, "go test -tags=acceptance ./projects/04-investment-community/acceptance")
		if composeIndex < 0 || acceptanceIndex < 0 || composeIndex > acceptanceIndex {
			t.Errorf("%s must start the seeded API before running acceptance", relative)
		}
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}

func openAPIOperations(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	pathPattern := regexp.MustCompile(`^  (/[^:]+):$`)
	serverPattern := regexp.MustCompile(`^  - url: (/[^ ]+)$`)
	methodPattern := regexp.MustCompile(`^    (get|post|put|patch|delete):$`)
	operationPattern := regexp.MustCompile(`^      operationId: ([A-Za-z0-9]+)$`)
	operations := make(map[string]string)
	var serverURL, currentPath, currentMethod string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if match := serverPattern.FindStringSubmatch(line); match != nil {
			serverURL = strings.TrimRight(match[1], "/")
		}
		if match := pathPattern.FindStringSubmatch(line); match != nil {
			currentPath = match[1]
		}
		if match := methodPattern.FindStringSubmatch(line); match != nil {
			currentMethod = strings.ToUpper(match[1])
		}
		if match := operationPattern.FindStringSubmatch(line); match != nil {
			// OpenAPI 把共同前缀放在 servers.url，运行路由则保存完整路径。
			operations[match[1]] = currentMethod + " " + serverURL + currentPath
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return operations
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate delivery test")
	}
	return filepath.Dir(current)
}
