#!/bin/bash
# DeepSeek API Monitor - Quick Start Script
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
FRONTEND_DIR="$SCRIPT_DIR/frontend"

echo "=== DeepSeek API Monitor ==="

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check dependencies
# Use the right Go binary (prefer /usr/local/go/bin/go for 1.22+)
if [ -x /usr/local/go/bin/go ]; then
  export PATH="/usr/local/go/bin:$PATH"
fi

command -v go >/dev/null 2>&1 || { echo "Error: Go is required (go version 1.22+). Install from https://go.dev/dl/"; exit 1; }
GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
echo "  Go version: $(go version)"

# Check API key
if [ -z "$DEEPSEEK_API_KEY" ]; then
  if [ -f "$BACKEND_DIR/config.yaml" ]; then
    KEY_IN_CONFIG=$(grep 'api_key:' "$BACKEND_DIR/config.yaml" | head -1 | awk '{print $2}' | tr -d '"')
    if [ -n "$KEY_IN_CONFIG" ] && [ "$KEY_IN_CONFIG" != '""' ] && [ "$KEY_IN_CONFIG" != "''" ]; then
      export DEEPSEEK_API_KEY="$KEY_IN_CONFIG"
    fi
  fi
fi

if [ -z "$DEEPSEEK_API_KEY" ]; then
  echo "Error: DEEPSEEK_API_KEY is not set."
  echo "  export DEEPSEEK_API_KEY='your-api-key'"
  echo "  Or edit backend/config.yaml and set api_key"
  exit 1
fi
export DEEPSEEK_API_KEY

# Require JWT_SECRET
if [ -z "$JWT_SECRET" ]; then
  echo ""
  echo -e "${YELLOW}⚠ JWT_SECRET not set. Generating one...${NC}"
  export JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)
  echo -e "  JWT_SECRET=$JWT_SECRET"
  echo -e "  ${YELLOW}Save this for production! Set JWT_SECRET env var to reuse across restarts.${NC}"
  echo ""
fi
export JWT_SECRET

mkdir -p "$BACKEND_DIR/data"

echo ""
echo -e "${YELLOW}[1/3]${NC} Building backend..."
cd "$BACKEND_DIR"
go build -o deepseek-monitor . 2>&1
echo -e "  ${GREEN}✓${NC} Backend built"

echo ""
echo -e "${YELLOW}[2/3]${NC} Building frontend..."
cd "$FRONTEND_DIR"
npm install --silent 2>/dev/null
npm run build 2>&1 | tail -1
echo -e "  ${GREEN}✓${NC} Frontend built"

echo ""
echo -e "${YELLOW}[3/3]${NC} Starting server..."
echo ""
echo -e "  ${GREEN}Open http://localhost:8080 in your browser${NC}"
echo ""
echo "  Press Ctrl+C to stop"
echo ""

cd "$BACKEND_DIR"
exec ./deepseek-monitor config.yaml
