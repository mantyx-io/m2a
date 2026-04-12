#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
port="${1:-${PORT:-8080}}"

echo "Serving GitHub Pages preview from ${root}/docs"
echo "Open http://127.0.0.1:${port}/"
exec python3 -m http.server "$port" --bind 127.0.0.1 --directory "$root/docs"
