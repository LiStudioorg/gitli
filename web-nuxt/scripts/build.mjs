#!/usr/bin/env node
// 构建前端并发布到 Go 的 embed 目录。
//
// 用 node 子进程而不是 `nuxt generate && node publish-dist.mjs`：npm 在 Windows
// 上经 cmd.exe 解析 `&&` 时会走 PATH 查找 nuxt，可能命中全局安装的另一份版本，
// 造成「本地构建成功、CI 产物不同」。这里全部走本仓库 node_modules 的具体入口。
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const root = dirname(dirname(fileURLToPath(import.meta.url)))

const steps = [
  [process.execPath, [join(root, 'node_modules', 'nuxt', 'bin', 'nuxt.mjs'), 'generate'], 'nuxt generate'],
  [process.execPath, [join(root, 'scripts', 'publish-dist.mjs')], 'publish dist'],
]

for (const [bin, args, label] of steps) {
  console.log(`\n=== ${label}`)
  const res = spawnSync(bin, args, { cwd: root, stdio: 'inherit' })
  if (res.error) {
    console.error(`spawn failed: ${res.error.message}`)
    process.exit(1)
  }
  if (res.status !== 0) {
    console.error(`\n=== FAILED (exit ${res.status}): ${label}`)
    process.exit(res.status ?? 1)
  }
}
console.log('\n=== build OK')
