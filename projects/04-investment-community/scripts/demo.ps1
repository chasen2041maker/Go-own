[CmdletBinding()]
param(
    [string]$BaseUrl = 'http://127.0.0.1:8084',
    [string]$AdminPassword = 'LocalAdminPass!2026',
    [switch]$SkipCompose,
    [switch]$Down
)

$ErrorActionPreference = 'Stop'
$compose = Join-Path $PSScriptRoot '..\compose.yaml'
$repository = Resolve-Path (Join-Path $PSScriptRoot '..\..\..\..')

if ($Down) {
    docker compose -f $compose down
    exit $LASTEXITCODE
}

if (-not $SkipCompose) {
    docker compose -f $compose up -d --build --wait
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

$previousBaseUrl = $env:COMMUNITY_ACCEPTANCE_BASE_URL
$previousAdminPassword = $env:COMMUNITY_ACCEPTANCE_ADMIN_PASSWORD
try {
    # 验收凭据只进入当前子进程环境；测试不会打印 Token 或密码。
    $env:COMMUNITY_ACCEPTANCE_BASE_URL = $BaseUrl
    $env:COMMUNITY_ACCEPTANCE_ADMIN_PASSWORD = $AdminPassword
    Push-Location $repository
    try {
        go test -tags=acceptance ./projects/04-investment-community/acceptance -run '^TestCommunityGovernanceJourney$' -count=1 -v
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }
} finally {
    $env:COMMUNITY_ACCEPTANCE_BASE_URL = $previousBaseUrl
    $env:COMMUNITY_ACCEPTANCE_ADMIN_PASSWORD = $previousAdminPassword
}

Write-Host "Swagger UI: http://127.0.0.1:8085"
