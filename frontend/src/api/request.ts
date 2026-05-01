// Re-export the configured axios instance for direct use in admin pages.
import axios from 'axios'
import { Toast } from '@douyinfe/semi-ui'

const request = axios.create({
  baseURL: '/api',
  timeout: 120000,
})

request.interceptors.request.use((config) => {
  // Cookie "auth_token" is sent automatically by the browser (HttpOnly).
  // No need to manually attach Authorization header for the SPA.
  return config
})

request.interceptors.response.use(
  (res) => res,
  (err) => {
    const msg = err.response?.data?.error || err.response?.data?.message || err.message || 'Request failed'
    if (err.response?.status === 401) {
      // Don't auto-redirect — Layout handles auth state gracefully
    } else {
      Toast.error(msg)
    }
    return Promise.reject(err)
  }
)

export default request
