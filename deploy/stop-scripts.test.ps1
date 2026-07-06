$ErrorActionPreference = "Stop"

function Assert {
    param(
        [bool]$Condition,
        [string]$Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$rootStop = Get-Content -LiteralPath (Join-Path $repoRoot "stop.bat") -Raw
$deployStop = Get-Content -LiteralPath (Join-Path $PSScriptRoot "stop-windows.cmd") -Raw

Assert ($rootStop -notmatch "(?im)^\s*pause\s*$") "stop.bat must not wait for manual terminal closure."
Assert ($deployStop -notmatch "(?im)^\s*pause\s*$") "deploy\stop-windows.cmd must not wait for manual terminal closure."

foreach ($title in @(
    "Nexus Docker App Logs",
    "Nexus Docker Postgres Logs",
    "Nexus Docker Redis Logs"
)) {
    Assert ($rootStop.Contains($title)) "stop.bat must close the $title terminal."
}

Write-Host "Stop script terminal behavior checks passed."
