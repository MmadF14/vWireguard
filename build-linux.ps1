# Cross-compile vWireguard for a Linux server, from Windows.
#
#   powershell -ExecutionPolicy Bypass -File .\build-linux.ps1
#
# vWireguard has no cgo dependencies (no go-sqlite3), so a pure cross-compile
# works - you do NOT need a Linux box just to build.

$ErrorActionPreference = "Stop"
$out = "vwireguard"

Write-Host "[build] GOOS=linux GOARCH=amd64 CGO_ENABLED=0" -ForegroundColor Cyan

$env:GOOS        = "linux"
$env:GOARCH      = "amd64"
$env:CGO_ENABLED = "0"

# -s -w strips the symbol table and DWARF info: smaller binary, no debug loss
# that matters for a service you run under systemd.
go build -trimpath -ldflags "-s -w" -o $out .

if ($LASTEXITCODE -ne 0) { Write-Host "[build] FAILED" -ForegroundColor Red; exit 1 }

# Reset so a later plain `go build`/`go test` targets Windows again.
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue

$size = [math]::Round((Get-Item $out).Length / 1MB, 1)
Write-Host "[build] ok -> $out ($size MB, linux/amd64)" -ForegroundColor Green
Write-Host ""
Write-Host "Copy it up and restart the service:" -ForegroundColor Yellow
Write-Host "  scp $out root@<server>:/usr/local/bin/vwireguard"
Write-Host "  ssh root@<server> 'chmod +x /usr/local/bin/vwireguard && systemctl restart vwireguard'"
Write-Host ""
Write-Host "If your server is ARM (Oracle/Ampere/Hetzner ARM), set GOARCH=arm64 instead."
