// 与后端 internal/model 对齐的 DTO

export interface User {
  id: number
  username: string
  role: string
  enabled: boolean
  notes: string
  lastLoginAt?: string
  createdAt: string
  updatedAt: string
}

export interface GroupInfo {
  group: string
  displayName: string
  enabled: boolean
  deviceKey: string
  authMode: string
  apiKey: string
  notes: string
  totalDevices: number
  onlineDevices: number
  requests7d: number
  success7d: number
  lastSeenAt?: string
  lastRequestAt?: string
  status: string
  statusLabel: string
  successRate: number
  createdAt: string
  updatedAt: string
}

export interface Device {
  id: number
  clientId: string
  group: string
  platform: string
  status: string
  lastSeenAt: string
  lastIp: string
  extraJson: string
  deviceMachine: string
  deviceModel: string
  osName: string
  osVersion: string
  sdkName: string
  sdkVersion: string
  actions: string[]
  createdAt: string
  updatedAt: string
}

export interface StoredFile {
  id: string
  originalName: string
  contentType: string
  sizeBytes: number
  sha256: string
  status: string
  createdAt: string
}

export interface ActionInput {
  path: string
  type: string
  label: string
  required: boolean
  accept?: string[]
}

export interface ActionDefinition {
  name: string
  payloadTemplate: Record<string, unknown>
  inputs: ActionInput[]
}

export interface RpcRequest {
  id: number
  requestId: string
  group: string
  action: string
  clientId: string
  requesterUserId?: number
  requestPayload: string
  responsePayload: string
  status: string
  httpCode: number
  latencyMs: number
  errorMessage: string
  createdAt: string
  finishedAt?: string
}

export interface RPCRequestPage {
  items: RpcRequest[]
  page: number
  pageSize: number
  total: number
  totalPages: number
}

export interface TrendPoint {
  statDate: string
  totalRequests: number
  successRequests: number
  failedRequests: number
  timeoutRequests: number
  avgLatencyMs: number
  maxLatencyMs: number
  successRate: number
}

export interface RequestFilterOptions {
  groups: string[]
  actions: string[]
  clientIds: string[]
}

export interface ClientQueueItem {
  clientId: string
  group: string
  platform: string
  status: string
  lastSeenAt: string
  lastIp: string
  pendingCount: number
  inFlight: number
  maxInFlight: number
}
