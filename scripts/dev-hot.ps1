# GoKeep backend hot-reload script
# Watches server/internal + server/cmd *.go files; auto rebuild & restart on change.
# Usage: powershell -ExecutionPolicy Bypass -File .\scripts\dev-hot.ps1
# (Windows PowerShell 5.1 compatible; Ctrl+C to stop)
$ErrorActionPreference = 'Stop'

# --- portable go toolchain detection (same as dev-server.ps1) ---
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    $goRoots = @(
        'D:\All\tools\go-1.24.13',
        "$env:USERPROFILE\scoop\apps\go\current",
        "$env:USERPROFILE\go"
    )
    foreach ($r in $goRoots) {
        $candidate = Join-Path $r 'bin\go.exe'
        if (Test-Path $candidate) {
            $env:GOROOT = $r
            $env:Path = "$r\bin;$env:Path"
            break
        }
    }
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "go toolchain not found." -ForegroundColor Red
    exit 1
}
if (-not $env:GOPROXY) { $env:GOPROXY = 'https://goproxy.cn,direct' }
$goExe = (Get-Command go).Source

# --- load .env ---
$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root '.env'
if (-not (Test-Path $envFile)) {
    Write-Host "Missing $envFile" -ForegroundColor Yellow
    exit 1
}
foreach ($line in Get-Content $envFile -Encoding UTF8) {
    $line = $line.Trim()
    if ($line -match '^\s*#' -or $line -eq '') { continue }
    if ($line -match '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)\s*$') {
        [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
    }
}

$serverDir = Join-Path $root 'server'

# --- source change stamp (watches internal/ and cmd/ .go files) ---
function Get-SrcStamp {
    $sum = [int64]0
    Get-ChildItem -Path "$serverDir\internal", "$serverDir\cmd" -Recurse -Filter '*.go' -ErrorAction SilentlyContinue |
        ForEach-Object { $sum += $_.LastWriteTimeUtc.Ticks }
    return $sum
}

function Stop-ServerTree {
    if ($script:proc -and -not $script:proc.HasExited) {
        & taskkill /PID $script:proc.Id /T /F 2>$null | Out-Null
    }
    $script:proc = $null
}

function Start-Server {
    $script:proc = Start-Process -FilePath $goExe `
        -ArgumentList 'run', './cmd/server' `
        -WorkingDirectory $serverDir -PassThru -NoNewWindow
    Start-Sleep -Seconds 2
}

Write-Host "[hot] watching *.go under server/internal & server/cmd ... (Ctrl+C to stop)" -ForegroundColor Cyan
$script:proc = $null
$stamp = Get-SrcStamp

while ($true) {
    Start-Server
    if ($script:proc.HasExited) {
        Write-Host "[hot] server exited with code $($script:proc.ExitCode) - restarting in 2s ..." -ForegroundColor Yellow
        Start-Sleep -Seconds 2
        $stamp = Get-SrcStamp
        continue
    }
    # monitor until process exits or source changes
    $restart = $false
    while (-not $script:proc.HasExited) {
        Start-Sleep -Seconds 1
        $newStamp = Get-SrcStamp
        if ($newStamp -ne $stamp) {
            $stamp = $newStamp
            $restart = $true
            break
        }
    }
    if ($restart) {
        Write-Host "[hot] source changed - restarting ..." -ForegroundColor Cyan
        Stop-ServerTree
        Start-Sleep -Milliseconds 800   # wait port release
        continue
    }
}
