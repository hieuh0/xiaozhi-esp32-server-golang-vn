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
    "cd '$ROOT' && echo '[main-server] Build + chay voi -tags supertonic...' && CGO_ENABLED=1 ONNXRUNTIME_LIB_PATH='$ONNXRUNTIME_LIB_PATH' go run -tags supertonic ./cmd/server/..." Enter
fi

# Panel logs — tail logs neu co
tmux new-window -t "$SESSION" -n "logs"
tmux send-keys -t "$SESSION:logs" \
  "echo 'Panel nay de xem logs. Dung: tmux select-window -t backend/frontend/main-server'" Enter

# Quay lai backend window
tmux select-window -t "$SESSION:backend"

echo ""
echo "============================================================"
echo -e "  ${GREEN}STACK DA KHOI DONG${NC} — tmux session: ${CYAN}$SESSION${NC}"
echo "============================================================"
echo "  Management API:  http://localhost:8080"
echo "  Frontend (UI):   http://localhost:3000"
if [ "$MAIN_SERVER" = true ]; then
echo "  Main Server WS:  ws://localhost:8989/xiaozhi/v1/"
echo "  MQTT Broker:     mqtt://localhost:1883"
fi
echo "------------------------------------------------------------"
echo "  Attach vao tmux:  tmux attach -t $SESSION"
echo "  Doi window:       Ctrl+b  n/p  hoac  Ctrl+b  w"
echo "  Tach ra:          Ctrl+b  d"
echo "  Dung tat ca:      tmux kill-session -t $SESSION"
echo "============================================================"
echo ""
echo "  Dang attach vao tmux..."
sleep 1
exec tmux attach -t "$SESSION"
