#!/usr/bin/env bash
# Script chay moi truong dev local tren Windows voi Git Bash
# Su dung: ./dev.sh [sqlite] [reset] [backend] [frontend] [no-main-server]

set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$ROOT/manager/backend"
FRONTEND_DIR="$ROOT/manager/frontend"

CONFIG_FILE="config/config.json"
RESET_FLAG=""
BACKEND_ONLY=false
FRONTEND_ONLY=false
MAIN_SERVER=false   # Phai bật thủ công vì cần CGO + libopus + ONNX Runtime

# Xu ly tham so
for arg in "$@"; do
  case "$arg" in
    sqlite)       CONFIG_FILE="config/config.sqlite.json" ;;
    reset)        RESET_FLAG="-reset-db" ;;
    backend)      BACKEND_ONLY=true ;;
    frontend)     FRONTEND_ONLY=true ;;
    main-server)  MAIN_SERVER=true ;;
    help|-h)
      echo "Su dung: ./dev.sh [tuy chon]"
      echo ""
      echo "  (khong tham so)   Management (8080) + Frontend (3000)"
      echo "  sqlite             Dung SQLite (khong can MySQL)"
      echo "  reset              Reset database truoc khi chay"
      echo "  backend            Chi chay management backend"
      echo "  frontend           Chi chay frontend"
      echo "  main-server        Chay them main xiaozhi server (8989)"
      echo "                     Yeu cau: CGO, libopus, ONNX Runtime (Linux/WSL2)"
      echo ""
      echo "Vi du:"
      echo "  ./dev.sh sqlite              # SQLite, management + frontend"
      echo "  ./dev.sh sqlite reset        # SQLite + reset DB"
      echo "  ./dev.sh frontend            # Chi frontend"
      echo "  ./dev.sh sqlite main-server  # Day du (can Linux/WSL2)"
      exit 0
      ;;
  esac
done

# Kiem tra cong cu
echo "=== Kiem tra moi truong ==="
command -v go  &>/dev/null && echo "  [OK] Go:      $(go version)" || { echo "  [LOI] Go chua cai - https://go.dev/dl/"; exit 1; }
command -v node &>/dev/null && echo "  [OK] Node.js: $(node --version)" || { echo "  [LOI] Node.js chua cai - https://nodejs.org/"; exit 1; }

# Tao config SQLite neu chua co
if [[ "$CONFIG_FILE" == *sqlite* ]] && [ ! -f "$BACKEND_DIR/$CONFIG_FILE" ]; then
  echo ""
  echo "[INFO] Tao config SQLite..."
  node -e "
    const fs = require('fs');
    const cfg = JSON.parse(fs.readFileSync('$BACKEND_DIR/config/config.json', 'utf8'));
    cfg.database.type = 'sqlite';
    fs.writeFileSync('$BACKEND_DIR/config/config.sqlite.json', JSON.stringify(cfg, null, 2));
    console.log('[OK] Tao $BACKEND_DIR/config/config.sqlite.json');
  "
fi

# Xac nhan reset
if [ -n "$RESET_FLAG" ]; then
  echo ""
  echo "[CANH BAO] Se reset toan bo database!"
  read -rp "Xac nhan? (y/N): " confirm
  [[ "$confirm" =~ ^[Yy]$ ]] || { echo "Da huy."; exit 0; }
fi

# Cai npm neu chua co
if [ "$BACKEND_ONLY" = false ] && [ ! -d "$FRONTEND_DIR/node_modules" ]; then
  echo ""
  echo "=== Cai npm dependencies ==="
  cd "$FRONTEND_DIR" && npm install
fi

# Ham mo terminal moi (mintty la terminal cua Git Bash tren Windows)
open_terminal() {
  local title="$1"
  local dir="$2"
  local cmd="$3"
  # Thu mintty truoc (Git Bash built-in), fallback sang cmd
  if command -v mintty &>/dev/null; then
    mintty --title "$title" -e bash -c "cd '$dir' && $cmd; echo; read -rp 'Nhan Enter de dong...' _" &
  else
    start "" bash -c "cd '$dir' && $cmd; echo; read -rp 'Nhan Enter de dong...' _" &
  fi
}

# Chay main xiaozhi server (port 8989) - chi khi co flag 'main-server'
if [ "$FRONTEND_ONLY" = false ] && [ "$MAIN_SERVER" = true ]; then
  echo ""
  echo "=== Khoi dong Main Xiaozhi Server (cong 8989) ==="
  open_terminal "Main Server :8989" "$ROOT" "go run ./cmd/server/..."
  echo "  [OK] Main server dang khoi dong trong cua so moi..."
  sleep 2
fi

# Chay management backend (port 8080)
if [ "$FRONTEND_ONLY" = false ]; then
  echo ""
  echo "=== Khoi dong Management Backend (cong 8080) ==="
  BACKEND_CMD="go mod tidy -e 2>/dev/null; go run main.go -config='$CONFIG_FILE' $RESET_FLAG"
  open_terminal "Management :8080" "$BACKEND_DIR" "$BACKEND_CMD"
  echo "  [OK] Management backend dang khoi dong trong cua so moi..."
  sleep 2
fi

# Chay frontend
if [ "$BACKEND_ONLY" = false ]; then
  echo ""
  echo "=== Khoi dong Frontend (cong 3000) ==="
  open_terminal "Frontend :3000" "$FRONTEND_DIR" "npm run dev"
  echo "  [OK] Frontend dang khoi dong trong cua so moi..."
fi

echo ""
echo "========================================================="
echo "  MOI TRUONG DEV DA KHOI DONG"
echo "========================================================="
echo "  Frontend (UI):      http://localhost:3000"
echo "  Management (API):   http://localhost:8080"
if [ "$MAIN_SERVER" = true ]; then
  echo "  Main Server (WS):   ws://localhost:8989/xiaozhi/v1/"
fi
echo "---------------------------------------------------------"
echo "  De dung: dong cac cua so terminal tuong ung"
echo "========================================================="
