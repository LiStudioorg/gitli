#!/usr/bin/env sh
# SPDX-License-Identifier: AGPL-3.0
# Build the frontend then the single gitli binary.
set -eu
cd "$(dirname "$0")/.."

if [ -d web ]; then
  (cd web && npm install && npm run build)
fi

go build -o gitli .
echo "built ./gitli"
