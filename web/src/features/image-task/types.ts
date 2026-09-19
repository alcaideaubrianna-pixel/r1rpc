export interface ImageBatch {
  id: string
  status: string
  total: number
  queued: number
  running: number
  completed: number
  failed: number
  cacheHit: number
  createdAt: string
}

export interface ImageBatchPage {
  items: ImageBatch[]
  page: number
  pageSize: number
  total: number
}
