#!/usr/bin/env bash
#
# Cross-platform Docker local setup for xiaozhi-esp32-server.
# Linux/macOS: run directly.  Windows: run via Git Bash.
#
# Usage:
#   ./docker-local.sh                   # Auto-detect LAN IP, start containers
#   ./docker-local.sh --ip 192.168.1.50 # Manual IP override
#   ./docker-local.sh --down            # Stop containers
#   ./docker-local.sh --logs            # Tail logs after start
#   ./docker-local.sh --help

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_PATH="$SCRIPT_DIR/config/config.yaml"
PERSONAL_CONFIG="$SCRIPT_DIR/config/config.personal.yaml"
BACKUP_PATH="$PERSONAL_CONFIG.bak"
COMPOSE_DIR="$SCRIPT_DIR/docker/docker-composer"
COMPOSE_FILE="docker-compose.local.yml"  # build from source; switch to docker-compose.yml when images are published

# ─── Detect OS ───────────────────────────────────────────────────────────────
_OS=$(uname -s 2>/dev/null || echo "Unknown")
case "$_OS" in
    MINGW*|MSYS*|CYGWIN*) OS_TYPE="windows" ;;
    Darwin)                OS_TYPE="macos"   ;;
    Linux)                 OS_TYPE="linux"   ;;
    *)                     OS_TYPE="unknown" ;;
esac

# ─── Parse args ──────────────────────────────────────────────────────────────
DOWN=false
LOGS=false
MANUAL_IP=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --down|-d)  DOWN=true ;;
        --logs|-l)  LOGS=true ;;
        --ip)       MANUAL_IP="${2:-}"; shift ;;
        --help|-h)
            sed -n '2,8p' "$0"   # Print the usage comment block
            exit 0
            ;;
        *) echo "[LOI] Unknown argument: $1. Dung --help de xem cach dung."; exit 1 ;;
    esac
    shift
done

# ─── [0] Early exit for --down ───────────────────────────────────────────────
if [[ "$DOWN" == true ]]; then
    echo ""
    echo "=== Dung tat ca containers ==="
    (cd "$COMPOSE_DIR" && docker compose -f "$COMPOSE_FILE" down)
    echo "[OK] Containers da dung."
    exit 0
fi

echo ""
echo "=== Xiaozhi Docker Local Setup (bash / $OS_TYPE) ==="

# ─── [1] Check prerequisites ─────────────────────────────────────────────────
echo ""
echo "[1/7] Kiem tra prerequisites..."

if ! command -v docker &>/dev/null; then
    echo "[LOI] Docker chua duoc cai dat."
    case "$OS_TYPE" in
        linux)   echo "      https://docs.docker.com/engine/install/" ;;
        macos)   echo "      https://docs.docker.com/desktop/mac/" ;;
        windows) echo "      https://docs.docker.com/desktop/windows/" ;;
    esac
    exit 1
fi

if ! docker compose version &>/dev/null 2>&1; then
    echo "[LOI] Docker Compose plugin chua co. Cap nhat Docker Desktop."
    exit 1
fi

if ! docker info &>/dev/null 2>&1; then
    echo "[LOI] Docker daemon chua chay."
    [[ "$OS_TYPE" != "linux" ]] && echo "      Mo Docker Desktop roi thu lai."
    exit 1
fi
echo "  [OK] Docker dang chay."

# ─── [2] Build mode info (no registry login needed) ─────────────────────────
echo ""
echo "[2/7] Che do: build tu source ($COMPOSE_FILE)..."
echo "  [OK] Khong can dang nhap ghcr.io."

# ─── [3] Detect LAN IP ───────────────────────────────────────────────────────
echo ""
echo "[3/7] Xac dinh LAN IP..."

_get_lan_ip_linux() {
    local ip=""
    if command -v ip &>/dev/null; then
        ip=$(ip route get 1.1.1.1 2>/dev/null \
            | awk 'NR==1 {for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}')
    fi
    # Fallback: first non-loopback address
    if [[ -z "$ip" ]] && command -v hostname &>/dev/null; then
        ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
    echo "$ip"
}

