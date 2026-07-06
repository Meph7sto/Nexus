param(
    [string]$ComposeFile = "docker-compose.local.yml",
    [switch]$Pull,
    [switch]$Logs,
    [switch]$UseLocalBuild
)

$ErrorActionPreference = "Stop"

function New-RandomHex {
    param([int]$Bytes = 32)

    $buffer = New-Object byte[] $Bytes
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $rng.GetBytes($buffer)
    } finally {
        $rng.Dispose()
    }
    return -join ($buffer | ForEach-Object { $_.ToString("x2") })
}

function Set-EnvValue {
    param(
        [string]$Path,
        [string]$Name,
        [string]$Value
    )

    $content = Get-Content -LiteralPath $Path -Raw
    $escapedName = [regex]::Escape($Name)
    if ($content -match "(?m)^$escapedName=") {
        $content = [regex]::Replace($content, "(?m)^$escapedName=.*$", "$Name=$Value")
    } else {
        $content = $content.TrimEnd() + [Environment]::NewLine + "$Name=$Value" + [Environment]::NewLine
    }
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($Path, $content, $utf8NoBom)
}

function Get-EnvValue {
    param(
        [string]$Path,
        [string]$Name
    )

    $line = Get-Content -LiteralPath $Path | Where-Object { $_ -match "^$([regex]::Escape($Name))=" } | Select-Object -First 1
    if (-not $line) {
        return ""
    }
    return ($line -split "=", 2)[1].Trim()
}

$deployDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -LiteralPath $deployDir

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker was not found. Install Docker Desktop, start it, then run this script again."
}

$composeFiles = @($ComposeFile)
if ($UseLocalBuild) {
    $composeFiles += "docker-compose.windows.yml"
}

$composeArgs = @("compose")
foreach ($file in $composeFiles) {
    $composePath = Join-Path $deployDir $file
    if (-not (Test-Path -LiteralPath $composePath)) {
        throw "Compose file not found: $composePath"
    }
    $composeArgs += @("-f", $file)
}

$envPath = Join-Path $deployDir ".env"
$envExamplePath = Join-Path $deployDir ".env.example"
if (-not (Test-Path -LiteralPath $envPath)) {
    if (-not (Test-Path -LiteralPath $envExamplePath)) {
        throw ".env does not exist and .env.example was not found."
    }

    Copy-Item -LiteralPath $envExamplePath -Destination $envPath
    Set-EnvValue -Path $envPath -Name "SERVER_PORT" -Value "18080"
    Set-EnvValue -Path $envPath -Name "POSTGRES_PASSWORD" -Value (New-RandomHex 24)
    Set-EnvValue -Path $envPath -Name "JWT_SECRET" -Value (New-RandomHex 32)
    Set-EnvValue -Path $envPath -Name "TOTP_ENCRYPTION_KEY" -Value (New-RandomHex 32)

    Write-Host "Created deploy\\.env with generated secrets."
} else {
    if ((Get-EnvValue -Path $envPath -Name "SERVER_PORT") -eq "8080") {
        Set-EnvValue -Path $envPath -Name "SERVER_PORT" -Value "18080"
        Write-Host "Changed SERVER_PORT to 18080 in deploy\\.env to avoid common local port conflicts."
    }

    $postgresPassword = Get-EnvValue -Path $envPath -Name "POSTGRES_PASSWORD"
    if ([string]::IsNullOrWhiteSpace($postgresPassword) -or $postgresPassword -eq "change_this_secure_password") {
        Set-EnvValue -Path $envPath -Name "POSTGRES_PASSWORD" -Value (New-RandomHex 24)
        Write-Host "Generated POSTGRES_PASSWORD in deploy\\.env."
    }

    if ([string]::IsNullOrWhiteSpace((Get-EnvValue -Path $envPath -Name "JWT_SECRET"))) {
        Set-EnvValue -Path $envPath -Name "JWT_SECRET" -Value (New-RandomHex 32)
        Write-Host "Generated JWT_SECRET in deploy\\.env."
    }

    if ([string]::IsNullOrWhiteSpace((Get-EnvValue -Path $envPath -Name "TOTP_ENCRYPTION_KEY"))) {
        Set-EnvValue -Path $envPath -Name "TOTP_ENCRYPTION_KEY" -Value (New-RandomHex 32)
        Write-Host "Generated TOTP_ENCRYPTION_KEY in deploy\\.env."
    }
}

New-Item -ItemType Directory -Force -Path `
    (Join-Path $deployDir "data"), `
    (Join-Path $deployDir "postgres_data"), `
    (Join-Path $deployDir "redis_data") | Out-Null

if ($Pull) {
    & docker @($composeArgs + @("pull"))
}

$upArgs = $composeArgs + @("up", "-d")
if ($UseLocalBuild) {
    $upArgs += "--build"
}
& docker @upArgs

$port = Get-EnvValue -Path $envPath -Name "SERVER_PORT"
if ([string]::IsNullOrWhiteSpace($port)) {
    $port = "8080"
}

Write-Host ""
Write-Host "Nexus is starting."
Write-Host "Web UI: http://localhost:$port"
$composeDisplay = (($composeFiles | ForEach-Object { "-f $_" }) -join " ")
Write-Host "Logs:   docker compose $composeDisplay logs -f nexus"

if ($Logs) {
    & docker @($composeArgs + @("logs", "-f", "nexus"))
}
