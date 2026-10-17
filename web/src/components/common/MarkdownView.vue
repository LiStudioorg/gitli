<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'
import type { Options } from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import json from 'highlight.js/lib/languages/json'
import go from 'highlight.js/lib/languages/go'
import bash from 'highlight.js/lib/languages/bash'
import yaml from 'highlight.js/lib/languages/yaml'
import markdown from 'highlight.js/lib/languages/markdown'
import python from 'highlight.js/lib/languages/python'
import sql from 'highlight.js/lib/languages/sql'

hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('css', css)
hljs.registerLanguage('json', json)
hljs.registerLanguage('go', go)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('shell', bash)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('markdown', markdown)
hljs.registerLanguage('python', python)
hljs.registerLanguage('sql', sql)

const props = defineProps<{ source: string }>()

const options: Options = {
  html: false,
  linkify: true,
  breaks: false,
  highlight(str: string, lang: string): string {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs"><code>${hljs.highlight(str, { language: lang, ignoreIllegals: true }).value}</code></pre>`
      } catch {
        /* fall through */
      }
    }
    return `<pre class="hljs"><code>${MarkdownIt().utils.escapeHtml(str)}</code></pre>`
  },
}

const md = new MarkdownIt(options)

const html = computed(() => md.render(props.source || ''))
</script>

<template>
  <div class="markdown-body" v-html="html" />
</template>

<style scoped>
.markdown-body {
  font-size: 0.92rem;
  line-height: 1.7;
  word-wrap: break-word;
}
.markdown-body :deep(h1),
.markdown-body :deep(h2) {
  border-bottom: 1px solid var(--border);
  padding-bottom: 0.3em;
}
.markdown-body :deep(a) {
  color: var(--primary);
}
.markdown-body :deep(pre) {
  background: color-mix(in srgb, var(--border) 35%, transparent);
  padding: 0.75rem;
  border-radius: 6px;
  overflow-x: auto;
}
.markdown-body :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.85em;
}
.markdown-body :deep(blockquote) {
  margin: 0;
  padding-left: 1em;
  border-left: 3px solid var(--border);
  color: var(--muted);
}
.markdown-body :deep(table) {
  border-collapse: collapse;
}
.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid var(--border);
  padding: 0.35rem 0.6rem;
}
</style>