_get_lan_ip_macos() {
    local ip="" iface=""
    # Use the interface for the default route
    iface=$(route get 1.1.1.1 2>/dev/null | awk '/interface:/{print $2}')
    if [[ -n "$iface" ]]; then
        ip=$(ipconfig getifaddr "$iface" 2>/dev/null || true)
    fi
    # Fallback: common Wi-Fi/Ethernet interfaces
    if [[ -z "$ip" ]]; then
        for iface in en0 en1 en2 en3; do
            ip=$(ipconfig getifaddr "$iface" 2>/dev/null || true)
            [[ -n "$ip" ]] && break
        done
    fi
    echo "$ip"
}

_get_lan_ip_windows() {
    # Delegate to PowerShell for VPN-filtered, WiFi-prioritised detection
    powershell.exe -NoProfile -Command "
\$vpn = 'VPN|TAP|tun|Cisco|OpenVPN|WireGuard|Hyper-V|vEthernet|Loopback'
\$addrs = Get-NetIPAddress -AddressFamily IPv4 |
    Where-Object {
        \$_.IPAddress -match '^(192\.168\.|10\.|172\.(1[6-9]|2\d|3[01])\.)' -and
        \$_.PrefixOrigin -ne 'WellKnown'
    } |
    Where-Object {
        \$adapter = Get-NetAdapter -InterfaceIndex \$_.InterfaceIndex -ErrorAction SilentlyContinue
        \$adapter -and \$adapter.Status -eq 'Up' -and \$adapter.InterfaceDescription -notmatch \$vpn
    }
\$wifi = \$addrs | Where-Object {
    (Get-NetAdapter -InterfaceIndex \$_.InterfaceIndex).InterfaceDescription -match 'Wi.?Fi|Wireless|WLAN'
} | Select-Object -First 1
if (\$wifi) { \$wifi.IPAddress } else { (\$addrs | Select-Object -First 1).IPAddress }
" 2>/dev/null | tr -d '\r\n'
}

if [[ -n "$MANUAL_IP" ]]; then
    LAN_IP="$MANUAL_IP"
    echo "  [OK] IP manual override: $LAN_IP"
else
    case "$OS_TYPE" in
        linux)   LAN_IP=$(_get_lan_ip_linux) ;;
        macos)   LAN_IP=$(_get_lan_ip_macos) ;;
        windows) LAN_IP=$(_get_lan_ip_windows) ;;
        *)       LAN_IP="" ;;
    esac

    if [[ -z "$LAN_IP" ]]; then
        echo "[LOI] Khong the tu dong xac dinh LAN IP."
        echo "      Chay lai voi: ./docker-local.sh --ip 192.168.x.x"
        exit 1
    fi
    echo "  [OK] LAN IP: $LAN_IP"
fi

# ─── [4] Patch config/config.personal.yaml ───────────────────────────────────
echo ""
echo "[4/7] Patch config/config.personal.yaml..."

if [[ ! -f "$CONFIG_PATH" ]]; then
    echo "[LOI] Khong tim thay: $CONFIG_PATH"
    exit 1
fi

# Create config.personal.yaml from config.yaml if it doesn't exist
if [[ ! -f "$PERSONAL_CONFIG" ]]; then
    cp "$CONFIG_PATH" "$PERSONAL_CONFIG"
    echo "  [OK] Tao config.personal.yaml tu config.yaml"
fi

cp "$PERSONAL_CONFIG" "$BACKUP_PATH"
echo "  [OK] Backup: config.personal.yaml.bak"

NEW_URL="ws://${LAN_IP}:8989/xiaozhi/v1/"
export NEW_URL PERSONAL_CONFIG

# perl: slurp entire file (-0), dot-matches-newline (s flag), use $ENV to avoid shell quoting
if command -v perl &>/dev/null; then
    perl -0pe \
        's|(ota:.*?test:.*?websocket:.*?url:\s*")[^"]*"|${1}$ENV{NEW_URL}"|s' \
        "$PERSONAL_CONFIG" > "${PERSONAL_CONFIG}.tmp"
    mv "${PERSONAL_CONFIG}.tmp" "$PERSONAL_CONFIG"
elif command -v python3 &>/dev/null; then
    python3 - <<'PYEOF'
import re, os
path    = os.environ['PERSONAL_CONFIG']
new_url = os.environ['NEW_URL']
with open(path) as f:
    content = f.read()
patched = re.sub(
    r'(ota:.*?test:.*?websocket:.*?url:\s*")[^"]*"',
    lambda m: m.group(1) + new_url + '"',
    content, flags=re.DOTALL
)
with open(path, 'w') as f:
    f.write(patched)
