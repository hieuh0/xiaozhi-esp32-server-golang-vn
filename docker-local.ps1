<#
.SYNOPSIS
    One-command Docker local setup for xiaozhi-esp32-server.
    Detects LAN IP, patches OTA WebSocket URL, opens firewall, starts containers.

.PARAMETER IP
    Override auto-detected LAN IP.

.PARAMETER Down
    Run docker compose down and exit (no config patching).

.PARAMETER Logs
    Tail docker compose logs after containers start.

.PARAMETER Help
    Show this help.

.EXAMPLE
    .\docker-local.ps1
    .\docker-local.ps1 -IP "192.168.1.50"
    .\docker-local.ps1 -Down
#>
param(
    [string]$IP,
    [switch]$Down,
    [switch]$Logs,
    [switch]$Help
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ROOT                = $PSScriptRoot
$COMPOSE_DIR         = Join-Path $ROOT "docker\docker-composer"
$COMPOSE_FILE        = "docker-compose.local.yml"   # build from source; switch to docker-compose.yml when images are published
$CONFIG_PATH         = Join-Path $ROOT "config\config.yaml"
$PERSONAL_CONFIG     = Join-Path $ROOT "config\config.personal.yaml"
$BACKUP_PATH         = "$PERSONAL_CONFIG.bak"

if ($Help) {
    Get-Help $MyInvocation.MyCommand.Path
    exit 0
}

# ─── [0] Early exit for -Down ────────────────────────────────────────────────
if ($Down) {
    Write-Host "`n=== Dung tat ca containers ===" -ForegroundColor Cyan
    Push-Location $COMPOSE_DIR
    try { docker compose -f $COMPOSE_FILE down } finally { Pop-Location }
    Write-Host "[OK] Containers da dung." -ForegroundColor Green
    exit 0
}

Write-Host "`n=== Xiaozhi Docker Local Setup ===" -ForegroundColor Cyan

# ─── [1] Check prerequisites ─────────────────────────────────────────────────
Write-Host "`n[1/7] Kiem tra prerequisites..." -ForegroundColor White

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "[LOI] Docker chua duoc cai dat. Tai tai: https://docs.docker.com/desktop/windows/" -ForegroundColor Red
    exit 1
}

docker compose version 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "[LOI] Docker Compose plugin chua co. Cap nhat Docker Desktop." -ForegroundColor Red
    exit 1
}

docker info 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "[LOI] Docker Desktop chua chay. Mo Docker Desktop roi thu lai." -ForegroundColor Red
    exit 1
}
Write-Host "  [OK] Docker dang chay." -ForegroundColor Green

# ─── [2] Build mode info (no registry login needed) ─────────────────────────
Write-Host "`n[2/7] Che do: build tu source ($COMPOSE_FILE)..." -ForegroundColor White
Write-Host "  [OK] Khong can dang nhap ghcr.io." -ForegroundColor Green

# ─── [3] Detect LAN IP ───────────────────────────────────────────────────────
Write-Host "`n[3/7] Xac dinh LAN IP..." -ForegroundColor White

function Get-LanIP {
    $vpnKeywords = 'VPN|TAP|tun|Cisco|OpenVPN|WireGuard|Hyper-V|vEthernet|Loopback'
    $privateRanges = '^(192\.168\.|10\.|172\.(1[6-9]|2\d|3[01])\.)'

    $adapters = Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
        Where-Object {
            $_.IPAddress -match $privateRanges -and
            $_.PrefixOrigin -ne 'WellKnown'
        } |
        Where-Object {
            $adapter = Get-NetAdapter -InterfaceIndex $_.InterfaceIndex -ErrorAction SilentlyContinue
            $adapter -and ($adapter.InterfaceDescription -notmatch $vpnKeywords) -and
            $adapter.Status -eq 'Up'
        }

    # Priority: WiFi
    $wifi = $adapters | Where-Object {
        $adapter = Get-NetAdapter -InterfaceIndex $_.InterfaceIndex -ErrorAction SilentlyContinue
        $adapter -and $adapter.InterfaceDescription -match 'Wi.?Fi|Wireless|WLAN'
    } | Select-Object -First 1

    if ($wifi) { return $wifi.IPAddress }

    # Fallback: Ethernet
    $eth = $adapters | Select-Object -First 1
    if ($eth) { return $eth.IPAddress }

    return $null
}

