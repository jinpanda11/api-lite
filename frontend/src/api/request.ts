// Re-export the configured axios instance for direct use in admin pages.
import axios from 'axios'
import { Toast } from '@douyinfe/semi-ui'
import { useAppStore } from '../store'

const request = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

request.interceptors.request.use((config) => {
  const token = useAppStore.getState().token
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

request.interceptors.response.use(
  (res) => res,
  (err) => {
    const msg = err.response?.data?.error || err.response?.data?.message || err.message || 'Request failed'
    if (err.response?.status === 401) {
      useAppStore.getState().logout()
      window.location.href = '/login'
    } else {
      Toast.error(msg)
    }
    return Promise.reject(err)
  }
)

export default request