PYEOF
else
    echo "[LOI] Can perl hoac python3 de patch config.personal.yaml."
    exit 1
fi

echo "  [OK] OTA WebSocket URL -> $NEW_URL"

# ─── [5] Firewall ────────────────────────────────────────────────────────────
echo ""
echo "[5/7] Cau hinh firewall..."

if [[ "$OS_TYPE" == "windows" ]]; then
    # Delegate to PowerShell — idempotent, Admin-aware
    powershell.exe -NoProfile -Command "
\$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not \$isAdmin) {
    Write-Host '  [CANH BAO] Khong co quyen Admin - bo qua firewall.' -ForegroundColor Yellow
    Write-Host '             Chay lai Git Bash as Administrator de mo firewall tu dong.' -ForegroundColor Yellow
    exit 0
}
\$rules = @(
    @{Name='Xiaozhi-WS-8989';   Protocol='TCP'; Port=8989},
    @{Name='Xiaozhi-UI-8080';   Protocol='TCP'; Port=8080},
    @{Name='Xiaozhi-API-8081';  Protocol='TCP'; Port=8081},
    @{Name='Xiaozhi-MQTT-2883'; Protocol='TCP'; Port=2883},
    @{Name='Xiaozhi-UDP-8888';  Protocol='UDP'; Port=8888}
)
foreach (\$r in \$rules) {
    \$exists = Get-NetFirewallRule -DisplayName \$r.Name -ErrorAction SilentlyContinue
    if (-not \$exists) {
        New-NetFirewallRule -DisplayName \$r.Name -Direction Inbound -Action Allow \`
            -Protocol \$r.Protocol -LocalPort \$r.Port | Out-Null
        Write-Host \"  [FW] Added : \$(\$r.Name)\" -ForegroundColor Green
    } else {
        Write-Host \"  [FW] Exists: \$(\$r.Name)\" -ForegroundColor DarkGray
    }
}
" 2>/dev/null || true
else
    echo "  [INFO] Linux/macOS: Docker tu quan ly network rules."
    echo "         Neu can mo port thu cong:"
    echo "           Linux:  sudo ufw allow 8080,8081,8989,2883/tcp && sudo ufw allow 8888/udp"
    echo "           macOS:  khong can them firewall rule voi Docker Desktop"
fi

# ─── [6] Docker compose build + up ──────────────────────────────────────────
echo ""
echo "[6/7] Build va khoi dong Docker containers..."
echo "  [INFO] Building images tu source (lan dau co the mat 5-15 phut)..."
echo "  [INFO] Kiem tra Docker disk space truoc khi build..."
docker system df
echo ""
echo "  [TIP] Neu disk thap, chay: docker system prune -af"
echo ""

# Build sequentially to avoid parallel disk exhaustion (disk full → go.sum I/O error → BuildKit EOF)
(
    cd "$COMPOSE_DIR"
    echo "  [BUILD 1/3] backend..."
    docker compose -f "$COMPOSE_FILE" build backend
    echo "  [BUILD 2/3] main-server..."
    docker compose -f "$COMPOSE_FILE" build main-server
    echo "  [BUILD 3/3] frontend..."
    docker compose -f "$COMPOSE_FILE" build frontend
    docker compose -f "$COMPOSE_FILE" up -d
)

echo "  [OK] Containers da khoi dong."

# ─── [7] Summary ─────────────────────────────────────────────────────────────
echo ""
echo "[7/7] Ket qua:"
cat <<SUMMARY

=========================================================
  DOCKER LOCAL DA KHOI DONG
=========================================================
  Phone (UI):         http://${LAN_IP}:8080
  Management API:     http://${LAN_IP}:8081
  ESP32 WebSocket:    ws://${LAN_IP}:8989/xiaozhi/v1/
  MQTT Broker:        ${LAN_IP}:2883
---------------------------------------------------------
  Tat containers: ./docker-local.sh --down
  Rollback IP:    cp config/config.personal.yaml.bak config/config.personal.yaml
=========================================================
SUMMARY

if [[ "$LOGS" == true ]]; then
    echo "[INFO] Hien logs (Ctrl+C de thoat)..."
    (cd "$COMPOSE_DIR" && docker compose -f "$COMPOSE_FILE" logs -f)
fi
