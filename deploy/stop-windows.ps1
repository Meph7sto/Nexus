param(
    [string]$ComposeFile = "docker-compose.local.yml",
    [switch]$RemoveVolumes,
    [switch]$UseLocalBuild
)

$ErrorActionPreference = "Stop"

$deployDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -LiteralPath $deployDir

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker was not found. Install Docker Desktop, start it, then run this script again."
}

$composeFiles = @($ComposeFile)
if ($UseLocalBuild) {
    $composeFiles += "docker-compose.windows.yml"
}

$dockerArgs = @("compose")
foreach ($file in $composeFiles) {
    $composePath = Join-Path $deployDir $file
    if (-not (Test-Path -LiteralPath $composePath)) {
        throw "Compose file not found: $composePath"
    }
    $dockerArgs += @("-f", $file)
}

$dockerArgs += "down"
if ($RemoveVolumes) {
    $dockerArgs += "-v"
}

& docker @dockerArgs

Write-Host ""
if ($RemoveVolumes) {
    Write-Host "Nexus has stopped and Docker volumes were removed."
    Write-Host "Local directories such as deploy\\data are not deleted by this script."
} else {
    Write-Host "Nexus has stopped. Data is kept."
}
