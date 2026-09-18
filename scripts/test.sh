#!/usr/bin/env bash
set -euo pipefail
# If there are no Go files at all yet, treat as a clean empty-module state.
if ! find . -name '*.go' -not -path './.git/*' -not -path './.worktrees/*' -not -path './vendor/*' | grep -q .; then
    echo "no Go files yet — skipping tests"
    exit 0
fi
go test ./... -race -count=1
