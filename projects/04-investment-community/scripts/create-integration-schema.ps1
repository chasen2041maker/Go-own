[CmdletBinding()]
param(
    [string]$ComposeFile = (Join-Path $PSScriptRoot '..\compose.yaml')
)

$ErrorActionPreference = 'Stop'
$rootPassword = if ($env:COMMUNITY_MYSQL_ROOT_PASSWORD) { $env:COMMUNITY_MYSQL_ROOT_PASSWORD } else { 'root_local_only' }
$appPassword = if ($env:COMMUNITY_MYSQL_PASSWORD) { $env:COMMUNITY_MYSQL_PASSWORD } else { 'community_local_only' }
$mysqlPort = if ($env:COMMUNITY_MYSQL_PORT) { $env:COMMUNITY_MYSQL_PORT } else { '13385' }

# integration 会重建自己管理的表，因此单独创建 schema；绝不能复用 API 正在服务的演示库。
$sql = @"
CREATE DATABASE IF NOT EXISTS investment_community_test
  CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
GRANT ALL PRIVILEGES ON investment_community_test.* TO 'community'@'%';
FLUSH PRIVILEGES;
"@

docker compose -f $ComposeFile exec -T -e "MYSQL_PWD=$rootPassword" mysql mysql -uroot -e $sql
if ($LASTEXITCODE -ne 0) {
    throw '创建专用 integration schema 失败；请先确认 Compose MySQL 已健康。'
}

# 只输出 DSN，调用者可直接赋给当前 PowerShell 的环境变量。
Write-Output "community:$($appPassword)@tcp(127.0.0.1:$mysqlPort)/investment_community_test?parseTime=true&loc=UTC"
