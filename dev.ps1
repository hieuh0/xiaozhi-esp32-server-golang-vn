# Script chay moi truong dev local tren Windows (khong can Docker)
# Su dung: .\dev.ps1 [--sqlite] [--reset-db] [--backend-only] [--frontend-only] [--no-main-server]
param(
    [switch]$sqlite,         # Dung SQLite thay vi MySQL
    [switch]$resetDb,        # Reset database truoc khi chay
    [switch]$backendOnly,    # Chi chay backend (management, khong frontend)
    [switch]$frontendOnly,   # Chi chay frontend
    [switch]$mainServer,     # Chay them main xiaozhi server (can CGO + libopus + ONNX Runtime)
    [switch]$help
)

$ROOT = $PSScriptRoot
$BACKEND_DIR = Join-Path $ROOT "manager\backend"
$FRONTEND_DIR = Join-Path $ROOT "manager\frontend"
$MAIN_SERVER_DIR = $ROOT

if ($help) {
    Write-Host @"
Cach dung: .\dev.ps1 [tuy chon]

  (khong tham so)     Management (8080) + Frontend (3000)
  -sqlite             Dung SQLite thay vi MySQL (khong can cai MySQL)
  -resetDb            Reset toan bo database truoc khi chay
  -backendOnly        Chi chay management backend
  -frontendOnly       Chi chay frontend
  -mainServer         Chay them main xiaozhi server (8989)
                      Yeu cau: CGO, libopus, ONNX Runtime (Linux/WSL2)
  -help               Hien thi tro giup nay

Vi du:
  .\dev.ps1                        # Management + frontend (Windows OK)
  .\dev.ps1 -sqlite                # SQLite (khuyen nghi lan dau)
  .\dev.ps1 -sqlite -resetDb       # SQLite + reset DB
  .\dev.ps1 -frontendOnly          # Chi frontend
  .\dev.ps1 -sqlite -mainServer    # Day du (can Linux/WSL2)
"@
    exit 0
}

# -------------------------------------------------------------------
# Kiem tra cong cu
# -------------------------------------------------------------------
Write-Host "`n=== Kiem tra moi truong ===" -ForegroundColor Cyan

$goOk = $null -ne (Get-Command go -ErrorAction SilentlyContinue)
$nodeOk = $null -ne (Get-Command node -ErrorAction SilentlyContinue)
$npmOk = $null -ne (Get-Command npm -ErrorAction SilentlyContinue)

if (-not $goOk -and -not $frontendOnly) {
    Write-Host "[LOI] Go chua duoc cai dat. Tai tai: https://go.dev/dl/" -ForegroundColor Red
    exit 1
}
if (-not $nodeOk -and -not $backendOnly) {
    Write-Host "[LOI] Node.js chua duoc cai dat. Tai tai: https://nodejs.org/" -ForegroundColor Red
    exit 1
}
if ($goOk) { Write-Host "  [OK] Go:       $(go version)" -ForegroundColor Green }
if ($nodeOk) { Write-Host "  [OK] Node.js:  $(node --version)" -ForegroundColor Green }
if ($npmOk) { Write-Host "  [OK] npm:      $(npm --version)" -ForegroundColor Green }

# -------------------------------------------------------------------
# Chon file config backend
# -------------------------------------------------------------------
$configFile = "config\config.json"
$resetFlag = ""

if ($sqlite) {
    $sqliteConfig = Join-Path $BACKEND_DIR "config\config.sqlite.json"
    if (-not (Test-Path $sqliteConfig)) {
        Write-Host "`n[INFO] Tao file config SQLite: config\config.sqlite.json" -ForegroundColor Yellow
        $baseConfig = Get-Content (Join-Path $BACKEND_DIR "config\config.json") | ConvertFrom-Json
        $baseConfig.database.type = "sqlite"
        # Use WriteAllText to avoid UTF-8 BOM that PowerShell 5 adds with -Encoding UTF8
        [System.IO.File]::WriteAllText($sqliteConfig, ($baseConfig | ConvertTo-Json -Depth 10), [System.Text.Encoding]::UTF8)
    }
    $configFile = "config\config.sqlite.json"
    Write-Host "`n[INFO] Dung database SQLite: data\xiaozhi.db" -ForegroundColor Yellow
}

