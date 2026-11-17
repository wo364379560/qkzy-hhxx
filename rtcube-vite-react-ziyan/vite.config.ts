import http from 'http'
import https from 'https'
import path from 'path'
import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'

/** 开发代理默认使用用户指定的 API。 */
const API_TARGET = process.env.VITE_DEV_API_TARGET || 'https://api.prfgsu.com'

function isAllowedCosHost(hostname: string) {
  return /(\.myqcloud\.com|\.tencentcos\.cn)$/i.test(hostname)
}

/** 开发时把 COS PUT 转到同源代理，避免桶 CORS 未放行 localhost:5175 */
function cosPutProxy(): Plugin {
  return {
    name: 'cos-put-proxy',
    configureServer(server) {
      server.middlewares.use('/__cos_put', (req, res) => {
        if (req.method === 'OPTIONS') {
          res.statusCode = 204
          res.end()
          return
        }
        if (req.method !== 'PUT') {
          res.statusCode = 405
          res.end()
          return
        }
        const raw = String(req.headers['x-cos-put-url'] || '')
        let target: URL
        try {
          target = new URL(raw)
        } catch {
          res.statusCode = 400
          res.end('missing cos url')
          return
        }
        if (!isAllowedCosHost(target.hostname)) {
          res.statusCode = 400
          res.end('forbidden host')
          return
        }
        const lib = target.protocol === 'https:' ? https : http
        const headers: Record<string, string | number> = {}
        const contentType = req.headers['content-type']
        const contentLength = req.headers['content-length']
        if (contentType) headers['Content-Type'] = String(contentType)
        if (contentLength) headers['Content-Length'] = String(contentLength)
        const proxyReq = lib.request(
          {
            method: 'PUT',
            hostname: target.hostname,
            port: target.port || undefined,
            path: `${target.pathname}${target.search}`,
            headers,
          },
          (proxyRes) => {
            res.statusCode = proxyRes.statusCode || 502
            proxyRes.pipe(res)
          },
        )
        proxyReq.on('error', (err) => {
          if (!res.headersSent) {
            res.statusCode = 502
            res.end(err.message || 'cos proxy failed')
          }
        })
        req.pipe(proxyReq)
      })
    },
  }
}

export default defineConfig({
  base: './',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  /** 电报贴纸 gzip lottie，按静态资源处理，勿当 JS 解析 */
  assetsInclude: ['**/*.tgs'],
  plugins: [react(), cosPutProxy()],
  server: {
    port: 5175,
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
        secure: true,
      },
      '/ping': {
        target: API_TARGET,
        changeOrigin: true,
        secure: true,
      },
    },
  },
})
