# Cross-compile vWireguard for a Linux server, from Windows.
#
#   powershell -ExecutionPolicy Bypass -File .\build-linux.ps1
#
# vWireguard has no cgo dependencies (no go-sqlite3), so a pure cross-compile
# works - you do NOT need a Linux box just to build.
#
# Optional switches:
#   -Arch arm64     build for ARM servers (Oracle Ampere, Hetzner ARM, ...)
#   -Check          only run `go build ./...`, `gofmt -l .` and `go vet ./...`
#                   (the same things CI checks) without producing a binary

param(
    [string]$Arch = "amd64",
    [switch]$Check
)

$ErrorActionPreference = "Stop"
$out = "vwireguard"

# ---------------------------------------------------------------------------
#  Locate the Go toolchain.
#  `go` is often installed but not on PATH in the current shell (the installer
#  adds it to the *machine* PATH, which existing terminals do not pick up until
#  they are restarted). So look in the usual places before giving up.
# ---------------------------------------------------------------------------
function Find-Go {
    $cmd = Get-Command go -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }

    $candidates = @(
        "$env:ProgramFiles\Go\bin\go.exe",
        "${env:ProgramFiles(x86)}\Go\bin\go.exe",
        "$env:LOCALAPPDATA\Programs\Go\bin\go.exe",
        "$env:USERPROFILE\go\bin\go.exe",
        "$env:USERPROFILE\scoop\apps\go\current\bin\go.exe",
        "C:\Go\bin\go.exe",
        "C:\tools\go\bin\go.exe"
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path $c)) { return $c }
    }
    return $null
}

$go = Find-Go

if (-not $go) {
    Write-Host ""
    Write-Host "  Go is not installed (or not on PATH in this window)." -ForegroundColor Red
    Write-Host ""
    Write-Host "  Install it with ONE of these, then open a NEW terminal:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "    winget install --id GoLang.Go -e"
    Write-Host "    choco install golang -y"
    Write-Host "    https://go.dev/dl/   (download the .msi and run it)"
    Write-Host ""
    Write-Host "  Any version >= 1.21 works (go.mod requires 1.21; CI uses 1.21)."
    Write-Host ""
    Write-Host "  Already installed? Your terminal just has a stale PATH." -ForegroundColor Yellow
    Write-Host "  Close this window and open a new PowerShell, or run:"
    Write-Host '    $env:Path += ";$env:ProgramFiles\Go\bin"'
    Write-Host ""
    exit 1
}

# Make sure the Go bin dir is on PATH for this session (needed by `go` itself).
$goBin = Split-Path $go -Parent
if ($env:Path -notlike "*$goBin*") { $env:Path = "$goBin;$env:Path" }

$goVersion = (& $go version)
Write-Host "[build] using $go" -ForegroundColor DarkGray
Write-Host "[build] $goVersion" -ForegroundColor DarkGray

# ---------------------------------------------------------------------------
#  -Check : run exactly what CI runs, so you never push a red build again.
# ---------------------------------------------------------------------------
if ($Check) {
    $failed = $false

    Write-Host ""
    Write-Host "[check] gofmt -l ." -ForegroundColor Cyan
    $fmt = & "$goBin\gofmt.exe" -l .
    if ($fmt) {
        Write-Host "  These files are not gofmt-clean (CI will fail):" -ForegroundColor Red
        $fmt | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
        Write-Host "  Fix them all with:  gofmt -w ." -ForegroundColor Yellow
        $failed = $true
    } else {
        Write-Host "  ok" -ForegroundColor Green
    }

    Write-Host ""
    Write-Host "[check] go build ./..." -ForegroundColor Cyan
    & $go build ./...
    if ($LASTEXITCODE -ne 0) { $failed = $true } else { Write-Host "  ok" -ForegroundColor Green }

    Write-Host ""
    Write-Host "[check] go vet ./..." -ForegroundColor Cyan
    & $go vet ./...
    if ($LASTEXITCODE -ne 0) { $failed = $true } else { Write-Host "  ok" -ForegroundColor Green }

    Write-Host ""
    if ($failed) {
        Write-Host "[check] FAILED - fix the above before pushing." -ForegroundColor Red
        exit 1
    }
    Write-Host "[check] all good - safe to push." -ForegroundColor Green
    exit 0
}

# ---------------------------------------------------------------------------
#  Cross-compile
# ---------------------------------------------------------------------------
Write-Host "[build] GOOS=linux GOARCH=$Arch CGO_ENABLED=0" -ForegroundColor Cyan

$env:GOOS        = "linux"
$env:GOARCH      = $Arch
$env:CGO_ENABLED = "0"

# -s -w strips the symbol table and DWARF info: smaller binary, no debug loss
# that matters for a service you run under systemd.
& $go build -trimpath -ldflags "-s -w" -o $out .
$code = $LASTEXITCODE

# Reset so a later plain `go build`/`go test` targets Windows again.
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue

if ($code -ne 0) {
    Write-Host "[build] FAILED" -ForegroundColor Red
    exit 1
}

$size = [math]::Round((Get-Item $out).Length / 1MB, 1)
Write-Host "[build] ok -> $out ($size MB, linux/$Arch)" -ForegroundColor Green
Write-Host ""
Write-Host "Copy it up and restart the service:" -ForegroundColor Yellow
Write-Host "  scp $out root@<server>:/usr/local/bin/vwireguard"
Write-Host "  ssh root@<server> 'chmod +x /usr/local/bin/vwireguard && systemctl restart vwireguard'"
