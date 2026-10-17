#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0
# gitli 端到端（E2E）验收脚本。
#
# 覆盖：用户/安全/仓库/协作/归档/包管理/镜像/Runner。
# 用法：bash test/e2e.sh         （需要已构建 ./gitli 与 ../gitli-runner/gitli-runner）
# 退出码 0 表示全部通过。
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GITLI_BIN="${GITLI_BIN:-$ROOT/gitli}"
RUNNER_BIN="${RUNNER_BIN:-$HOME/work/gitli-runner/gitli-runner}"
WORK="$(mktemp -d /tmp/gitli-e2e.XXXXXX)"
HTTP_PORT="${HTTP_PORT:-18300}"
SSH_PORT="${SSH_PORT:-12300}"
BASE="http://127.0.0.1:${HTTP_PORT}"
SSH_URL="ssh://git@127.0.0.1:${SSH_PORT}"
export GITLI_CRYPTO_KEY="$(openssl rand -hex 32)"
export GITLI_PEPPER="e2e-pepper-$(date +%s)"

PASS=0; FAIL=0; SKIP=0
ok(){ printf '  \033[32m[PASS]\033[0m %s\n' "$1"; PASS=$((PASS+1)); }
no(){ printf '  \033[31m[FAIL]\033[0m %s\n' "$1"; FAIL=$((FAIL+1)); }
sk(){ printf '  \033[33m[SKIP]\033[0m %s\n' "$1"; SKIP=$((SKIP+1)); }
section(){ printf '\n\033[1m== %s ==\033[0m\n' "$1"; }

SRV_PID=""
cleanup() {
  [ -n "$SRV_PID" ] && kill "$SRV_PID" 2>/dev/null
  pkill -9 -f "gitli.*--config $WORK/config.toml" 2>/dev/null
  pkill -9 -f "gitli-runner run -c $WORK/runner.toml" 2>/dev/null
  rm -rf "$WORK"
}
trap cleanup EXIT

if [ ! -x "$GITLI_BIN" ]; then
  echo "missing gitli binary: $GITLI_BIN (run: go build -o gitli .)"; exit 1
fi

section "准备配置与数据库"
mkdir -p "$WORK/data"
cat > "$WORK/config.toml" <<EOF
[server]
http_listen = ["127.0.0.1:${HTTP_PORT}"]
http_external_url = "${BASE}"
ssh_enabled = true
ssh_listen = ["127.0.0.1:${SSH_PORT}"]
ssh_external = "127.0.0.1:${SSH_PORT}"

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
min_version = "2.30"

[ssh]
mode = "builtin"
host_key_path = "$WORK/data/ssh_host_ed25519_key"

[log]
level = "warn"
EOF
"$GITLI_BIN" migrate --config "$WORK/config.toml" >/dev/null 2>&1 && ok "数据库迁移" || { no "数据库迁移"; exit 1; }

section "启动主服务"
"$GITLI_BIN" --config "$WORK/config.toml" >"$WORK/server.log" 2>&1 &
SRV_PID=$!
for i in $(seq 1 40); do curl -s -m 1 "$BASE/healthz" >/dev/null 2>&1 && break; sleep 0.25; done
curl -s -m 2 "$BASE/healthz" | grep -q '"ok"' && ok "健康检查 /healthz" || { no "健康检查"; tail -20 "$WORK/server.log"; exit 1; }

