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
}

export interface TopupLog {
  id: number
  user_id: number
  amount: number
  code: string
  remark: string
  created_at: string
}
