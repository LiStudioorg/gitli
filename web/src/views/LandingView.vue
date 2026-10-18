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
  <div class="landing">
    <TopNav variant="landing" />

    <section class="hero">
      <h1>轻量级自研 Git 服务平台，低内存，单二进制，Go + Vue3</h1>
      <p class="subtitle">
        支持 SSH/HTTP Git 协议、Issue/PR、TOTP、细粒度 PAT、
        包管理（NPM/Docker/Go/Maven）、CI 等。AGPL-3.0 开源。
      </p>
      <div class="cta">
        <RouterLink class="btn primary" to="/register">快速开始</RouterLink>
        <a class="btn" href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">GitHub 仓库</a>
      </div>
    </section>

    <section id="features" class="features">
      <FeatureCard
        v-for="f in features"
        :key="f.title"
        :icon="f.icon"
        :title="f.title"
        :description="f.lines.join(' / ')"
      />
    </section>

    <section id="about" class="about">
      <p class="stack">技术栈：Go 1.22+ / Vue 3 / SQLite / MySQL / PostgreSQL / Docker / OCI</p>
      <p class="license">gitli 是开源项目，采用 AGPL-3.0 许可证。</p>
      <nav class="doc-links">
        <a href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">文档</a>
        <a href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">部署指南</a>
        <a href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">贡献指南</a>
        <a href="https://www.gnu.org/licenses/agpl-3.0.html" target="_blank" rel="noopener">许可证</a>
      </nav>
    </section>

    <footer class="footer">
      <p>© {{ new Date().getFullYear() }} gitli. All rights reserved.</p>
    </footer>
  </div>
</template>

<style scoped>
.landing {
  max-width: 1200px;
  margin: 0 auto;
}
.hero {
  position: relative;
  text-align: center;
  padding: 5rem 1.5rem 3.5rem;
  background:
    radial-gradient(900px 320px at 50% -20%, var(--primary-soft), transparent 68%);
  border-radius: 0 0 28px 28px;
}
.hero h1 {
  font-size: clamp(1.8rem, 4vw, 2.6rem);
  line-height: 1.28;
  letter-spacing: -0.02em;
  margin: 0 0 1rem;
}
.subtitle {
  color: var(--muted);
  font-size: 1.1rem;
  max-width: 760px;
  margin: 0 auto 2rem;
  line-height: 1.6;
}
.cta {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
  flex-wrap: wrap;
}
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.65rem 1.6rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 0.98rem;
  font-weight: 600;
  color: var(--color-fg);
  background: var(--card);
  text-decoration: none;
  transition: background 0.15s ease, border-color 0.15s ease, transform 0.08s ease, box-shadow 0.15s ease;
}
.btn:hover {
  background: var(--hover-bg);
  border-color: color-mix(in srgb, var(--primary) 40%, var(--border));
}
.btn:active {
  transform: translateY(1px);
}
.btn.primary {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
  box-shadow: 0 4px 12px color-mix(in srgb, var(--primary) 35%, transparent);
}
.btn.primary:hover {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
  color: #fff;
}
.features {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
  padding: 1.5rem;
}
.features > * {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.15s ease, transform 0.15s ease, border-color 0.15s ease;
}
.features > *:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--primary) 35%, var(--border));
}
.about {
  text-align: center;
  padding: 2.5rem 1.5rem;
  border-top: 1px solid var(--border);
}
.stack {
  font-size: 1rem;
  margin-bottom: 0.5rem;
}
.license {
  color: var(--muted);
  margin-bottom: 1.25rem;
}
.doc-links {
  display: flex;
  gap: 1.5rem;
  justify-content: center;
  flex-wrap: wrap;
}
.footer {
  text-align: center;
  padding: 1.5rem;
  border-top: 1px solid var(--border);
  color: var(--muted);
  font-size: 0.9rem;
}
@media (max-width: 900px) {
  .features {
    grid-template-columns: repeat(2, 1fr);
  }
}
@media (max-width: 600px) {
  .features {
    grid-template-columns: 1fr;
  }
  .hero h1 {
    font-size: 1.8rem;
  }
}
</style>
