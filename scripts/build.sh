#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist

echo "Building better-ccusage..."
go build -trimpath -o dist/better-ccusage ./apps/better-ccusage/cmd/better-ccusage
echo "Building better-ccusage-mcp..."
go build -trimpath -o dist/better-ccusage-mcp ./apps/mcp/cmd/better-ccusage-mcp
echo "Building better-ccusage-codex..."
go build -trimpath -o dist/better-ccusage-codex ./apps/codex/cmd/better-ccusage-codex
echo "Building better-ccusage-opencode..."
go build -trimpath -o dist/better-ccusage-opencode ./apps/opencode/cmd/better-ccusage-opencode

ls -lh dist/
