# gitli Web（Nuxt 4 重构中）

这个目录是 gitli 前端的**下一代实现**，与仍在生效的 `web/`（Vite + Vue 3）**并行存在**。
在阶段 4 切换之前，`web/` 才是权威产物：Go 仍然 `//go:embed all:dist` 嵌入 `web/dist`，
本目录的任何产物都不会进入二进制。

## 为什么必须是纯静态

AGENTS.md 的核心约束不可违背：单二进制、无外部强依赖、空闲内存 < 150 MB。
Nuxt 4 的默认形态是 Node SSR 服务，那会引入一个常驻 Node 进程。因此这里锁定：

- `ssr: false` —— 只产出客户端 SPA，不做服务端渲染；
- `nitro.preset = 'static'` —— 不生成 Nitro 运行时，产物是纯文件；
- `npm run build` = `nuxt generate` —— 输出 `.output/public/`。

`generate` 会同时产出 `index.html` 与内容一致的 `200.html`，因此现有
`web/embed.go` 的「未知路径回退 index.html」在切换时**零 Go 改动**即可支撑
`/owner/repo`、`/users/x` 这些无法预渲染的动态路由。

## 本地跑起来

```bash
cd web-nuxt
npm install          # 需要 Node >= 22.12（nuxt 4.6 的 engines）
npm run dev          # http://localhost:5174，/api 代理到 127.0.0.1:8080
npm run generate     # 静态产物：.output/public
npm run typecheck    # nuxt typecheck（比旧前端更严：含 noUncheckedIndexedAccess）
npm run verify       # typecheck + 测试类型门禁 + 42 个行为测试
npm test             # 只跑行为测试（tests/**）
```

### 测试分层

| 层 | 位置 | 覆盖 |
| --- | --- | --- |
| 行为单测 | `tests/*.test.ts`（42 用例） | 主题循环、快捷键 `g i`/`g p`/`c`/`/`、Ctrl+K 面板、SSE 解析与退避重连、未读计数、i18n 语言切换、CSRF 头、登录回读 `/user/me`、Tab 契约、TopNav 管理入口可见性、12 视图与页面壳导入契约 |
| 类型门禁 | `tests/tsconfig.json`（继承 `.nuxt`）+ `tsc -p` | 测试代码本身也在类型检查内（曾发现它形同虚设，见下） |
| 端到端 | 手工脚本 | Nuxt dev(5174) → 真实 gitli(8080)：SPA 回退、代理、CSRF、Session、注册登录、建仓、Issue/评论、SSE |

> 两个踩过的坑，避免重犯：
> 1. `nuxt typecheck` 依赖 `.nuxt` 生成物，与 `nuxt prepare`/`postinstall` 并发会假红。
> 2. npm 在 Windows 用 `cmd.exe` 解析 `&&` 链时可能命中全局 `nuxt`。因此 `npm run verify`
>    走 `scripts/verify.mjs`，用本仓库 `node_modules` 的具体入口顺序执行。

后端另开一个实例即可：`./gitli serve --config config.toml`（默认监听 `:8080`）。

## 与旧前端的对应关系

| 旧 `web/src` | 新 `web-nuxt/app` | 说明 |
| --- | --- | --- |
| `router/index.ts`（12 条路由） | `pages/**`（12 个薄壳） | 路径逐条对齐；页面壳只负责 `definePageMeta` 与导入视图 |
| `views/*.vue` | `views/*.vue` | **原样复制**，放在 `app/views/` 以保持相对导入深度不变 |
| `components/`、`composables/`、`stores/`、`api/`、`utils/`、`constants/`、`locales/`、`i18n.ts`、`style.css` | 同名同层级 | 原样复制，避免迁移期两份实现行为漂移 |
| `router.beforeEach` 守卫 | `middleware/auth.global.ts` | `requiresAuth` / `requiresAdmin` / `guestOnly` 语义逐条对应 |
| `createApp().use(pinia)` | `@pinia/nuxt` | options store 写法不变 |
| `createApp().use(i18n)` | `plugins/i18n.client.ts` | 仍用 vue-i18n **9.14.5**（与旧前端同版本，避免行为漂移），不引入 `@nuxtjs/i18n` |

## 当前状态（阶段 3 进行中）

- `nuxt typecheck`、`npm run verify`（42 个行为测试）、`nuxt generate` 均通过；12 条路由全部产出。
- 已用「Nuxt dev（5174）+ 真实 gitli（8080）」做过端到端冒烟：SPA 外壳与深链接回退、
  `/api` 代理、CSRF 双提交、Session Cookie、注册/登录（登录体与 `/user/me` 一致）、
  建仓、Issue/评论（201）、SSE `text/event-stream` 全部走通。
- 自动化行为回归已覆盖：暗色模式、Ctrl+K 命令面板、`g i`/`g p`/`c`/`/` 快捷键、
  SSE 解析与退避重连 + 未读计数、i18n 中英切换、CSRF 头注入、登录后 `isAdmin`、
  仓库页 Tab 契约、TopNav 管理入口、12 视图/页面壳导入契约。
- 尚未做：RepoView 各 Tab 的数据流回归、AdminView/UserSettingsView 各面板交互回归、
  响应式与无障碍抽查。

## 切换前必须一起改的外部条件

1. `.github/workflows/ci.yml` 与 `cd.yml` 目前钉 `node-version: '20'`，nuxt 4 要求
   `^22.22.3 || ^24.15.0 || >=26.0.0`，必须同步抬版本，否则 Release 直接失败。
2. `Makefile:17` 的 `cd web && npm install && npm run build` 需要换成 Nuxt 构建并把
   `.output/public` 落到 embed 目录。
3. `web/dist` 是提交进 git 的产物（让 `go build` 不依赖 Node）；切换时要决定 Nuxt 产物
   的落盘位置与是否继续提交。