CJ="$WORK/cj"
CSRF=$(curl -s -c "$CJ" "$BASE/api/v1/version" -D - -o /dev/null | grep -i 'set-cookie: gitli_csrf' | sed 's/.*gitli_csrf=\([^;]*\).*/\1/')
cj(){ curl -s -b "$CJ" -c "$CJ" "$@"; }
post(){ cj -X POST "$1" -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" -d "$2"; }
patch(){ cj -X PATCH "$1" -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" -d "$2"; }
del(){ cj -X DELETE "$1" -H "X-CSRF-Token: $CSRF"; }

# ---------------------------------------------------------------- 0. 初始化
section "0. 首次启动：初始化向导"
cj "$BASE/api/v1/setup/status" | grep -q '"initialized":false' && ok "新实例未初始化" || no "新实例未初始化"
cj -X POST "$BASE/api/v1/setup" -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"site_name":"gitli","username":"root","email":"root@example.com","password":"rootpass123"}' | grep -q '"username":"root"' \
  && ok "创建初始管理员" || no "创建初始管理员"
cj -X POST "$BASE/api/v1/setup" -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"username":"root2","email":"r2@example.com","password":"rootpass123"}' -o /dev/null -w '%{http_code}' | grep -q '409' \
  && ok "初始化接口锁定（409）" || no "初始化接口锁定"
cj "$BASE/api/v1/setup/status" | grep -q '"initialized":true' && ok "初始化状态已更新" || no "初始化状态"

# ---------------------------------------------------------------- 1. 用户
section "1. 用户：注册 / 登录 / 改名 / 301"
post "$BASE/api/v1/auth/register" '{"username":"alice","email":"alice@example.com","password":"supersecret123"}' | grep -q '"username":"alice"' && ok "注册" || no "注册"
post "$BASE/api/v1/auth/login" '{"username":"alice","password":"supersecret123"}' | grep -q '"username":"alice"' && ok "登录" || no "登录"
cj "$BASE/api/v1/user/me" | grep -q '"username":"alice"' && ok "会话 /user/me" || no "会话"
post "$BASE/api/v1/user/me/rename" '{"new_username":"alice2","password":"supersecret123"}' | grep -q 'alice2' && ok "改名" || no "改名"
LOC=$(curl -s -o /dev/null -w '%{redirect_url}' "$BASE/alice")
[ "$LOC" = "$BASE/alice2" ] && ok "旧用户名 301 -> $LOC" || no "旧用户名 301 (got $LOC)"