if ($resetDb) {
    $resetFlag = "-reset-db"
    Write-Host "[CANH BAO] Se reset toan bo database!" -ForegroundColor Red
    $confirm = Read-Host "Xac nhan reset? (y/N)"
    if ($confirm -ne "y" -and $confirm -ne "Y") {
        Write-Host "Da huy." -ForegroundColor Yellow
        exit 0
    }
}

# -------------------------------------------------------------------
# Cai dat npm dependencies neu chua co
# -------------------------------------------------------------------
if (-not $backendOnly) {
    $nodeModules = Join-Path $FRONTEND_DIR "node_modules"
    if (-not (Test-Path $nodeModules)) {
        Write-Host "`n=== Cai dat npm dependencies ===" -ForegroundColor Cyan
        Push-Location $FRONTEND_DIR
        npm install
        Pop-Location
        if ($LASTEXITCODE -ne 0) {
            Write-Host "[LOI] npm install that bai." -ForegroundColor Red
            exit 1
        }
    }
}

# -------------------------------------------------------------------
# Mo terminal rieng cho main xiaozhi server (port 8989) - chi khi co -mainServer
# -------------------------------------------------------------------
if (-not $frontendOnly -and $mainServer) {
    Write-Host "`n=== Khoi dong Main Xiaozhi Server (cong 8989) ===" -ForegroundColor Cyan

    $mainServerCmd = "cd '$MAIN_SERVER_DIR'; go run ./cmd/server/...; Write-Host '`n[Main Server da dung]' -ForegroundColor Red; Read-Host 'Nhan Enter de dong cua so'"

    Start-Process powershell -ArgumentList "-NoExit", "-Command", $mainServerCmd `
        -WorkingDirectory $MAIN_SERVER_DIR

    Write-Host "  [OK] Main server dang khoi dong trong cua so moi..." -ForegroundColor Green
    Start-Sleep -Seconds 2
}

# -------------------------------------------------------------------
# Mo terminal rieng cho management backend (port 8080)
# -------------------------------------------------------------------
if (-not $frontendOnly) {
    Write-Host "`n=== Khoi dong Management Backend (cong 8080) ===" -ForegroundColor Cyan

    $backendCmd = "cd '$BACKEND_DIR'; "
    if ($resetFlag) {
        $backendCmd += "go run main.go -config='$configFile' $resetFlag"
    } else {
        $backendCmd += "go run main.go -config='$configFile'"
    }
    $backendCmd += "; Write-Host '`n[Backend da dung]' -ForegroundColor Red; Read-Host 'Nhan Enter de dong cua so'"

    Start-Process powershell -ArgumentList "-NoExit", "-Command", $backendCmd `
        -WorkingDirectory $BACKEND_DIR

    Write-Host "  [OK] Management backend dang khoi dong trong cua so moi..." -ForegroundColor Green
    Start-Sleep -Seconds 2
}

# -------------------------------------------------------------------
# Mo terminal rieng cho frontend
# -------------------------------------------------------------------
if (-not $backendOnly) {
    Write-Host "`n=== Khoi dong Frontend (cong 3000) ===" -ForegroundColor Cyan

    $frontendCmd = "cd '$FRONTEND_DIR'; npm run dev; Write-Host '`n[Frontend da dung]' -ForegroundColor Red; Read-Host 'Nhan Enter de dong cua so'"

    Start-Process powershell -ArgumentList "-NoExit", "-Command", $frontendCmd `
        -WorkingDirectory $FRONTEND_DIR

    Write-Host "  [OK] Frontend dang khoi dong trong cua so moi..." -ForegroundColor Green
}

# -------------------------------------------------------------------
# Thong tin truy cap
# -------------------------------------------------------------------
$mainServerLine = if ($mainServer) { "  Main Server (WS):   ws://localhost:8989/xiaozhi/v1/`n" } else { "" }
Write-Host @"

=========================================================
  MOI TRUONG DEV DA KHOI DONG
=========================================================
  Frontend (UI):      http://localhost:3000
  Management (API):   http://localhost:8080
$($mainServerLine)  API Docs:           http://localhost:3000/openapi-docs
---------------------------------------------------------
  De dung: dong cac cua so PowerShell tuong ung
=========================================================
"@ -ForegroundColor Cyan
