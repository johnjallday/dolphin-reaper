#!/bin/bash

# Build script for dolphin-reaper plugin

set -e # Exit on any error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PLUGIN_NAME="dolphin-reaper"
OUTPUT_NAME="dolphin-reaper"

echo -e "${BLUE}Building dolphin-reaper RPC plugin...${NC}"

# Update dependencies
echo -e "${YELLOW}Updating dependencies...${NC}"
go mod tidy

# Get version information
VERSION=$(cat VERSION 2>/dev/null || echo "0.0.1")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo -e "${BLUE}Version: $VERSION${NC}"
echo -e "${BLUE}Build Time: $BUILD_TIME${NC}"
echo -e "${BLUE}Git Commit: $GIT_COMMIT${NC}"

# Build as standalone executable (RPC plugin)
echo -e "${YELLOW}Building RPC plugin executable with embedded version...${NC}"
if go build \
    -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT" \
    -o "$OUTPUT_NAME" main.go; then
  echo -e "${GREEN}✓ Successfully built $OUTPUT_NAME${NC}"
  echo -e "${GREEN}🏷️  Version $VERSION embedded in binary${NC}"
  echo -e "${BLUE}Plugin binary: $OUTPUT_NAME${NC}"
  ls -la "$OUTPUT_NAME"

  # Copy to dolphin-agent uploaded_plugins if it exists
  AGENT_PLUGINS_DIR="../dolphin-agent/uploaded_plugins"
  if [ -d "$AGENT_PLUGINS_DIR" ]; then
    cp "$OUTPUT_NAME" "$AGENT_PLUGINS_DIR/"
    echo -e "${GREEN}✓ Copied to dolphin-agent: $AGENT_PLUGINS_DIR/$OUTPUT_NAME${NC}"
  fi
else
  echo -e "${RED}✗ Failed to build plugin${NC}"
  exit 1
fi

echo -e "${GREEN}🎉 Build completed successfully!${NC}"

