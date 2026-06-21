#!/usr/bin/env bash
# =============================================================
#  WSL Dev Runner — chay full stack tren WSL2
#  Su dung:
#    bash wsl-dev.sh            # backend + frontend (SQLite)
#    bash wsl-dev.sh supertonic # them main server voi CGO TTS
#    bash wsl-dev.sh reset      # reset DB truoc khi chay
#    bash wsl-dev.sh help
# =============================================================
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$ROOT/manager/backend"
FRONTEND_DIR="$ROOT/manager/frontend"
CONFIG_FILE="config/config.sqlite.json"
SESSION="xiaozhi"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[OK]${NC} $*"; }
info() { echo -e "${CYAN}[>>] $*${NC}"; }
warn() { echo -e "${YELLOW}[!!]${NC} $*"; }

WIN_LAN_IP=""

# Cau hinh Windows portproxy de phone/LAN co the truy cap dich vu qua Windows IP
setup_win_portproxy() {
  local wsl_ip="$1"
  command -v powershell.exe &>/dev/null || { warn "powershell.exe khong tim thay — bo qua LAN portproxy"; return; }

  local ports=(8080 3000)
  [ "$MAIN_SERVER" = true ] && ports+=(8989 1883)

  # Kiem tra quyen Admin
  local is_admin
  is_admin=$(powershell.exe -NoProfile -NonInteractive -Command \
    "([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole('Administrator')" \
    2>/dev/null | tr -d '\r\n ') || true

  if [ "$is_admin" != "True" ]; then
    warn "Portproxy can quyen Admin — mo PowerShell Admin va chay lenh sau:"
    for p in "${ports[@]}"; do
      warn "  netsh interface portproxy add v4tov4 listenport=$p listenaddress=0.0.0.0 connectport=$p connectaddress=$wsl_ip"
      warn "  netsh advfirewall firewall add rule name='WSL-$p' dir=in action=allow protocol=TCP localport=$p"
    done
    return
  fi

  for p in "${ports[@]}"; do
    powershell.exe -NoProfile -NonInteractive -Command \
      "netsh interface portproxy delete v4tov4 listenport=$p listenaddress=0.0.0.0; netsh interface portproxy add v4tov4 listenport=$p listenaddress=0.0.0.0 connectport=$p connectaddress=$wsl_ip" \
      > /dev/null 2>&1 || true
    powershell.exe -NoProfile -NonInteractive -Command \
      "netsh advfirewall firewall delete rule name='WSL-$p'; netsh advfirewall firewall add rule name='WSL-$p' dir=in action=allow protocol=TCP localport=$p" \
      > /dev/null 2>&1 || true
  done

  WIN_LAN_IP=$(powershell.exe -NoProfile -NonInteractive -Command \
    "(Get-NetIPAddress -AddressFamily IPv4 | Where-Object { \$_.IPAddress -notlike '127.*' -and \$_.IPAddress -notlike '172.*' -and \$_.IPAddress -notlike '169.*' } | Select-Object -First 1).IPAddress" \
    2>/dev/null | tr -d '\r\n ') || true

  ok "LAN portproxy OK — phone/LAN dung: ${WIN_LAN_IP:-<WIN-IP>} | cong: ${ports[*]}"
}

MAIN_SERVER=false
RESET_DB=""

for arg in "$@"; do
  case "$arg" in
    supertonic) MAIN_SERVER=true ;;
    reset)      RESET_DB="-reset-db" ;;
    help|-h)
      echo "Su dung: bash wsl-dev.sh [tuy-chon]"
      echo ""
      echo "  (khong tham so)   Management backend (8080) + Frontend (3000)"
      echo "  supertonic        Them main server (8989) voi CGO + supertonic TTS"
      echo "  reset             Reset database truoc khi chay"
      echo ""
      echo "  MQTT built-in:    main server tu dong khoi dong MQTT broker (1883)"
      echo "  WebSocket:        ws://localhost:8989/xiaozhi/v1/"
      echo ""
      echo "Yeu cau cho 'supertonic': da chay wsl-setup.sh thanh cong"
      exit 0 ;;
  esac
done

# --- Kiem tra cong cu ---
echo ""
echo "============================================================"
echo "  Xiaozhi Dev Stack — WSL2"
echo "============================================================"

