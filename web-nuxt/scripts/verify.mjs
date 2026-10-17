#!/usr/bin/env node
// 顺序执行前端验证链。用 node 子进程而不是 `&&` 链，是因为 npm 在 Windows 上
// 通过 cmd.exe 解析 `&&` 时会走 PATH 查找 `nuxt`，可能命中全局安装的另一份 nuxt，
// 造成「单跑绿、链跑红」的假红。这里全部走本仓库 node_modules 的具体入口。
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const root = dirname(dirname(fileURLToPath(import.meta.url)))

/** @type {Array<[string, string]>} */
const steps = [
  [process.execPath, join(root, 'node_modules', 'nuxt', 'bin', 'nuxt.mjs') + ' typecheck'],
  [process.execPath, join(root, 'node_modules', 'vue-tsc', 'bin', 'vue-tsc.js') + ' --noEmit -p tests/tsconfig.json'],
  [process.execPath, join(root, 'node_modules', 'vitest', 'vitest.mjs') + ' run'],
]

for (const [bin, argString] of steps) {
  const args = argString.split(' ')
  const label = `$ ${bin.split(/[\\/]/).slice(-3).join('/')} ${argString}`
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
console.log('\n=== verify OK')
