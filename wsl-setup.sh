#!/usr/bin/env bash
# =============================================================
#  WSL Setup — cai dat moi truong dev cho xiaozhi-esp32-server
#  Chay 1 lan duy nhat. Su dung: bash wsl-setup.sh
# =============================================================
set -e

ORT_VERSION="1.20.0"   # ONNX Runtime version
GO_VERSION="1.24.2"
NODE_VERSION="20"      # LTS

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[OK]${NC} $*"; }
info() { echo -e "${CYAN}[..] $*${NC}"; }
warn() { echo -e "${YELLOW}[!!]${NC} $*"; }
fail() { echo -e "${RED}[ERR]${NC} $*"; exit 1; }

echo ""
echo "============================================================"
echo "  WSL Setup: xiaozhi-esp32-server dev environment"
echo "  ORT: $ORT_VERSION | Go: $GO_VERSION | Node: $NODE_VERSION"
echo "============================================================"
echo ""

# --- 1. System packages ---
info "Cap nhat apt va cai packages can thiet..."
sudo apt-get update -qq
sudo apt-get install -y \
  build-essential gcc g++ make \
  libopus-dev libopus0 libopusfile-dev \
  libc++-dev libc++abi-dev \
  wget curl git git-lfs tmux \
  pkg-config
ok "System packages da cai"

# --- 2. Go ---
if command -v go &>/dev/null; then
  CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
  ok "Go da co: $CURRENT_GO (skip)"
else
  info "Cai Go $GO_VERSION..."
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz

  # Them vao PATH neu chua co
  PROFILE="${HOME}/.bashrc"
  if ! grep -q "/usr/local/go/bin" "$PROFILE" 2>/dev/null; then
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> "$PROFILE"
  fi
  export PATH=$PATH:/usr/local/go/bin
  ok "Go $GO_VERSION da cai"
fi

# --- 3. Node.js ---
if command -v node &>/dev/null; then
  ok "Node.js da co: $(node --version) (skip)"
else
  info "Cai Node.js $NODE_VERSION (LTS) qua NodeSource..."
  curl -fsSL "https://deb.nodesource.com/setup_${NODE_VERSION}.x" | sudo -E bash -
  sudo apt-get install -y nodejs
  ok "Node.js da cai: $(node --version)"
fi

# --- 4. ONNX Runtime ---
ORT_LIB="/usr/local/lib/libonnxruntime.so.${ORT_VERSION}"
ORT_SYMLINK="/usr/local/lib/libonnxruntime.so"

if [ -f "$ORT_SYMLINK" ]; then
  ok "ONNX Runtime da co (skip)"
else
  info "Cai ONNX Runtime $ORT_VERSION..."
  ORT_DIR="onnxruntime-linux-x64-${ORT_VERSION}"
  ORT_TAR="${ORT_DIR}.tgz"
  ORT_URL="https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/${ORT_TAR}"

  wget -q --show-progress "$ORT_URL" -O "/tmp/${ORT_TAR}"
  tar -C /tmp -xzf "/tmp/${ORT_TAR}"
  sudo cp /tmp/${ORT_DIR}/lib/libonnxruntime*.so* /usr/local/lib/
  sudo cp -r /tmp/${ORT_DIR}/include/* /usr/local/include/
  sudo ldconfig
  rm -rf "/tmp/${ORT_TAR}" "/tmp/${ORT_DIR}"
  ok "ONNX Runtime $ORT_VERSION da cai"
fi

# --- 5. Ghi bien moi truong vao .bashrc ---
PROFILE="${HOME}/.bashrc"
TEN_VAD_LIB_PATH="${PROJECT_DIR}/lib/ten-vad/lib/Linux/x64"
if ! grep -q "ONNXRUNTIME_LIB_PATH" "$PROFILE" 2>/dev/null; then
  {
    echo ""
    echo "# xiaozhi-esp32-server: ONNX Runtime + ten_vad"
    echo "export ONNXRUNTIME_LIB_PATH=/usr/local/lib/libonnxruntime.so"
    echo "export CGO_ENABLED=1"
    echo "export LD_LIBRARY_PATH=${TEN_VAD_LIB_PATH}:\$LD_LIBRARY_PATH"
  } >> "$PROFILE"
  ok "Bien moi truong da them vao $PROFILE"
else
  ok "Bien moi truong da co (skip)"
fi
export ONNXRUNTIME_LIB_PATH=/usr/local/lib/libonnxruntime.so
export CGO_ENABLED=1
export LD_LIBRARY_PATH="${TEN_VAD_LIB_PATH}:$LD_LIBRARY_PATH"

# --- 6. Kiem tra build supertonic ---
echo ""
info "Kiem tra build -tags supertonic..."
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_DIR"
if CGO_ENABLED=1 go build -tags supertonic ./internal/domain/tts/supertonic/... 2>&1; then
  ok "Build -tags supertonic: PASS"
else
  warn "Build -tags supertonic FAIL — kiem tra log o tren"
  warn "Nguyen nhan pho bien: libonnxruntime khong tim thay hoac phien ban sai"
  warn "Thu: export ONNXRUNTIME_LIB_PATH=/usr/local/lib/libonnxruntime.so.${ORT_VERSION}"
fi

echo ""
echo "============================================================"
echo -e "  ${GREEN}SETUP HOAN THANH${NC}"
echo "============================================================"
echo "  Chay server: bash wsl-dev.sh"
echo "  Chay voi supertonic TTS: bash wsl-dev.sh supertonic"
echo ""
echo "  LUU Y: Khoi dong lai terminal (hoac: source ~/.bashrc)"
echo "         de bien moi truong co hieu luc."
echo "============================================================"