command -v go   &>/dev/null || { echo -e "${RED}[ERR]${NC} Go chua cai — chay: bash wsl-setup.sh"; exit 1; }
command -v node &>/dev/null || { echo -e "${RED}[ERR]${NC} Node.js chua cai — chay: bash wsl-setup.sh"; exit 1; }
command -v tmux &>/dev/null || { echo -e "${RED}[ERR]${NC} tmux chua cai — chay: sudo apt-get install tmux"; exit 1; }

ok "Go:     $(go version | awk '{print $3}')"
ok "Node:   $(node --version)"

# --- Kiem tra CGO neu can main server ---
if [ "$MAIN_SERVER" = true ]; then
  export CGO_ENABLED=1
  export ONNXRUNTIME_LIB_PATH="${ONNXRUNTIME_LIB_PATH:-/usr/local/lib/libonnxruntime.so}"
  export LD_LIBRARY_PATH="$ROOT/lib/ten-vad/lib/Linux/x64:${LD_LIBRARY_PATH}"

  if [ ! -f "$ONNXRUNTIME_LIB_PATH" ]; then
    warn "ONNX Runtime khong tim thay tai: $ONNXRUNTIME_LIB_PATH"
    warn "Chay wsl-setup.sh truoc, hoac: export ONNXRUNTIME_LIB_PATH=<duong-dan>"
    exit 1
  fi
  ok "ORT:    $ONNXRUNTIME_LIB_PATH"

  command -v gcc &>/dev/null || { echo -e "${RED}[ERR]${NC} gcc chua cai — chay: sudo apt-get install build-essential"; exit 1; }
  ok "GCC:    $(gcc --version | head -1)"
fi

# --- Tao config SQLite neu chua co ---
if [ ! -f "$BACKEND_DIR/$CONFIG_FILE" ]; then
  info "Tao config SQLite..."
  node -e "
    const fs = require('fs');
    const src = '$BACKEND_DIR/config/config.json';
    const dst = '$BACKEND_DIR/$CONFIG_FILE';
    const cfg = JSON.parse(fs.readFileSync(src, 'utf8'));
    cfg.database.type = 'sqlite';
    fs.writeFileSync(dst, JSON.stringify(cfg, null, 2));
    console.log('Tao: ' + dst);
  "
fi

# --- Cai npm neu chua co ---
if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
  info "Cai npm dependencies..."
  cd "$FRONTEND_DIR" && npm install --silent
  cd "$ROOT"
fi

# --- Kill session cu neu con ton tai ---
tmux kill-session -t "$SESSION" 2>/dev/null && warn "Da dong session tmux cu: $SESSION" || true

# --- Tao tmux session moi ---
tmux new-session -d -s "$SESSION" -x 220 -y 50

# Panel 0 — Management Backend (cong 8080)
info "Khoi dong Management Backend (cong 8080)..."
tmux rename-window -t "$SESSION:0" "backend"
tmux send-keys -t "$SESSION:0" \
  "cd '$BACKEND_DIR' && echo '[backend] Khoi dong...' && go run main.go -config='$CONFIG_FILE' $RESET_DB" Enter

sleep 1

# Panel 1 — Frontend (cong 3000)
info "Khoi dong Frontend (cong 3000)..."
tmux new-window -t "$SESSION" -n "frontend"
tmux send-keys -t "$SESSION:frontend" \
  "cd '$FRONTEND_DIR' && echo '[frontend] Khoi dong...' && npm run dev" Enter

# Panel 2 — Main Server (cong 8989, 1883) — chi khi co flag supertonic
if [ "$MAIN_SERVER" = true ]; then
  info "Khoi dong Main Server (cong 8989, MQTT 1883)..."
  tmux new-window -t "$SESSION" -n "main-server"
  tmux send-keys -t "$SESSION:main-server" \
    "cd '$ROOT' && echo '[main-server] Build + chay voi -tags supertonic...' && CGO_ENABLED=1 ONNXRUNTIME_LIB_PATH='$ONNXRUNTIME_LIB_PATH' LD_LIBRARY_PATH='$ROOT/lib/ten-vad/lib/Linux/x64:$LD_LIBRARY_PATH' go run -tags supertonic ./cmd/server/..." Enter
fi

WSL_IP=$(ip addr show eth0 2>/dev/null | grep 'inet ' | awk '{print $2}' | cut -d/ -f1)
WSL_IP="${WSL_IP:-$(hostname -I | awk '{print $1}')}"

