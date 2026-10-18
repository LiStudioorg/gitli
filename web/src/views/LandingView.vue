<script setup lang="ts">
import { RouterLink } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import FeatureCard from '../components/common/FeatureCard.vue'

const features = [
  { icon: '📦', title: '代码托管', lines: ['SSH/HTTP Git', '分支/标签保护'] },
  { icon: '🤝', title: '协作与评审', lines: ['Issue/PR/Review', '行级评论'] },
  { icon: '📚', title: '包管理', lines: ['NPM/Docker/Go', 'Maven/Generic'] },
  { icon: '🔒', title: '安全', lines: ['TOTP 2FA', '经典/细粒度 PAT'] },
  { icon: '⚙️', title: 'CI/CD', lines: ['宿主机执行', '实时日志'] },
  { icon: '🪶', title: '单二进制', lines: ['100MB 内存以内', '多数据库支持'] },
]
</script>

<template>
  <div class="min-h-screen bg-[#f6f8fa] dark:bg-[#0d1117]">
    <TopNav variant="landing" />

    <section class="mx-auto flex max-w-[1000px] flex-col items-center px-4 pb-12 pt-16 text-center md:px-6">
      <h1 class="text-3xl font-semibold leading-tight tracking-tight text-[#1f2328] dark:text-[#e6edf3] md:text-4xl">
        轻量级自研 Git 服务平台，低内存，单二进制，Go + Vue3
      </h1>
      <p class="mt-4 max-w-2xl text-base leading-relaxed text-[#59636e]">
        支持 SSH/HTTP Git 协议、Issue/PR、TOTP、细粒度 PAT、包管理（NPM/Docker/Go/Maven）、CI 等。AGPL-3.0 开源。
      </p>
      <div class="mt-8 flex flex-wrap justify-center gap-3">
        <RouterLink
          to="/register"
          class="inline-flex h-10 items-center rounded-md bg-[#2da44e] px-5 text-sm font-semibold text-white hover:bg-[#2c974b]"
        >
          快速开始
        </RouterLink>
        <a
          href="https://github.com/LiStudioorg/gitli"
          target="_blank"
          rel="noopener"
          class="inline-flex h-10 items-center rounded-md border border-[#d1d9e0] bg-white px-5 text-sm font-semibold text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3] dark:hover:bg-[#161b22]"
        >
          仓库主页
        </a>
      </div>
    </section>

    <section id="features" class="mx-auto grid max-w-[1100px] grid-cols-1 gap-4 px-4 pb-12 sm:grid-cols-2 md:px-6 lg:grid-cols-3">
      <FeatureCard
        v-for="f in features"
        :key="f.title"
        :icon="f.icon"
        :title="f.title"
        :description="f.lines.join(' / ')"
      />
    </section>

    <section id="about" class="border-t border-[#d1d9e0] py-10 text-center dark:border-[#3d444d]">
      <p class="text-sm text-[#1f2328] dark:text-[#e6edf3]">技术栈：Go 1.22+ / Vue 3 / SQLite / MySQL / PostgreSQL / Docker / OCI</p>
      <p class="mt-1.5 text-sm text-[#59636e]">gitli 是开源项目，采用 AGPL-3.0 许可证。</p>
      <nav class="mt-4 flex flex-wrap justify-center gap-4 text-sm">
        <a class="text-[#0969da] hover:underline" href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">文档</a>
        <a class="text-[#0969da] hover:underline" href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">部署指南</a>
        <a class="text-[#0969da] hover:underline" href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">贡献指南</a>
        <a class="text-[#0969da] hover:underline" href="https://www.gnu.org/licenses/agpl-3.0.html" target="_blank" rel="noopener">许可证</a>
      </nav>
    </section>

    <footer class="border-t border-[#d1d9e0] py-6 text-center text-xs text-[#59636e] dark:border-[#3d444d]">
      <p>© {{ new Date().getFullYear() }} gitli. All rights reserved.</p>
    </footer>
  </div>
</template>