if ($IP) {
    $lanIP = $IP
    Write-Host "  [OK] IP manual override: $lanIP" -ForegroundColor Green
} else {
    $lanIP = Get-LanIP
    if (-not $lanIP) {
        Write-Host "[LOI] Khong the tu dong xac dinh LAN IP." -ForegroundColor Red
        Write-Host "      Chay lai voi flag: .\docker-local.ps1 -IP '192.168.x.x'" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "  [OK] LAN IP: $lanIP" -ForegroundColor Green
}

# ─── [4] Patch config/config.personal.yaml ───────────────────────────────────
Write-Host "`n[4/7] Patch config/config.personal.yaml..." -ForegroundColor White

if (-not (Test-Path $CONFIG_PATH)) {
    Write-Host "[LOI] Khong tim thay: $CONFIG_PATH" -ForegroundColor Red
    exit 1
}

# Create config.personal.yaml from config.yaml if it doesn't exist
if (-not (Test-Path $PERSONAL_CONFIG)) {
    Copy-Item $CONFIG_PATH $PERSONAL_CONFIG -Force
    Write-Host "  [OK] Tao config.personal.yaml tu config.yaml" -ForegroundColor Gray
}

# Backup personal config before patching
Copy-Item $PERSONAL_CONFIG $BACKUP_PATH -Force
Write-Host "  [OK] Backup: config.personal.yaml.bak" -ForegroundColor Gray

$content = Get-Content $PERSONAL_CONFIG -Raw
$newUrl  = "ws://${lanIP}:8989/xiaozhi/v1/"

# Replace only the url under ota.test.websocket using lazy dot-all regex
$patched = $content -replace '(?s)(ota:.*?test:.*?websocket:.*?url:\s*")[^"]*(")', "`${1}${newUrl}`${2}"

if ($patched -eq $content) {
    Write-Host "[CANH BAO] Khong tim thay ota.test.websocket.url — kiem tra lai config.personal.yaml." -ForegroundColor Yellow
} else {
    [System.IO.File]::WriteAllText($PERSONAL_CONFIG, $patched, [System.Text.Encoding]::UTF8)
    Write-Host "  [OK] OTA WebSocket URL -> $newUrl" -ForegroundColor Green
}

# ─── [5] Firewall rules (Admin only) ─────────────────────────────────────────
Write-Host "`n[5/7] Cau hinh Windows Firewall..." -ForegroundColor White

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)

if ($isAdmin) {
    $rules = @(
        @{ Name="Xiaozhi-WS-8989";   Protocol="TCP"; Port=8989 },
        @{ Name="Xiaozhi-UI-8080";   Protocol="TCP"; Port=8080 },
        @{ Name="Xiaozhi-API-8081";  Protocol="TCP"; Port=8081 },
        @{ Name="Xiaozhi-MQTT-2883"; Protocol="TCP"; Port=2883 },
        @{ Name="Xiaozhi-UDP-8888";  Protocol="UDP"; Port=8888 }
    )
    foreach ($r in $rules) {
        $exists = Get-NetFirewallRule -DisplayName $r.Name -ErrorAction SilentlyContinue
        if (-not $exists) {
            New-NetFirewallRule -DisplayName $r.Name -Direction Inbound -Action Allow `
                -Protocol $r.Protocol -LocalPort $r.Port | Out-Null
            Write-Host "  [FW] Added : $($r.Name)" -ForegroundColor Green
        } else {
            Write-Host "  [FW] Exists: $($r.Name)" -ForegroundColor Gray
        }
    }
} else {
    Write-Host "  [CANH BAO] Khong co quyen Admin - bo qua firewall." -ForegroundColor Yellow
    Write-Host "             Chay lai voi 'Run as Administrator' de mo firewall tu dong." -ForegroundColor Yellow
}

# ─── [6] Docker compose build + up ──────────────────────────────────────────
Write-Host "`n[6/7] Build va khoi dong Docker containers..." -ForegroundColor White
Write-Host "  [INFO] Building images tu source (lan dau co the mat 5-15 phut)..." -ForegroundColor Yellow
Write-Host "  [INFO] Kiem tra Docker disk space truoc khi build..." -ForegroundColor Yellow
docker system df
Write-Host "  [TIP] Neu disk thap, chay: docker system prune -af" -ForegroundColor DarkGray

# Build sequentially to avoid parallel disk exhaustion (disk full → go.sum I/O error → BuildKit EOF)
Push-Location $COMPOSE_DIR
try {
    Write-Host "  [BUILD 1/3] backend..." -ForegroundColor Cyan
    docker compose -f $COMPOSE_FILE build backend
    if ($LASTEXITCODE -ne 0) { Write-Host "[LOI] Build backend that bai." -ForegroundColor Red; exit 1 }

    Write-Host "  [BUILD 2/3] main-server..." -ForegroundColor Cyan
    docker compose -f $COMPOSE_FILE build main-server
    if ($LASTEXITCODE -ne 0) { Write-Host "[LOI] Build main-server that bai." -ForegroundColor Red; exit 1 }

    Write-Host "  [BUILD 3/3] frontend..." -ForegroundColor Cyan
    docker compose -f $COMPOSE_FILE build frontend
    if ($LASTEXITCODE -ne 0) { Write-Host "[LOI] Build frontend that bai." -ForegroundColor Red; exit 1 }

    docker compose -f $COMPOSE_FILE up -d
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[LOI] docker compose up that bai." -ForegroundColor Red
        exit 1
    }
} finally {
    Pop-Location
}

Write-Host "  [OK] Containers da khoi dong." -ForegroundColor Green

# ─── [7] Print summary ────────────────────────────────────────────────────────
Write-Host "`n[7/7] Ket qua:" -ForegroundColor White

$summary = @"

=========================================================
  DOCKER LOCAL DA KHOI DONG
=========================================================
  Phone (UI):         http://${lanIP}:8080
  Management API:     http://${lanIP}:8081
  ESP32 WebSocket:    ws://${lanIP}:8989/xiaozhi/v1/
  MQTT Broker:        ${lanIP}:2883
---------------------------------------------------------
  Tat containers: .\docker-local.ps1 -Down
  Rollback IP:    copy config\config.personal.yaml.bak config\config.personal.yaml
=========================================================
"@

Write-Host $summary -ForegroundColor Cyan

if ($Logs) {
    Write-Host "`n[INFO] Hien logs (Ctrl+C de thoat)..." -ForegroundColor Yellow
    Push-Location $COMPOSE_DIR
    try { docker compose -f $COMPOSE_FILE logs -f } finally { Pop-Location }
}
