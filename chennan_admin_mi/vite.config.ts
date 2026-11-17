import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

const devProxyTarget = process.env.VITE_DEV_PROXY_TARGET || 'https://api.ruizhihe.com'

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({ resolvers: [ElementPlusResolver()] }),
    Components({ resolvers: [ElementPlusResolver()] }),
  ],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': { target: devProxyTarget, changeOrigin: true }
    }
  },
  preview: {
    port: 4173,
    proxy: {
      '/api': { target: devProxyTarget, changeOrigin: true }
    }
  },
  build: {
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('/src/views/im/') || id.includes('/src/components/im/')) {
            return 'im-tools'
          }

          if (id.includes('/src/views/SettingsView.vue') || id.includes('/src/components/BackupCodesDialog.vue')) {
            return 'security-tools'
          }

          if (id.includes('/node_modules/vue/') || id.includes('/node_modules/vue-router/') || id.includes('/node_modules/pinia/')) {
            return 'vue-vendor'
          }

          if (id.includes('/node_modules/axios/')) {
            return 'axios'
          }

          if (id.includes('/node_modules/@element-plus/icons-vue/')) {
            return 'element-icons'
          }

          if (id.includes('/node_modules/element-plus/')) {
            return 'element-plus'
          }

          if (id.includes('/node_modules/qrcode.vue/')) {
            return 'qrcode'
          }
        },
      }
    }
  }
})