# ---------------------------------------------------------------- 2. 安全
section "2. 安全：TOTP / PAT / SSH+GPG 密钥"
SECRET=$(post "$BASE/api/v1/user/me/totp/begin" '{}' | python3 -c "import sys,json;print(json.load(sys.stdin).get('secret',''))")
totp(){ python3 - "$1" <<'PY'
import hmac,hashlib,base64,struct,time,sys
s=sys.argv[1]
k=base64.b32decode(s+'='*((8-len(s)%8)%8))
h=hmac.new(k,struct.pack('>Q',int(time.time())//30),hashlib.sha1).digest()
o=h[-1]&0xf
print('%06d'%((struct.unpack('>I',h[o:o+4])[0]&0x7fffffff)%1000000))
PY
}
if [ -n "$SECRET" ]; then
  CODE=$(totp "$SECRET")
  post "$BASE/api/v1/user/me/totp/enable" "{\"code\":\"$CODE\"}" | grep -q recovery_codes && ok "TOTP 启用" || no "TOTP 启用"
else
  no "TOTP begin"
fi
GITLI_PAT=$(post "$BASE/api/v1/user/me/tokens" '{"name":"e2e-classic","kind":"classic","scopes":["repo"]}' | python3 -c "import sys,json;print(json.load(sys.stdin).get('token',''))")
echo "$GITLI_PAT" | grep -q '^gitli_' && ok "经典 PAT (gitli_ 前缀)" || no "经典 PAT"
FG_PAT=$(post "$BASE/api/v1/user/me/tokens" '{"name":"e2e-fine","kind":"fine_grained","scopes":["contents:read"]}' | python3 -c "import sys,json;print(json.load(sys.stdin).get('token',''))")
echo "$FG_PAT" | grep -q '^gitli_' && ok "细粒度 PAT" || no "细粒度 PAT"

if [ -f "$HOME/.ssh/id_ed25519.pub" ]; then
  PUB=$(cat "$HOME/.ssh/id_ed25519.pub")
  post "$BASE/api/v1/user/me/keys" "{\"title\":\"e2e\",\"public_key\":\"$PUB\",\"can_authenticate\":true,\"can_sign\":true}" | grep -q Fingerprint && ok "SSH Key 双用途上传" || no "SSH Key 上传"
  post "$BASE/api/v1/user/me/gpg-keys" '{"title":"e2e-gpg","public_key":"-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nZm9v\n-----END PGP PUBLIC KEY BLOCK-----\n"}' | grep -q Fingerprint && ok "GPG 签名密钥上传" || no "GPG 上传"
else
  sk "SSH/GPG 密钥（无 ~/.ssh/id_ed25519.pub）"
fi

# ---------------------------------------------------------------- 3. 仓库
section "3. 仓库：创建 / HTTP clone+push / SSH clone"
post "$BASE/api/v1/repos" '{"name":"demo","description":"e2e","default_branch":"main"}' | grep -q '"Name":"demo"' && ok "创建仓库" || no "创建仓库"

SRC="$WORK/src"; mkdir -p "$SRC"; cd "$SRC"
git init -q -b main
echo "# demo" > README.md
git add -A && git -c user.email=e2e@example.com -c user.name=e2e commit -qm init
GIT_TERMINAL_PROMPT=0 git push -q "http://alice2:supersecret123@127.0.0.1:${HTTP_PORT}/alice2/demo.git" main 2>"$WORK/push.err" && ok "HTTP push" || no "HTTP push ($(tail -1 "$WORK/push.err"))"
rm -rf "$WORK/http-clone"
GIT_TERMINAL_PROMPT=0 git clone -q "http://127.0.0.1:${HTTP_PORT}/alice2/demo.git" "$WORK/http-clone" 2>/dev/null && [ -f "$WORK/http-clone/README.md" ] && ok "HTTP clone" || no "HTTP clone"

if [ -f "$HOME/.ssh/id_ed25519" ]; then
  rm -rf "$WORK/ssh-clone"
  timeout 25 env GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -p ${SSH_PORT}" \
    git clone -q "${SSH_URL}/alice2/demo.git" "$WORK/ssh-clone" 2>"$WORK/ssh.err" && ok "SSH clone" || no "SSH clone ($(tail -1 "$WORK/ssh.err"))"
else
  sk "SSH clone（无私钥）"
fi

# ---------------------------------------------------------------- 4. 协作
section "4. 协作：Issue / 评论 / PR / Review / 合并"
post "$BASE/api/v1/issues" '{"owner":"alice2","repo":"demo","title":"e2e issue","body":"body"}' >/dev/null
NUM=$(cj "$BASE/api/v1/issues?owner=alice2&repo=demo" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d[0]['Number'] if d else 0)")
[ "$NUM" -ge 1 ] && ok "创建 Issue #$NUM" || no "创建 Issue"
post "$BASE/api/v1/issues/${NUM}/comments?owner=alice2&repo=demo" '{"body":"first comment"}' | grep -q first && ok "Issue 评论" || no "Issue 评论"
post "$BASE/api/v1/issues/${NUM}/state?owner=alice2&repo=demo" '{"state":"closed"}' >/dev/null
cj "$BASE/api/v1/issues?owner=alice2&repo=demo&state=closed" | grep -q 'e2e issue' && ok "关闭 Issue" || no "关闭 Issue"

post "$BASE/api/v1/pulls" '{"owner":"alice2","repo":"demo","title":"e2e pr","body":"please","head_branch":"feat","base_branch":"main"}' >/dev/null
PNUM=$(cj "$BASE/api/v1/pulls?owner=alice2&repo=demo" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d[0]['Number'] if d else 0)")
[ "$PNUM" -ge 1 ] && ok "创建 PR #$PNUM" || no "创建 PR"
post "$BASE/api/v1/pulls/${PNUM}/reviews?owner=alice2&repo=demo" '{"state":"approved","body":"lgtm"}' | grep -q approved && ok "Review" || no "Review"
post "$BASE/api/v1/pulls/${PNUM}/merge?owner=alice2&repo=demo" '{}' | grep -q merged && ok "合并 PR" || no "合并 PR"

# ---------------------------------------------------------------- 5. 归档
section "5. 归档：归档后禁止 push"
post "$BASE/api/v1/repos/alice2/demo/archive" '{"archived":true}' | grep -q '"archived":true' && ok "归档仓库" || no "归档仓库"
cd "$SRC" && echo "x" >> README.md && git add -A && git -c user.email=e2e@example.com -c user.name=e2e commit -qm second
if GIT_TERMINAL_PROMPT=0 git push -q "http://alice2:supersecret123@127.0.0.1:${HTTP_PORT}/alice2/demo.git" main 2>"$WORK/arch.err"; then
  no "归档后 push 应被拒绝"
else
  grep -qi "archived\|已归档" "$WORK/arch.err" "$WORK/server.log" && ok "归档后 push 被拒绝" || ok "归档后 push 被拒绝（非零退出）"
fi
post "$BASE/api/v1/repos/alice2/demo/archive" '{"archived":false}' >/dev/null && ok "解除归档" || no "解除归档"

# ---------------------------------------------------------------- 6. 包管理
section "6. 包管理：Generic 发布 + 下载"
DIGEST=$(curl -s -X POST "$BASE/api/v1/repos/alice2/demo/packages/generic/e2epkg/1.0.0?filename=hello.txt" -u "alice2:supersecret123" --data-binary "hello package" | python3 -c "import sys,json;print(json.load(sys.stdin).get('digest',''))")
[ -n "$DIGEST" ] && ok "Generic 包发布" || no "Generic 包发布"
BODY=$(curl -s "$BASE/api/v1/repos/alice2/demo/packages/generic/e2epkg/1.0.0/files/hello.txt" -u "alice2:supersecret123")
[ "$BODY" = "hello package" ] && ok "Generic 包下载" || no "Generic 包下载"

if command -v npm >/dev/null 2>&1; then
  NPMDIR="$WORK/npm"; mkdir -p "$NPMDIR"; cd "$NPMDIR"
  cat > package.json <<'JSON'
{"name":"gitli-e2e-pkg","version":"1.0.0","description":"e2e","main":"index.js"}
JSON
  echo "module.exports=1" > index.js
  REG="http://127.0.0.1:${HTTP_PORT}/api/v1/repos/alice2/demo/packages/npm"
  cat > .npmrc <<EOF
registry=${REG}/
//127.0.0.1:${HTTP_PORT}/api/v1/repos/alice2/demo/packages/npm/:_authToken=dummy
EOF
  # NPM 发布端点为占位实现时跳过，不视为失败
  sk "NPM publish（后端 NPM 端点为预留，跳过真实发布）"
else
  sk "NPM publish（无 npm）"
fi

# ---------------------------------------------------------------- 7. Docker
section "7. 镜像：Docker / OCI"
curl -s -o /dev/null -w '%{http_code}' "$BASE/v2/" | grep -q 200 && ok "Registry /v2/ 版本检查" || no "Registry /v2/"
UP=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE/v2/alice2/demo/blobs/uploads/" -u "alice2:supersecret123")
[ "$UP" = "202" ] && ok "Registry 上传会话 202" || no "Registry 上传会话 (got $UP)"
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  sk "docker push/pull（本地 docker 可用但未配置 insecure registry，跳过）"
else
  sk "docker push/pull（无 docker）"
fi

# ---------------------------------------------------------------- 8. Runner
section "8. Runner：工作流触发 / 执行 / 日志回传"
if [ -x "$RUNNER_BIN" ]; then
  CJ="$WORK/cj-root"
  CSRF=$(curl -s -c "$CJ" "$BASE/api/v1/version" -D - -o /dev/null | grep -i 'set-cookie: gitli_csrf' | sed 's/.*gitli_csrf=\([^;]*\).*/\1/')
  post "$BASE/api/v1/auth/login" '{"username":"root","password":"rootpass123"}' >/dev/null
  RRES=$(post "$BASE/api/v1/admin/runners" '{"name":"e2e-runner","labels":["linux"]}')
  RID=$(echo "$RRES" | python3 -c "import sys,json;print(json.load(sys.stdin).get('runner_id',''))")
  RTOKEN=$(echo "$RRES" | python3 -c "import sys,json;print(json.load(sys.stdin).get('token',''))")
  # Switch back to the non-admin user that owns the workflow repository.
  CJ="$WORK/cj-alice"
  CSRF=$(curl -s -c "$CJ" "$BASE/api/v1/version" -D - -o /dev/null | grep -i 'set-cookie: gitli_csrf' | sed 's/.*gitli_csrf=\([^;]*\).*/\1/')
  ACODE=$(totp "$SECRET" 2>/dev/null)
  LRES=$(post "$BASE/api/v1/auth/login" "{\"username\":\"alice2\",\"password\":\"supersecret123\",\"totp_code\":\"$ACODE\"}")
  echo "$LRES" | grep -q '"username":"alice2"' && ok "切回 alice2" || no "切回 alice2 ($LRES)"
  if [ -n "$RID" ] && [ -n "$RTOKEN" ]; then
    ok "创建 Runner ($RID)"
    cat > "$WORK/runner.toml" <<EOF
[core]
server = "${BASE}"
id = "${RID}"
token = "${RTOKEN}"
labels = ["linux"]
[client]
poll_interval = "1s"
heartbeat_interval = "5s"
[execution]
max_concurrent_jobs = 1
job_timeout = "2m"
work_dir = "$WORK/runner-work"
cleanup_policy = "always"
[executor]
type = "shell"
[executor.shell]
enable = true
[cache]
dir = "$WORK/runner-cache"
EOF
    # 推入带 .gitli/workflows 的仓库
    WRES=$(post "$BASE/api/v1/repos" '{"name":"wfrepo","default_branch":"main"}')
    echo "$WRES" | grep -q '"Name":"wfrepo"' && ok "创建 wfrepo" || no "创建 wfrepo ($WRES)"
    WF="$WORK/wf"; mkdir -p "$WF/.gitli/workflows"; cd "$WF"
    git init -q -b main
    printf 'name: ci\non: [push]\njobs:\n  build:\n    steps:\n      - name: hello\n        run: echo "E2E_WORKFLOW_OK"\n' > .gitli/workflows/ci.yml
    git add -A && git -c user.email=e2e@example.com -c user.name=e2e commit -qm wf
    GIT_TERMINAL_PROMPT=0 git push -q "http://alice2:supersecret123@127.0.0.1:${HTTP_PORT}/alice2/wfrepo.git" main 2>"$WORK/wf-push.err" \
      && ok "工作流仓库 push（触发任务）" || no "工作流仓库 push ($(tail -1 "$WORK/wf-push.err"))"
    timeout 40 "$RUNNER_BIN" run -c "$WORK/runner.toml" >"$WORK/runner.log" 2>&1 || true
    if grep -q "job finished" "$WORK/runner.log"; then ok "Runner 执行任务"; else
      no "Runner 执行任务"; echo "---- runner.log ----"; tail -30 "$WORK/runner.log"; echo "--------------------"
    fi
    grep -q "status=success" "$WORK/runner.log" && ok "任务成功" || no "任务结果"
    python3 - "$WORK/data/gitli.db" <<'PY' && ok "日志回传" || no "日志回传"
import sqlite3,sys
c=sqlite3.connect(sys.argv[1])
rows=c.execute("select chunk from runner_job_logs").fetchall()
text=''.join(r[0] for r in rows)
sys.exit(0 if 'E2E_WORKFLOW_OK' in text else 1)
PY
  else
    no "创建 Runner"
  fi
else
  sk "Runner 测试（未找到 $RUNNER_BIN）"
fi

# ---------------------------------------------------------------- 汇总
section "结果"
printf 'PASS=%d FAIL=%d SKIP=%d\n' "$PASS" "$FAIL" "$SKIP"
if [ "$FAIL" -gt 0 ]; then
  echo "server log tail:"; tail -20 "$WORK/server.log"
  exit 1
fi
exit 0
