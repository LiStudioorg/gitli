#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0
# Build a set of GOOS/GOARCH targets into dist/ with checksums.
# Usage: build-targets.sh "linux/amd64,linux/armv7" [output_dir]
# Targets with an "armvN" suffix map to GOOS=linux GOARCH=arm GOARM=N.
set -euo pipefail

targets="${1:?targets required}"
outdir="${2:-dist}"
version="${VERSION:-dev}"
commit="${COMMIT:-none}"
date="${DATE:-unknown}"

mkdir -p "$outdir"

ldflags="-s -w -X github.com/LiStudioorg/gitli/internal/version.version=${version} -X github.com/LiStudioorg/gitli/internal/version.commit=${commit} -X github.com/LiStudioorg/gitli/internal/version.date=${date} -X github.com/LiStudioorg/gitli/internal/version.builtBy=ci"

IFS=',' read -ra list <<< "$targets"
for t in "${list[@]}"; do
  goos="${t%%/*}"
  goarch="${t##*/}"
  goarm=""
  if [[ "$goarch" =~ ^armv([0-9]+)$ ]]; then
    goarm="${BASH_REMATCH[1]}"
    goarch="arm"
  fi

  ext=""
  if [ "$goos" = "windows" ]; then ext=".exe"; fi
  name="gitli-${goos}-${goarch}${goarm}${ext}"

  echo ">> building ${name}"
  envs=(CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch")
  if [ -n "$goarm" ]; then
    envs+=(GOARM="$goarm")
  fi
  env "${envs[@]}" go build -trimpath -ldflags "$ldflags" -o "${outdir}/${name}" .

  ( cd "$outdir" && sha256sum "${name}" > "${name}.sha256" )
done
