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
export const put = <T>(url: string, body?: unknown) =>
  request<T>(url, { method: 'PUT', body: body != null ? JSON.stringify(body) : undefined })

const maxImageRequests = 4
let activeImageRequests = 0
const pendingImageRequests: Array<() => void> = []

function runNextImageRequest() {
  while (activeImageRequests < maxImageRequests && pendingImageRequests.length > 0) {
    pendingImageRequests.shift()?.()
  }
}

function scheduleImageRequest<T>(request: () => Promise<T>, signal?: AbortSignal): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    let started = false
    let settled = false

    const rejectAbort = () => {
      if (settled) return
      settled = true
      reject(new DOMException('图片请求已取消', 'AbortError'))
    }
    const start = () => {
      if (settled || signal?.aborted) {
        rejectAbort()
        return
      }
      started = true
      activeImageRequests++
      request().then(resolve, reject).finally(() => {
        settled = true
        activeImageRequests--
        signal?.removeEventListener('abort', rejectAbort)
        runNextImageRequest()
      })
    }

    signal?.addEventListener('abort', rejectAbort, { once: true })
    if (signal?.aborted) {
      rejectAbort()
      return
    }
    pendingImageRequests.push(start)
    runNextImageRequest()

    // 已开始的请求由传入 fetch 的 signal 负责中止。
    if (started) signal?.removeEventListener('abort', rejectAbort)
  })
}

export async function getBlob(url: string, signal?: AbortSignal): Promise<Blob> {
  const headers: Record<string, string> = {}
  if (auth.token) headers.Authorization = `Bearer ${auth.token}`
  return scheduleImageRequest(async () => {
    const response = await fetch(url, { headers, credentials: 'include', signal })
    if (!response.ok) {
      if (response.status === 401) auth.logout()
      throw new ApiError(response.status, response.statusText || '图片加载失败')
    }
    return response.blob()
  }, signal)
}
