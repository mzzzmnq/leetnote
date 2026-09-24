import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    // 开发环境【不】使用 Vite 代理，而是让浏览器直连后端（:8080）。
    // 目的是真实验证后端的 CORS + Cookie 配置是否正确——
    // 用代理会掩盖跨域问题，等到上线才发现就晚了。
    // 生产环境前后端同源（Nginx 分流），届时不存在跨域。
  },
})
