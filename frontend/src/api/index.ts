import request from './request'

// ── Auth ───────────────────────────────────────────────────────────────────

export const login = (username: string, password: string) =>
  request.post('/user/login', { username, password })

export const register = (data: {
  username: string
  email: string
  password: string
  code: string
}) => request.post('/user/register', data)

export const sendVerificationCode = (email: string) =>
  request.post('/user/email/code', { email })

export const getEmailVerificationStatus = () =>
  request.get('/settings/email-verification')

// ── User ──────────────────────────────────────────────────────────────────

export const getUserInfo = () => request.get('/user/info')

export const logout = () => request.post('/user/logout')

export const updatePassword = (old_password: string, new_password: string) =>
  request.post('/user/update-password', { old_password, new_password })

// ── Dashboard ─────────────────────────────────────────────────────────────

export const getDashboard = () => request.get('/dashboard')

// ── Tokens ────────────────────────────────────────────────────────────────

export const listTokens = () => request.get('/token')

export const createToken = (data: {
  name: string
  remark?: string
  expired_at?: string | null
}) => request.post('/token', data)

export const updateToken = (
  id: number,
  data: { name?: string; remark?: string; status?: number; expired_at?: string | null }
) => request.put(`/token/${id}`, data)

export const deleteToken = (id: number) => request.delete(`/token/${id}`)

// ── Channels (admin) ──────────────────────────────────────────────────────

export const listChannels = () => request.get('/channel')

export const createChannel = (data: object) => request.post('/channel', data)

export const updateChannel = (id: number, data: object) =>
  request.put(`/channel/${id}`, data)

export const deleteChannel = (id: number) => request.delete(`/channel/${id}`)

export const testChannel = (id: number) => request.post('/channel/test', { id })

// ── Models ────────────────────────────────────────────────────────────────

export const listModels = () => request.get('/models')

// ── Notices ───────────────────────────────────────────────────────────────

export const getNotices = () => request.get('/notice')

export const listNotices = () => request.get('/admin/notice')

export const createNotice = (data: object) => request.post('/admin/notice', data)

export const updateNotice = (id: number, data: object) =>
  request.put(`/admin/notice/${id}`, data)

export const deleteNotice = (id: number) => request.delete(`/admin/notice/${id}`)

// ── Logs ──────────────────────────────────────────────────────────────────

export const getLogs = (params?: object) => request.get('/log', { params })

export const getDailyCosts = () => request.get('/admin/daily-costs')

// ── Wallet ────────────────────────────────────────────────────────────────

export const getBalance = () => request.get('/balance')

export const redeemCode = (code: string) => request.post('/redeem', { code })

export const getTopupLogs = () => request.get('/topup/logs')

// ── Check-in ──────────────────────────────────────────────────────────────

export const checkIn = () => request.post('/checkin')

export const getCheckInStatus = () => request.get('/checkin/status')

// ── Admin: Users ──────────────────────────────────────────────────────────

export const listUsers = (params?: object) => request.get('/admin/user', { params })

export const updateUserStatus = (id: number, data: object) =>
  request.put(`/admin/user/${id}`, data)

// ── Branding (public) ────────────────────────────────────────────────────────

export const getBranding = () => request.get('/settings/branding')

// ── Settings (admin) ────────────────────────────────────────────────────────

export const getSettings = () => request.get('/admin/settings')

export const updateSettings = (settings: Record<string, string>) =>
  request.put('/admin/settings', { settings })

// ── Status Monitoring ─────────────────────────────────────────────────────

export const getStatus = () => request.get('/status')
export const refreshStatus = () => request.post('/status')

export const toggleChannelMonitor = (id: number, monitor_enabled: boolean) =>
  request.put(`/channel/${id}/monitor`, { monitor_enabled })

export const getMonitorConfig = () => request.get('/admin/monitor-config')

export const updateMonitorConfig = (interval: number) =>
  request.put('/admin/monitor-config', { interval })

// ── Audit Log (admin) ──────────────────────────────────────────────────────

export const getAuditLogs = (params?: object) => request.get('/admin/audit', { params })

// ── Model Pricing (admin) ──────────────────────────────────────────────────

export const listModelPricing = () => request.get('/admin/model-pricing')

export const updateModelPricing = (modelName: string, data: object) =>
  request.put(`/admin/model-pricing/${encodeURIComponent(modelName)}`, data)

// ── Draw (image generation) ───────────────────────────────────────────────

export const generateImage = (data: {
  model: string
  prompt: string
  size?: string
  quality?: string
}) => request.post('/draw', data)

export const getDrawQuota = () => request.get('/draw/quota')
