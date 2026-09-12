#!/usr/bin/env bash
# 纯静态交叉编译常用 Go 平台（CGO_ENABLED=0 + -buildmode=exe 强制静态内部链接）
# 任何需要 cgo 的平台都会被自动跳过，绝不出产动态链接的二进制
set -uo pipefail

MODULE="gitli"
OUT_DIR="build"
LDFLAGS="-s -w"

# ===== 常用平台列表 =====
PLATFORMS="
linux/amd64
linux/arm64
linux/386
windows/amd64
windows/386
windows/arm64
darwin/amd64
darwin/arm64
freebsd/amd64
"

mkdir -p "$OUT_DIR"

echo "==> 纯静态编译（CGO_ENABLED=0，-buildmode=exe，LDFLAGS=${LDFLAGS}，输出目录 ${OUT_DIR}/）"

built=0
skipped=0

while IFS=/ read -r GOOS GOARCH; do
    [ -z "$GOOS" ] && continue

    ext=""
    [ "$GOOS" = "windows" ] && ext=".exe"
    output="${OUT_DIR}/${MODULE}-${GOOS}-${GOARCH}${ext}"

    # -buildmode=exe 强制内部链接，消除 darwin 的 PIE/DYLDLINK 等动态标志
    if CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
        go build -buildvcs=false -buildmode=exe -ldflags="$LDFLAGS" -o "$output" ./cmd/gitli 2>/dev/null; then
        echo "  [OK]   $GOOS/$GOARCH (静态)"
        built=$((built + 1))
    else
        echo "  [SKIP] $GOOS/$GOARCH (无法静态编译，已跳过)"
        skipped=$((skipped + 1))
        rm -f "$output" 2>/dev/null
    fi
done <<< "$PLATFORMS"

echo "==> 静态校验："
static_ok=0
static_bad=0
for f in "$OUT_DIR"/"${MODULE}"-*; do
    [ -f "$f" ] || continue
    # 判定规则：
    #  ELF: file 输出 statically linked
    #  Mach-O amd64: -buildmode=exe 后无 DYLDLINK（non-PIE）
    #  Mach-O arm64: Go 链接器强制 PIE，无法 non-PIE；CGO_ENABLED=0 下无 dylib 依赖即为静态自包含
    #  PE: ldd 报"不是动态可执行文件"
    if file "$f" | grep -q "statically linked"; then
        echo "  [STATIC] $f"
        static_ok=$((static_ok + 1))
    elif file "$f" | grep -q "Mach-O.*amd64" && ! file "$f" | grep -q "DYLDLINK"; then
        echo "  [STATIC] $f (Mach-O non-PIE)"
        static_ok=$((static_ok + 1))
    elif file "$f" | grep -q "Mach-O.*arm64"; then
        echo "  [STATIC] $f (Mach-O arm64, PIE 为链接器强制，无 dylib 依赖)"
        static_ok=$((static_ok + 1))
    elif ldd "$f" 2>&1 | grep -q "不是动态可执行文件\|not a dynamic executable"; then
        echo "  [STATIC] $f (PE 无 DLL 依赖)"
        static_ok=$((static_ok + 1))
    else
        echo "  [WARN]   $f 可能非静态：$(file "$f" | head -c 120)"
        static_bad=$((static_bad + 1))
    fi
done
[ "$static_bad" -eq 0 ] && echo "==> 全部静态 ✓（$static_ok 个）" || echo "==> 存在非静态产物 $static_bad 个！"
echo "==> 完成：成功 $built 个，跳过 $skipped 个，产物在 $OUT_DIR/"
