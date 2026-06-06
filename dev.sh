#!/usr/bin/env bash
# Script chay moi truong dev local tren Windows voi Git Bash
# Su dung: ./dev.sh [sqlite] [reset] [backend] [frontend]

set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$ROOT/manager/backend"
FRONTEND_DIR="$ROOT/manager/frontend"

CONFIG_FILE="config/config.json"
RESET_FLAG=""
BACKEND_ONLY=false
FRONTEND_ONLY=false

# Xu ly tham so
for arg in "$@"; do
  case "$arg" in
    sqlite)    CONFIG_FILE="config/config.sqlite.json" ;;
    reset)     RESET_FLAG="-reset-db" ;;
    backend)   BACKEND_ONLY=true ;;
    frontend)  FRONTEND_ONLY=true ;;
    help|-h)
      echo "Su dung: ./dev.sh [tuy chon]"
      echo ""
      echo "  (khong tham so)   Chay ca hai, dung config.json (MySQL)"
      echo "  sqlite             Dung SQLite (khong can MySQL)"
      echo "  reset              Reset database truoc khi chay"
      echo "  backend            Chi chay backend"
      echo "  frontend           Chi chay frontend"
      echo ""
      echo "Vi du:"
      echo "  ./dev.sh sqlite          # SQLite, ca hai"
      echo "  ./dev.sh sqlite reset    # SQLite + reset DB"
      echo "  ./dev.sh frontend        # Chi frontend"
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

# Chay backend
if [ "$FRONTEND_ONLY" = false ]; then
  echo ""
  echo "=== Khoi dong Backend (cong 8080) ==="
  BACKEND_CMD="go mod tidy -e 2>/dev/null; go run main.go -config='$CONFIG_FILE' $RESET_FLAG"
  open_terminal "Backend :8080" "$BACKEND_DIR" "$BACKEND_CMD"
  echo "  [OK] Backend dang khoi dong trong cua so moi..."
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
echo "  Frontend (UI):  http://localhost:3000"
echo "  Backend (API):  http://localhost:8080"
echo "---------------------------------------------------------"
echo "  De dung: dong cac cua so terminal backend/frontend"
echo "========================================================="
