$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

function Write-Step {
    param([string]$Message)
    Write-Host "[deskact-non-github-ci] $Message"
}

function Get-EnvOrDefault {
    param(
        [string]$Name,
        [string]$Default
    )
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $Default
    }
    return $value
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
        throw "Missing command after bootstrap: $Name"
    }
}

function Download-File {
    param(
        [string]$Url,
        [string]$Destination
    )
    if (Test-Path -LiteralPath $Destination) {
        return
    }
    Write-Step "downloading $Url"
    Invoke-WebRequest -Uri $Url -OutFile $Destination -UseBasicParsing
}

function Install-Msi {
    param(
        [string]$Path,
        [string]$Name
    )
    Write-Step "installing $Name"
    $process = Start-Process -FilePath msiexec.exe -ArgumentList @('/i', $Path, '/qn', '/norestart') -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "$Name installer failed with exit code $($process.ExitCode)"
    }
}

function Ensure-Go {
    Refresh-Path
    if (Get-Command go -ErrorAction SilentlyContinue) {
        return
    }
    $version = Get-EnvOrDefault -Name 'DESKACT_WINDOWS_GO_VERSION' -Default '1.25.0'
    $url = Get-EnvOrDefault -Name 'DESKACT_WINDOWS_GO_MSI_URL' -Default "https://go.dev/dl/go$version.windows-amd64.msi"
    $msi = Join-Path $Script:CacheDir "go-$version.windows-amd64.msi"
    Download-File -Url $url -Destination $msi
    Install-Msi -Path $msi -Name "Go $version"
}

function Ensure-Node {
    Refresh-Path
    if (Get-Command node -ErrorAction SilentlyContinue) {
        return
    }
    $version = Get-EnvOrDefault -Name 'DESKACT_WINDOWS_NODE_VERSION' -Default '20.18.1'
    $url = Get-EnvOrDefault -Name 'DESKACT_WINDOWS_NODE_MSI_URL' -Default "https://nodejs.org/dist/v$version/node-v$version-x64.msi"
    $msi = Join-Path $Script:CacheDir "node-v$version-x64.msi"
    Download-File -Url $url -Destination $msi
    Install-Msi -Path $msi -Name "Node.js $version"
}

function Ensure-MSYS2 {
    $msysBash = 'C:\msys64\usr\bin\bash.exe'
    if (Test-Path -LiteralPath $msysBash) {
        return
    }

    $url = Get-EnvOrDefault -Name 'DESKACT_WINDOWS_MSYS2_INSTALLER_URL' -Default 'https://github.com/msys2/msys2-installer/releases/latest/download/msys2-x86_64-latest.exe'
    $installer = Join-Path $Script:CacheDir 'msys2-x86_64-latest.exe'
    Download-File -Url $url -Destination $installer

    Write-Step 'installing MSYS2'
    $arguments = @('in', '--confirm-command', '--accept-messages', '--root', 'C:\msys64')
    $process = Start-Process -FilePath $installer -ArgumentList $arguments -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "MSYS2 installer failed with exit code $($process.ExitCode)"
    }

    if (-not (Test-Path -LiteralPath $msysBash)) {
        throw "MSYS2 bash was not found at $msysBash"
    }
}

function Ensure-MingwGcc {
    Refresh-Path
    if (Get-Command gcc -ErrorAction SilentlyContinue) {
        return
    }

    Write-Step 'installing MinGW GCC with MSYS2'
    $msysBash = 'C:\msys64\usr\bin\bash.exe'
    & $msysBash -lc 'pacman -Syuu --needed --noconfirm || true'
    & $msysBash -lc 'pacman -Sy --needed --noconfirm mingw-w64-x86_64-gcc'
    if ($LASTEXITCODE -ne 0) {
        throw "MSYS2 pacman failed with exit code $LASTEXITCODE"
    }
    Refresh-Path
}

$Script:CacheDir = Join-Path $env:TEMP 'deskact-build-tools'
New-Item -ItemType Directory -Force -Path $Script:CacheDir | Out-Null

Write-Step 'installing Windows build tools without requiring winget'
Ensure-Go
Ensure-Node
Ensure-MSYS2
Ensure-MingwGcc

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Step 'git is unavailable; manifest gitCommit will be empty unless Git is installed manually'
}

Require-Command go
Require-Command node
Require-Command npm
Require-Command gcc

Write-Step 'Windows builder bootstrap complete'
Write-Step 'Run artifacts\non-github-ci\<version>\vm\run-windows-build.ps1 from the shared repo checkout.'
