# GoKeep dev script: load root .env and start the Go gateway
# Usage: powershell ./scripts/dev-server.ps1   (works on Windows PowerShell 5.1 and pwsh)
$ErrorActionPreference = 'Stop'

# Auto-detect portable Go toolchain (when go is not on PATH)
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    $goRoots = @(
        'D:\All\tools\go-1.24.13',
        "$env:USERPROFILE\scoop\apps\go\current",
        "$env:USERPROFILE\go"
    )
    foreach ($root in $goRoots) {
        $candidate = Join-Path $root 'bin\go.exe'
        if (Test-Path $candidate) {
            $env:GOROOT = $root
            $env:Path = "$root\bin;$env:Path"
            break
        }
    }
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "go toolchain not found. Install Go 1.22+ or add the portable go bin dir to PATH." -ForegroundColor Red
    exit 1
}
if (-not $env:GOPROXY) {
    $env:GOPROXY = 'https://goproxy.cn,direct'
}

$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root '.env'
if (-not (Test-Path $envFile)) {
    Write-Host "Missing $envFile - copy .env.example to .env first." -ForegroundColor Yellow
    exit 1
}
foreach ($line in Get-Content $envFile -Encoding UTF8) {
    $line = $line.Trim()
    if ($line -match '^\s*#' -or $line -eq '') { continue }
    if ($line -match '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)\s*$') {
        [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
    }
}
$ver = (& go version) 2>$null
Write-Host "Loaded $envFile  |  $ver"
Set-Location (Join-Path $root 'server')
& go run ./cmd/server
