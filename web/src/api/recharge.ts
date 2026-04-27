import { http } from './http'

// 面向用户 + 管理员。端点拆成 user / admin 两组。

export interface Package {
  id: number
  name: string
  price_cny: number            // 分
  credits: number              // 厘
  bonus: number
  description: string
  sort: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface Order {
  id: number
  out_trade_no: string
  user_id: number
  package_id: number
  price_cny: number
  credits: number
  bonus: number
  channel: string
  pay_method: string
  status: 'pending' | 'paid' | 'expired' | 'cancelled' | 'failed'
  trade_no: string
  paid_at?: string | null
  pay_url?: string
  remark?: string
  created_at: string
  updated_at: string
}

export interface RedeemCode {
  id: number
  code: string
  credits: number
  batch_no: string
  remark: string
  created_by: number
  used_by?: number | null
  used_at?: string | null
  created_at: string
  updated_at: string
}

// ---------- user ----------

export function listMyPackages(): Promise<{ items: Package[]; enabled: boolean }> {
  return http.get('/api/recharge/packages')
}
export function createOrder(packageId: number, payType?: string): Promise<Order> {
  return http.post('/api/recharge/orders', { package_id: packageId, pay_type: payType })
}
export function listMyOrders(params: { status?: string; limit?: number; offset?: number } = {}):
  Promise<{ items: Order[]; total: number; limit: number; offset: number }> {
  return http.get('/api/recharge/orders', { params })
}
export function cancelMyOrder(id: number) {
  return http.post(`/api/recharge/orders/${id}/cancel`)
}
export function getMyRedeemStatus(): Promise<{ record?: RedeemCode | null }> {
  return http.get('/api/recharge/redeem-status')
}
export function redeemCode(code: string): Promise<RedeemCode> {
  return http.post('/api/recharge/redeem', { code })
}

// ---------- admin ----------

export function adminListPackages(): Promise<{ items: Package[]; total: number }> {
  return http.get('/api/admin/recharge/packages')
}
export function adminCreatePackage(p: Partial<Package>): Promise<Package> {
  return http.post('/api/admin/recharge/packages', p)
}
export function adminUpdatePackage(id: number, p: Partial<Package>): Promise<Package> {
  return http.patch(`/api/admin/recharge/packages/${id}`, p)
}
export function adminDeletePackage(id: number) {
  return http.delete(`/api/admin/recharge/packages/${id}`)
}
export function adminListOrders(params: {
  user_id?: number; status?: string; limit?: number; offset?: number
} = {}): Promise<{ items: Order[]; total: number; limit: number; offset: number }> {
  return http.get('/api/admin/recharge/orders', { params })
}
export function adminForcePaid(id: number, adminPassword: string) {
  return http.post(`/api/admin/recharge/orders/${id}/force-paid`, null, {
    headers: { 'X-Admin-Confirm': adminPassword },
  })
}
export function adminListRedeemCodes(params: {
  status?: 'used' | 'unused'
  batch_no?: string
  code?: string
  limit?: number
  offset?: number
} = {}): Promise<{ items: RedeemCode[]; total: number; limit: number; offset: number }> {
  return http.get('/api/admin/recharge/codes', { params })
}
export function adminGenerateRedeemCodes(payload: {
  count: number
  credits: number
  prefix?: string
  remark?: string
}): Promise<{ items: RedeemCode[]; batch_no: string; total: number }> {
  return http.post('/api/admin/recharge/codes/batch', payload)
}
export function adminDeleteRedeemCode(id: number) {
  return http.delete(`/api/admin/recharge/codes/${id}`)
}
