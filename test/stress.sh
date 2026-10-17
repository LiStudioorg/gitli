#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0
# gitli 压力测试：并发 Git clone/push + 持续 API 请求，观察内存与 Goroutine。
#
# 用法：bash test/stress.sh [并发数] [轮次]
# 默认 50 并发、3 轮。退出码 0 表示无泄漏迹象。
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GITLI_BIN="${GITLI_BIN:-$ROOT/gitli}"
CONCURRENCY="${1:-50}"
ROUNDS="${2:-3}"
WORK="$(mktemp -d /tmp/gitli-stress.XXXXXX)"
HTTP_PORT="${HTTP_PORT:-18400}"
BASE="http://127.0.0.1:${HTTP_PORT}"
export GITLI_CRYPTO_KEY="$(openssl rand -hex 32)"
export GITLI_PEPPER="stress-pepper-$(date +%s)"
export GIT_TERMINAL_PROMPT=0

# 内存增长容忍阈值（KB），超过则判定为疑似泄漏。
RSS_GROWTH_LIMIT_KB="${RSS_GROWTH_LIMIT_KB:-51200}"
GOROUTINE_GROWTH_LIMIT="${GOROUTINE_GROWTH_LIMIT:-200}"

PASS=0; FAIL=0
ok(){ printf '  \033[32m[PASS]\033[0m %s\n' "$1"; PASS=$((PASS+1)); }
no(){ printf '  \033[31m[FAIL]\033[0m %s\n' "$1"; FAIL=$((FAIL+1)); }
info(){ printf '  \033[36m[INFO]\033[0m %s\n' "$1"; }
section(){ printf '\n\033[1m== %s ==\033[0m\n' "$1"; }

SRV_PID=""
cleanup() {
  [ -n "$SRV_PID" ] && kill "$SRV_PID" 2>/dev/null
  pkill -9 -f "gitli.*--config $WORK/config.toml" 2>/dev/null
  rm -rf "$WORK"
}
trap cleanup EXIT

if [ ! -x "$GITLI_BIN" ]; then
  echo "missing gitli binary: $GITLI_BIN (run: go build -o gitli .)"; exit 1
fi

section "准备配置"
mkdir -p "$WORK/data"
cat > "$WORK/config.toml" <<EOF
[server]
http_listen = ["127.0.0.1:${HTTP_PORT}"]
http_external_url = "${BASE}"
pprof = true

[database]
driver = "sqlite"
dsn = "$WORK/data/gitli.db"

[storage]
repos = "$WORK/data/repos"
lfs = "$WORK/data/lfs"
packages = "$WORK/data/packages"
avatars = "$WORK/data/avatars"
attachments = "$WORK/data/attachments"

[security.password]
pepper = "${GITLI_PEPPER}"

[git]
bin = ""

[log]
level = "warn"
EOF
"$GITLI_BIN" migrate --config "$WORK/config.toml" >/dev/null 2>&1 || { echo "migrate failed"; exit 1; }

section "启动主服务"
"$GITLI_BIN" --config "$WORK/config.toml" >"$WORK/server.log" 2>&1 &
SRV_PID=$!
for i in $(seq 1 60); do curl -s -m 1 "$BASE/healthz" >/dev/null 2>&1 && break; sleep 0.25; done
curl -s -m 2 "$BASE/healthz" | grep -q '"ok"' && ok "服务已启动" || { no "服务启动失败"; tail -20 "$WORK/server.log"; exit 1; }

rss_kb(){ awk '/VmRSS/{print $2}' "/proc/$SRV_PID/status" 2>/dev/null || echo 0; }
goroutines(){
  curl -s -m 2 "$BASE/metrics" | awk '$1=="go_goroutines"{print $2}' | head -1
}

