$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

function Write-Step {
    param([string]$Message)
    $line = "[deskact-non-github-ci] $Message"
    Write-Host $line
    Add-Content -LiteralPath $Script:LocalLog -Value $line
}

function Find-StagedBuild {
    param([string]$RepoRoot)

    $artifactRoot = Join-Path $RepoRoot 'artifacts\non-github-ci'
    if (-not (Test-Path -LiteralPath $artifactRoot)) {
        return $null
    }

    return Get-ChildItem -LiteralPath $artifactRoot -Directory |
        ForEach-Object {
            $script = Join-Path $_.FullName 'vm\run-windows-build.ps1'
            if (Test-Path -LiteralPath $script) {
                [pscustomobject]@{
                    Version = $_.Name
                    Script = $script
                    LastWriteTime = (Get-Item -LiteralPath $script).LastWriteTimeUtc
                }
            }
        } |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
}

$Script:LocalLog = 'C:\OEM\deskact-oem.ps1.log'
Set-Content -LiteralPath $Script:LocalLog -Value "[deskact-non-github-ci] Windows OEM build started $(Get-Date -Format o)"

$Share = '\\host.lan\Data'
$RepoMarker = 'scripts\non-github-ci\bootstrap-windows.ps1'
for ($i = 0; $i -lt 180; $i++) {
    if (Test-Path -LiteralPath (Join-Path $Share $RepoMarker)) {
        break
    }
    Start-Sleep -Seconds 10
}
if (-not (Test-Path -LiteralPath (Join-Path $Share $RepoMarker))) {
    throw "shared repo did not appear at $Share"
}

Write-Step "mapping shared repo from $Share"
cmd.exe /c "net use Z: $Share /persistent:no" | Out-Host
$RepoRoot = 'Z:\'

$staged = $null
for ($i = 0; $i -lt 60; $i++) {
    $staged = Find-StagedBuild -RepoRoot $RepoRoot
    if ($staged) {
        break
    }
    Start-Sleep -Seconds 10
}
if (-not $staged) {
    throw 'no staged Windows build script was found under artifacts\non-github-ci'
}

$OutputDir = Join-Path $RepoRoot "artifacts\non-github-ci\$($staged.Version)"
$LogDir = Join-Path $OutputDir 'logs'
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$SharedLog = Join-Path $LogDir 'windows-oem.log'
Copy-Item -LiteralPath $Script:LocalLog -Destination $SharedLog -Force
Start-Transcript -Path $SharedLog -Append | Out-Null

try {
    Set-Location $RepoRoot
    Write-Step "running Windows bootstrap for $($staged.Version)"
    & (Join-Path $RepoRoot 'scripts\non-github-ci\bootstrap-windows.ps1')

    Write-Step "running staged Windows build: $($staged.Script)"
    & $staged.Script

    Write-Step 'Windows OEM build completed'
} finally {
    Stop-Transcript | Out-Null
}
