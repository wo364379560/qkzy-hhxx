import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import type { ApiResponse } from '@/types'
import { ADMIN_API_BASE_URL } from '@/config/runtime'

const request: AxiosInstance = axios.create({
  baseURL: ADMIN_API_BASE_URL,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json'
  }
})

function normalizeRequestUrl(url?: string) {
  if (!url) return ''
  const prefix = ADMIN_API_BASE_URL.replace(/\/$/, '')
  if (url.startsWith(prefix)) {
    return url.slice(prefix.length) || '/'
  }
  return url.startsWith('/api/v1/admin') ? url.slice('/api/v1/admin'.length) || '/' : url
}

function shouldSkipAuthRedirect(url?: string) {
  const normalized = normalizeRequestUrl(url)
  return normalized === '/login' || normalized === '/totp/bind' || normalized === '/totp/verify'
}

// Request interceptor: attach Authorization header
request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('admin_token')
    const hasAuthHeader = typeof config.headers?.get === 'function'
      ? !!config.headers.get('Authorization')
      : !!config.headers?.Authorization

    if (token && config.headers && !hasAuthHeader) {
      if (typeof config.headers.set === 'function') {
        config.headers.set('Authorization', `Bearer ${token}`)
      } else {
        config.headers['Authorization'] = `Bearer ${token}`
      }
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor: unwrap data or throw on non-zero code
request.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const res = response.data
    if (res.code !== 0) {
      const err = new Error(res.message || 'Request failed')
      return Promise.reject(err)
    }
    return response
  },
  (error) => {
    const persistentToken = localStorage.getItem('admin_token')
    const requestUrl = error.config?.url

    if (
      error.response?.status === 401 &&
      persistentToken &&
      !shouldSkipAuthRedirect(requestUrl)
    ) {
      localStorage.removeItem('admin_token')
      localStorage.removeItem('admin_user')
      window.location.href = '/login'
    }
    const msg = error.response?.data?.message || error.message || '请求失败'
    return Promise.reject(new Error(msg))
  }
)

export default request
