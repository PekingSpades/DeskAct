param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [string]$OutputDir,
    [switch]$SkipElectron
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$ScriptDir = Split-Path -Parent $PSCommandPath
$RepoRoot = (Resolve-Path (Join-Path $ScriptDir '..\..')).Path
$Platform = 'windows-amd64'

if (-not $OutputDir) {
    $OutputDir = Join-Path $RepoRoot "artifacts\non-github-ci\$Version"
}
$OutputDir = [System.IO.Path]::GetFullPath($OutputDir)
$GoDir = Join-Path $OutputDir "go\$Platform"
$ExamplesDir = Join-Path $GoDir 'examples'
$ElectronDir = Join-Path $OutputDir "electron\$Platform"
$LogDir = Join-Path $OutputDir 'logs'
New-Item -ItemType Directory -Force -Path $GoDir, $ExamplesDir, $ElectronDir, $LogDir | Out-Null

function Write-Step {
    param([string]$Message)
    Write-Host "[deskact-non-github-ci] $Message"
}

function Refresh-Path {
    $machine = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    $user = [Environment]::GetEnvironmentVariable('Path', 'User')
    $knownPaths = @(
        (Join-Path $env:ProgramFiles 'Git\cmd'),
        (Join-Path $env:ProgramFiles 'Go\bin'),
        (Join-Path $env:ProgramFiles 'nodejs'),
        (Join-Path $env:ProgramFiles '7-Zip'),
        'C:\msys64\mingw64\bin',
        'C:\msys64\usr\bin'
    ) | Where-Object { $_ -and (Test-Path -LiteralPath $_) }
    $env:Path = (@($machine, $user) + $knownPaths) -join [System.IO.Path]::PathSeparator
}

function Require-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Missing command: $Name"
    }
}

function Get-RelativeArtifactPath {
    param([string]$Path)
    $base = [System.IO.Path]::GetFullPath($OutputDir).TrimEnd([char[]]@('\', '/'))
    $fullPath = [System.IO.Path]::GetFullPath($Path)
    $prefix = $base + [System.IO.Path]::DirectorySeparatorChar
    if ($fullPath.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        $relative = $fullPath.Substring($prefix.Length)
    } else {
        $relative = $fullPath
    }
    return $relative.Replace('\', '/')
}

function Write-ChecksumsAndManifest {
    $checksumPath = Join-Path $OutputDir 'SHA256SUMS'
    $manifestPath = Join-Path $OutputDir 'manifest.json'
    $artifactRoots = @(
        (Join-Path $OutputDir 'go'),
        (Join-Path $OutputDir 'electron')
    ) | Where-Object { Test-Path -LiteralPath $_ }

    $files = @()
    foreach ($root in $artifactRoots) {
        $files += Get-ChildItem -LiteralPath $root -File -Recurse | Sort-Object FullName
    }

    $checksumLines = @()
    $manifestFiles = @()
    foreach ($file in $files) {
        $relative = Get-RelativeArtifactPath -Path $file.FullName
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $file.FullName).Hash.ToLowerInvariant()
        $checksumLines += "$hash  $relative"
        $manifestFiles += [ordered]@{
            path = $relative
            sha256 = $hash
        }
    }

    Set-Content -LiteralPath $checksumPath -Value $checksumLines -Encoding Ascii
    $commit = ''
    try {
        $commit = (& git -C $RepoRoot rev-parse HEAD).Trim()
    } catch {
        $commit = ''
    }
    $manifest = [ordered]@{
        name = 'deskact-non-github-ci'
        version = $Version
        gitCommit = $commit
        generatedAt = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
        files = $manifestFiles
    }
    $manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $manifestPath -Encoding Ascii
    Write-Step "wrote $checksumPath"
    Write-Step "wrote $manifestPath"
}

Refresh-Path
Require-Command go
Require-Command node
Require-Command npm

$MingwBin = 'C:\msys64\mingw64\bin'
if (Test-Path -LiteralPath $MingwBin) {
    $env:Path = $MingwBin + [System.IO.Path]::PathSeparator + $env:Path
    $env:CC = Join-Path $MingwBin 'gcc.exe'
    $env:CXX = Join-Path $MingwBin 'g++.exe'
}
Require-Command gcc

$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '1'
$env:CGO_LDFLAGS_ALLOW = '-weak_framework|ScreenCaptureKit'

$targets = @(
    @{ Name = 'deskact-tester'; Package = './cmd/deskact-tester'; Output = (Join-Path $GoDir 'deskact-tester.exe') },
    @{ Name = 'capture'; Package = './examples/capture'; Output = (Join-Path $ExamplesDir 'capture.exe') },
    @{ Name = 'display'; Package = './examples/display'; Output = (Join-Path $ExamplesDir 'display.exe') },
    @{ Name = 'keyboard'; Package = './examples/keyboard'; Output = (Join-Path $ExamplesDir 'keyboard.exe') },
    @{ Name = 'mouse'; Package = './examples/mouse'; Output = (Join-Path $ExamplesDir 'mouse.exe') },
    @{ Name = 'window'; Package = './examples/window'; Output = (Join-Path $ExamplesDir 'window.exe') },
    @{ Name = 'apps'; Package = './examples/apps'; Output = (Join-Path $ExamplesDir 'apps.exe') },
    @{ Name = 'windowops'; Package = './examples/windowops'; Output = (Join-Path $ExamplesDir 'windowops.exe') },
    @{ Name = 'windowops-selftest'; Package = './examples/windowops/selftest'; Output = (Join-Path $ExamplesDir 'windowops-selftest.exe') }
)

foreach ($target in $targets) {
    Write-Step "go build $($target.Package) -> $($target.Output)"
    Push-Location $RepoRoot
    try {
        & go build -a -v -o $($target.Output) $($target.Package)
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed for $($target.Name) with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
}

if (-not $SkipElectron) {
    Write-Step "building Electron display for $Platform"
    $ElectronSource = Join-Path $RepoRoot 'examples\display\electron'
    Push-Location $ElectronSource
    try {
        Remove-Item -LiteralPath (Join-Path $ElectronSource 'dist') -Recurse -Force -ErrorAction SilentlyContinue
        npm install
        & npx electron-builder --win --x64 --publish never
        if ($LASTEXITCODE -ne 0) {
            throw "electron-builder failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }

    $BuildOutput = Join-Path $ElectronSource 'dist\win-unpacked'
    if (-not (Test-Path -LiteralPath $BuildOutput)) {
        $BuildOutput = Get-ChildItem -LiteralPath (Join-Path $ElectronSource 'dist') -Directory |
            Where-Object { $_.Name -like '*-unpacked' } |
            Select-Object -First 1 -ExpandProperty FullName
    }
    if (-not $BuildOutput -or -not (Test-Path -LiteralPath $BuildOutput)) {
        throw "Electron build output directory was not found"
    }

    $Archive = Join-Path $ElectronDir "electron-display-$Platform.zip"
    Remove-Item -LiteralPath $Archive -Force -ErrorAction SilentlyContinue
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::CreateFromDirectory(
        $BuildOutput,
        $Archive,
        [System.IO.Compression.CompressionLevel]::Optimal,
        $false
    )
    Write-Step "Electron display artifact ready: $Archive"
}

Write-ChecksumsAndManifest
Write-Step "Windows artifacts ready under $OutputDir"