# Cau hinh Windows portproxy cho phone/LAN access
info "Cau hinh Windows portproxy cho truy cap LAN..."
setup_win_portproxy "$WSL_IP"

# Tu dong cap nhat ota.test.websocket.url voi IP hien tai
OTA_IP="${WIN_LAN_IP:-$WSL_IP}"
OTA_CONFIG="$ROOT/config/config.yaml"
if [ -n "$OTA_IP" ] && [ -f "$OTA_CONFIG" ]; then
  sed -i "s|url: \"ws://[^\"]*:8989/xiaozhi/v1/\"|url: \"ws://$OTA_IP:8989/xiaozhi/v1/\"|" "$OTA_CONFIG"
  ok "ota.test.url → ws://$OTA_IP:8989/xiaozhi/v1/"
fi

# Panel logs — hien thi URL truy cap va huong dan
tmux new-window -t "$SESSION" -n "logs"
tmux send-keys -t "$SESSION:logs" "clear" Enter
tmux send-keys -t "$SESSION:logs" "echo '============================================================'" Enter
tmux send-keys -t "$SESSION:logs" "echo '  XIAOZHI DEV STACK — DANG CHAY'" Enter
tmux send-keys -t "$SESSION:logs" "echo '============================================================'" Enter
tmux send-keys -t "$SESSION:logs" "echo '  Management API :  http://localhost:8080'" Enter
tmux send-keys -t "$SESSION:logs" "echo '  Frontend (WSL) :  http://localhost:3000'" Enter
tmux send-keys -t "$SESSION:logs" "echo '  Frontend (Win) :  http://$WSL_IP:3000   <-- mo tren Windows'" Enter
if [ "$MAIN_SERVER" = true ]; then
tmux send-keys -t "$SESSION:logs" "echo '  Main Server WS :  ws://localhost:8989/xiaozhi/v1/'" Enter
tmux send-keys -t "$SESSION:logs" "echo '  MQTT Broker    :  mqtt://localhost:1883'" Enter
fi
if [ -n "$WIN_LAN_IP" ]; then
tmux send-keys -t "$SESSION:logs" "echo '------------------------------------------------------------'" Enter
tmux send-keys -t "$SESSION:logs" "echo '  [LAN] API      :  http://$WIN_LAN_IP:8080  <-- phone/LAN'" Enter
tmux send-keys -t "$SESSION:logs" "echo '  [LAN] Frontend :  http://$WIN_LAN_IP:3000  <-- phone/LAN'" Enter
fi
tmux send-keys -t "$SESSION:logs" "echo '------------------------------------------------------------'" Enter
tmux send-keys -t "$SESSION:logs" "echo '  Ctrl+b n/p : doi window  |  Ctrl+b d : tach ra'" Enter
tmux send-keys -t "$SESSION:logs" "echo '============================================================'" Enter

# Quay lai backend window
tmux select-window -t "$SESSION:backend"

echo ""
echo "============================================================"
echo -e "  ${GREEN}STACK DA KHOI DONG${NC} — tmux session: ${CYAN}$SESSION${NC}"
echo "============================================================"
echo "  Management API:  http://localhost:8080"
echo "  Frontend (UI):   http://localhost:3000  (trong WSL)"
echo -e "  Frontend (Win):  ${CYAN}http://$WSL_IP:3000${NC}  (tu Windows)"
if [ "$MAIN_SERVER" = true ]; then
echo "  Main Server WS:  ws://localhost:8989/xiaozhi/v1/"
echo "  MQTT Broker:     mqtt://localhost:1883"
fi
if [ -n "$WIN_LAN_IP" ]; then
echo "------------------------------------------------------------"
echo -e "  Phone/LAN API:   ${CYAN}http://$WIN_LAN_IP:8080${NC}  (dien thoai/LAN)"
echo -e "  Phone/LAN UI:    ${CYAN}http://$WIN_LAN_IP:3000${NC}  (dien thoai/LAN)"
fi
echo "------------------------------------------------------------"
echo "  Attach vao tmux:  tmux attach -t $SESSION"
echo "  Doi window:       Ctrl+b  n/p  hoac  Ctrl+b  w"
echo "  Tach ra:          Ctrl+b  d"
echo "  Dung tat ca:      tmux kill-session -t $SESSION"
echo "============================================================"
echo ""
echo "  Dang attach vao tmux (logs window)..."
sleep 2
exec tmux attach -t "$SESSION:logs"
