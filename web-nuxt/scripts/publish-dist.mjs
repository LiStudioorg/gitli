#!/usr/bin/env node
// 把 Nuxt 的静态产物（.output/public）发布到 Go 的 embed 目录（../web/dist）。
//
// 为什么需要这一步：//go:embed 只能引用包目录及其子目录，web/embed.go 写死了
// all:dist，无法直接引用 ../web-nuxt/.output/public。因此由本脚本把产物落到
// web/dist，Go 侧零改动，同时保持「单二进制」约束。
//
// 发布前先清空目标目录：产物文件名带内容哈希，不清空会让已删除的路由残留成
// 可访问的旧页面。
import { cpSync, existsSync, mkdirSync, readdirSync, rmSync, statSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const root = resolve(here, '..')
const source = join(root, '.output', 'public')
const target = resolve(root, '..', 'web', 'dist')

if (!existsSync(source)) {
  console.error(`发布失败：找不到 Nuxt 产物 ${source}，请先执行 npm run generate`)
  process.exit(1)
}

rmSync(target, { recursive: true, force: true })
mkdirSync(target, { recursive: true })
cpSync(source, target, { recursive: true })

let files = 0
let bytes = 0
;(function walk(dir) {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name)
    const st = statSync(full)
    if (st.isDirectory()) walk(full)
    else {
      files++
      bytes += st.size
    }
  }
})(target)

console.log(`已发布 Nuxt 产物 -> web/dist：${files} 个文件，${(bytes / 1024).toFixed(1)} KB`)
