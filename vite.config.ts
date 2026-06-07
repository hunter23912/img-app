import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // 开发阶段把前端的 /api 请求转发到 Go 后端，避免前端代码里写死后端端口。
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