CJ="$WORK/cj"
CSRF=$(curl -s -c "$CJ" "$BASE/api/v1/version" -D - -o /dev/null | grep -i 'set-cookie: gitli_csrf' | sed 's/.*gitli_csrf=\([^;]*\).*/\1/')
cj(){ curl -s -b "$CJ" -c "$CJ" "$@"; }
post(){ cj -X POST "$1" -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" -d "$2"; }

section "准备测试数据"
post "$BASE/api/v1/setup" '{"username":"stress","email":"stress@example.com","password":"stresstest123"}' >/dev/null
post "$BASE/api/v1/auth/login" '{"username":"stress","password":"stresstest123"}' >/dev/null
cj "$BASE/api/v1/user/me" | grep -q '"username":"stress"' && ok "登录" || { no "登录"; exit 1; }
post "$BASE/api/v1/repos" '{"name":"bench","default_branch":"main"}' >/dev/null
PUSH_URL="http://stress:stresstest123@127.0.0.1:${HTTP_PORT}/stress/bench.git"

SRC="$WORK/src"; mkdir -p "$SRC"; cd "$SRC"
git init -q -b main
for i in $(seq 1 20); do head -c 2048 /dev/urandom | base64 > "file$i.bin"; done
git add -A && git -c user.email=s@example.com -c user.name=s commit -qm init
if ! git push -q "$PUSH_URL" main 2>"$WORK/init-push.err"; then
  # A leftover branch from a previous run may be ahead; reset it explicitly.
  git push -q --force "$PUSH_URL" main 2>>"$WORK/init-push.err" && ok "初始 push（强制重置）" || {
    no "初始 push ($(tail -1 "$WORK/init-push.err"))"; exit 1; }
else
  ok "初始 push"
fi
cj "$BASE/api/v1/repos/stress/bench/issues" -H 'Content-Type: application/json' \
  -H "X-CSRF-Token: $CSRF" -d '{"title":"stress issue","body":"x"}' >/dev/null

section "基线"
RSS_BASE=$(rss_kb); GORO_BASE=$(goroutines)
info "基线 RSS: ${RSS_BASE} KB, goroutines: ${GORO_BASE:-n/a}"

# 后台持续 API 请求
api_hammer(){
  local end=$((SECONDS + $1))
  while [ $SECONDS -lt $end ]; do
    curl -s -m 2 -o /dev/null "$BASE/api/v1/repos/stress/bench"
    curl -s -m 2 -o /dev/null "$BASE/api/v1/explore/repos"
    curl -s -m 2 -o /dev/null "$BASE/api/v1/explore/trending"
    curl -s -m 2 -o /dev/null "$BASE/api/v1/version"
    curl -s -m 2 -o /dev/null "$BASE/metrics"
  done
}

run_round(){
  local round=$1
  local jobs=()
  local errs="$WORK/round$round-errors.log"
  : > "$errs"

  for i in $(seq 1 "$CONCURRENCY"); do
    (
      dest="$WORK/clone-$round-$i"
      if ! timeout 60 git clone -q "$PUSH_URL" "$dest" 2>>"$errs"; then exit 1; fi
      cd "$dest" || exit 1
      if ! timeout 60 git push -q origin "HEAD:refs/heads/worker-$round-$i" 2>>"$errs"; then exit 1; fi
      rm -rf "$dest"
    ) &
    jobs+=($!)
  done

  local failed=0
  for p in "${jobs[@]}"; do wait "$p" || failed=$((failed+1)); done
  echo "$failed"
}

section "压测：${CONCURRENCY} 并发 × ${ROUNDS} 轮（clone+push）+ 持续 API"
api_hammer $((CONCURRENCY * ROUNDS + 20)) &
API_PID=$!

WORST_RSS=$RSS_BASE
for r in $(seq 1 "$ROUNDS"); do
  T0=$(date +%s)
  FAILED=$(run_round "$r")
  T1=$(date +%s)
  RSS=$(rss_kb); GORO=$(goroutines)
  [ "$RSS" -gt "$WORST_RSS" ] && WORST_RSS=$RSS
  if [ "$FAILED" -eq 0 ]; then
    ok "第 $r 轮完成（$((T1-T0))s, RSS=${RSS}KB, goroutines=${GORO:-n/a}）"
  else
    no "第 $r 轮有 $FAILED 个 worker 失败（日志 $WORK/round$r-errors.log）"
    head -5 "$WORK/round$r-errors.log" 2>/dev/null | sed 's/^/      /'
  fi
done
wait "$API_PID" 2>/dev/null

section "收敛：等待 30s 观察释放"
sleep 30
RSS_END=$(rss_kb); GORO_END=$(goroutines)
GROWTH=$((RSS_END - RSS_BASE))
info "结束 RSS: ${RSS_END} KB（增长 ${GROWTH} KB）, goroutines: ${GORO_END:-n/a}"

if [ "$GROWTH" -le "$RSS_GROWTH_LIMIT_KB" ]; then
  ok "内存增长 ${GROWTH} KB ≤ 阈值 ${RSS_GROWTH_LIMIT_KB} KB"
else
  no "内存增长 ${GROWTH} KB > 阈值 ${RSS_GROWTH_LIMIT_KB} KB（疑似泄漏）"
fi

if [ -n "$GORO_BASE" ] && [ -n "$GORO_END" ]; then
  GG=$((GORO_END - GORO_BASE))
  info "goroutine 增长: $GG"
else
  info "goroutine 指标不可用（/metrics 未暴露 go_goroutines）"
fi

section "稳定性"
curl -s -m 3 "$BASE/healthz" | grep -q '"ok"' && ok "压测后 /healthz 正常" || no "压测后 /healthz 异常"
kill -0 "$SRV_PID" 2>/dev/null && ok "主进程仍在运行" || no "主进程已退出"

section "结果"
printf 'PASS=%d FAIL=%d\n' "$PASS" "$FAIL"
if [ "$FAIL" -gt 0 ]; then
  echo "server log tail:"; tail -20 "$WORK/server.log"
  echo "工作目录保留: $WORK"
  trap - EXIT
  exit 1
fi
exit 0
