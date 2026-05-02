export interface UserInfo {
  id: number
  username: string
  email: string
  role: 'user' | 'admin'
  balance: number
  token_count?: number
  stats?: DashboardStats
}

export interface DashboardStats {
  today_requests: number
  total_requests: number
  today_cost: number
  total_cost: number
}

export interface ModelRank {
  model: string
  count: number
}

export interface DashboardData {
  stats: DashboardStats
  trend: DailyCount[]
  token_count: number
  balance: number
  total_users?: number
  active_channels?: number
  sys_stats?: DashboardStats
  top_models?: ModelRank[]
}

export interface DailyCount {
  date: string
  count: number
}

export interface Token {
  id: number
  user_id: number
  key: string
  name: string
  remark: string
  expired_at: string | null
  status: number
  created_at: string
}

export interface Channel {
  id: number
  name: string
  type: string
  base_url: string
  api_key: string
  models: string
  priority: number
  status: number
  fixed_path?: string
  created_at: string
}

export interface Log {
  id: number
  user_id: number
  token_name: string
  channel_name: string
  model: string
  input_tokens: number
  output_tokens: number
  cost: number
  status: number
  request_path: string
  created_at: string
}

export interface ModelInfo {
  id: string
  channel_name: string
  input_price: number
  output_price: number
  billing_mode?: string
  call_price?: number
  icon_url?: string
}

export interface TopupLog {
  id: number
  user_id: number
  amount: number
  code: string
  remark: string
  created_at: string
}

export interface CheckInStatus {
  checked_in_today: boolean
  streak: number
  today_reward: number
}

export interface AuditLog {
  id: number
  admin_name: string
  admin_id: number
  action: string
  detail: string
  created_at: string
}

export interface AdminUser {
  id: number
  username: string
  email: string
  role: string
  balance: number
  status: number
  price_multiplier: number
  created_at: string
}
