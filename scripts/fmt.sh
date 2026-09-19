#!/usr/bin/env bash
set -euo pipefail
gofumpt -w .
goimports -w .
