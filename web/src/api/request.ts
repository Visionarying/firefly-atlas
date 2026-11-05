import axios, { type AxiosRequestConfig } from 'axios'

const TOKEN_KEY = 'firefly_token'

export const getToken = () => localStorage.getItem(TOKEN_KEY) || ''
export const setToken = (token: string) => localStorage.setItem(TOKEN_KEY, token)
export const clearToken = () => localStorage.removeItem(TOKEN_KEY)

const instance = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

instance.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers = config.headers || {}
    config.headers.token = token
  }
  return config
})

instance.interceptors.response.use(
  (response) => {
    const body = response.data
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) return body.data
      const error = new Error(body.msg || '请求失败')
      ;(error as any).code = body.code
      if (body.code === 401 || body.code === -1) {
        // The backend returns HTTP 401 for invalid tokens (handled below);
        // code -1 here is a business error and should not log the user out.
      }
      return Promise.reject(error)
    }
    return body
  },
  (error) => {
    if (error.response?.status === 401) {
      clearToken()
      window.dispatchEvent(new CustomEvent('firefly:unauthorized'))
      return Promise.reject(new Error('登录已过期，请重新登录'))
    }
    const msg = error.response?.data?.msg || error.message || '网络错误'
    return Promise.reject(new Error(msg))
  },
)

type Http = {
  get<T = any>(url: string, config?: AxiosRequestConfig): Promise<T>
  post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T>
  delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<T>
}

const http = instance as unknown as Http
export const rawHttp = instance
export default http
