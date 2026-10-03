export const terminalSearchStatuses = new Set(['completed', 'partial_failed', 'failed', 'cancelled'])

export function statusLabel(status: string) {
  const labels: Record<string, string> = {
    created: '已创建', pending: '等待处理', preparing: '准备中', queued: '排队中', running: '执行中', analyzing: '分析中',
	passed: '已通过', blocked: '已屏蔽', waiting: '等待下轮', downloading: '下载中', downloaded: '已下载',
    search_completed: '搜索完成', matched: '已匹配', not_matched: '未匹配',
    completed: '已完成', partial_failed: '部分失败', retry_wait: '等待重试',
    failed: '失败', filtered: '已过滤', cancelled: '已取消',
  }
  return labels[status] ?? status
}

export function statusColor(status: string): 'green' | 'red' | 'amber' | 'blue' | 'gray' {
  if (status === 'matched' || status === 'completed') return 'green'
  if (status === 'failed' || status === 'partial_failed') return 'red'
  if (status === 'not_matched' || status === 'filtered' || status === 'cancelled') return 'gray'
  if (status === 'retry_wait') return 'amber'
  return 'blue'
}

export function formatScore(score?: number) {
  return score == null ? '—' : `${(score * 100).toFixed(1)}%`
}
