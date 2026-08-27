import { auth } from '../auth'

/** 后端统一响应信封 */
export interface Envelope<T = unknown> {
  success: boolean
  msg: string
  data: T
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public detail: string,
    public body?: unknown,
  ) {
    super(`HTTP ${status}: ${detail}`)
    this.name = 'ApiError'
  }
}

function isEnvelope(v: unknown): v is Envelope {
  return !!v && typeof v === 'object' && 'success' in v && 'data' in v
}

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...(options.headers as Record<string, string>),
  }
  if (options.body && typeof options.body === 'string') {
    headers['Content-Type'] = 'application/json'
  }
  if (auth.token) {
    headers['Authorization'] = `Bearer ${auth.token}`
  }

  const res = await fetch(url, { ...options, headers, credentials: 'include' })

  if (res.status === 204) return undefined as T

  let body: unknown = null
  try {
    body = await res.json()
  } catch {
    body = null
  }

  const env = isEnvelope(body) ? body : null
  const ok = res.ok && (env ? env.success : true)

  if (!ok) {
    const msg = env?.msg || res.statusText || '请求失败'
    const isLoginCall = url.includes('/api/auth/login')
    if (res.status === 401 && !isLoginCall) {
      auth.logout()
      if (!location.hash.startsWith('#/login')) location.hash = '#/login'
    }
    throw new ApiError(res.status, msg, body)
  }

  return (env ? (env.data as T) : (body as T))
}

export const get = <T>(url: string) => request<T>(url)
export const post = <T>(url: string, body?: unknown) =>
  request<T>(url, { method: 'POST', body: body != null ? JSON.stringify(body) : undefined })
export const postWithHeaders = <T>(url: string, body: unknown, headers: Record<string, string>) =>
  request<T>(url, { method: 'POST', headers, body: JSON.stringify(body) })
export const upload = <T>(url: string, body: FormData) => request<T>(url, { method: 'POST', body })
export const patch = <T>(url: string, body?: unknown) =>
  request<T>(url, { method: 'PATCH', body: body != null ? JSON.stringify(body) : undefined })
export const del = <T>(url: string) => request<T>(url, { method: 'DELETE' })
