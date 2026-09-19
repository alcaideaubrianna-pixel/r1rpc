import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Badge, Button, Card, Flex, Grid, Heading, Progress, Text } from '@radix-ui/themes'
import { ArrowLeftIcon, ReloadIcon } from '@radix-ui/react-icons'
import { get, post } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { fmtTime } from '../lib/format'
import { notify } from '../lib/toast'
import type { SearchCandidatePage, SearchRequestDetail, SearchResponseRecord } from '../features/image-search/types'
import { formatScore, statusColor, statusLabel, terminalSearchStatuses } from '../features/image-search/status'
import { AuthenticatedImage } from '../features/image-search/AuthenticatedImage'
import { CandidateGallery } from '../features/image-search/CandidateGallery'
import { XHSProfileDialog } from '../features/image-search/XHSProfileDialog'

export default function ImageSearchDetailPage() {
  const { id = '' } = useParams()
	const [retrying, setRetrying] = useState(false)
  const detail = useFetch(() => get<SearchRequestDetail>(`/api/v1/image-search/requests/${encodeURIComponent(id)}`), [id])
  const candidates = useFetch(() => get<SearchCandidatePage>(`/api/v1/image-search/requests/${encodeURIComponent(id)}/candidates?page=1&pageSize=100`), [id])
  const responses = useFetch(() => get<{ items: SearchResponseRecord[] }>(`/api/v1/image-search/requests/${encodeURIComponent(id)}/responses`), [id])

  useEffect(() => {
    if (detail.data && terminalSearchStatuses.has(detail.data.request.status)) return
    const timer = window.setInterval(() => {
      detail.reload()
      candidates.reload()
      responses.reload()
    }, 3000)
    return () => window.clearInterval(timer)
  }, [detail.data?.request.status])

  const request = detail.data?.request
  const groups = detail.data?.groups ?? []
  const items = detail.data?.items ?? []
  const candidateItems = candidates.data?.items ?? []
  const completedGroups = request ? request.matchedCount + request.notMatchedCount + request.failedCount : 0
  const progress = request?.groupCount ? Math.round(completedGroups / request.groupCount * 100) : 0

	async function retryAnalysis() {
		setRetrying(true)
		try {
			await post(`/api/v1/image-search/requests/${encodeURIComponent(id)}/retry-analysis`)
			notify.success('已使用保存的设备响应重新分析')
			detail.reload(); candidates.reload(); responses.reload()
		} catch (error) {
			notify.error(error, '重新分析失败')
		} finally {
			setRetrying(false)
		}
	}

  return (
    <Flex direction="column" gap="4" className="image-search-detail">
      <Flex justify="between" align="center" wrap="wrap" gap="3">
        <Flex align="center" gap="3"><Button asChild variant="soft" color="gray"><Link to="/image-search"><ArrowLeftIcon /> 返回任务</Link></Button><div><Heading size="4">任务结果</Heading><Text as="div" size="1" color="gray">{id}</Text></div></Flex>
		<Flex gap="2"><Button loading={retrying} variant="soft" onClick={retryAnalysis}>重新分析</Button><Button variant="soft" color="gray" onClick={() => { detail.reload(); candidates.reload(); responses.reload() }}><ReloadIcon /> 刷新</Button></Flex>
      </Flex>

      {request && <Card size="3" className="result-hero">
        <Grid columns={{ initial: '1', sm: '4' }} gap="5" align="center">
          <div><Text size="1" color="gray">当前状态</Text><Flex align="center" gap="2" mt="1"><span className={`status-beacon status-${request.status}`} /><Heading size="4">{statusLabel(request.status)}</Heading></Flex></div>
          <Metric label="图片组" value={String(request.groupCount)} />
          <Metric label="匹配成功" value={String(request.matchedCount)} accent="green" />
          <Metric label="未匹配 / 失败" value={`${request.notMatchedCount} / ${request.failedCount}`} />
        </Grid>
        <Flex align="center" gap="3" mt="5"><Progress value={progress} style={{ flex: 1 }} /><Text size="2" color="gray">{progress}%</Text></Flex>
        <Text as="div" size="1" color="gray" mt="2">创建于 {fmtTime(request.createdAt)} · 算法 {request.pipelineName}/{request.pipelineVersion}</Text>
      </Card>}

      <Flex direction="column" gap="4">
        {groups.map((group, groupIndex) => {
          const groupItems = items.filter((item) => item.groupId === group.id)
          const groupCandidates = candidateItems.filter((candidate) => groupItems.some((item) => item.id === candidate.searchItemId))
		  const downloadTotal = groupCandidates.reduce((sum, candidate) => sum + candidate.imageCount, 0)
		  const downloadDone = groupCandidates.reduce((sum, candidate) => sum + candidate.analyzedImages + candidate.failedImages, 0)
          return <Card key={group.id} size="3" className={group.status === 'matched' ? 'matched-group' : undefined}>
            <Flex justify="between" align="start" gap="3" wrap="wrap">
              <Flex gap="3" align="center"><span className="group-index">{String(groupIndex + 1).padStart(2, '0')}</span><div><Flex gap="2" align="center"><Heading size="3">{group.subjectUserId || group.externalId || '未命名图片组'}</Heading><Badge color={statusColor(group.status)}>{statusLabel(group.status)}</Badge></Flex><Text size="1" color="gray">{group.itemCount} 张图片 · 完成 {group.completedCount} · 失败 {group.failedCount}</Text></div></Flex>
              <div className="score-display"><Text size="1" color="gray">最佳相似度</Text><Text as="div" size="6" weight="bold">{formatScore(group.bestScore)}</Text></div>
            </Flex>
			{downloadTotal > 0 && <Flex align="center" gap="3" mt="3"><Progress value={downloadDone / downloadTotal * 100} style={{ flex: 1 }} /><Text size="1" color="gray">下载分析 {downloadDone}/{downloadTotal}</Text></Flex>}

            <Grid columns={{ initial: '1', md: '2' }} gap="4" mt="5">
              <div><Text size="1" weight="bold" color="gray">提交图片</Text><Grid columns={{ initial: '2', sm: '3' }} gap="3" mt="2">{groupItems.map((item) => <div key={item.id} className="source-image-card"><AuthenticatedImage url={item.fileUrl} alt="提交图片" className="source-image-preview" previewTitle={`提交图片 #${item.ordinal + 1}`} /><Flex justify="between" align="center" p="2"><Badge size="1" color={statusColor(item.status)}>{statusLabel(item.status)}</Badge><Text size="1" color="gray">#{item.ordinal + 1}</Text></Flex>{item.errorMessage && <Text as="div" size="1" color="red" style={{ padding: '0 8px 8px' }}>{item.errorMessage}</Text>}</div>)}</Grid></div>
              <div><Text size="1" weight="bold" color="gray">候选结果</Text><Flex direction="column" gap="2" mt="2">{groupCandidates.map((candidate) => <div key={candidate.id} className={`candidate-row ${candidate.matched ? 'candidate-matched' : ''}`}>
                <CandidateGallery candidate={candidate} />
                <div className="candidate-copy"><Flex align="center" gap="2"><Text weight="medium" truncate>{candidate.title || candidate.contentId}</Text>{candidate.matched && <Badge color="green">MATCH</Badge>}</Flex><Flex align="center" gap="1"><XHSProfileDialog candidate={candidate} /><Text size="1" color="gray">· {candidate.authorId || '无用户 ID'}</Text></Flex><Text as="div" size="1" color="gray">图片进度 {candidate.analyzedImages}/{candidate.imageCount}{candidate.failedImages > 0 ? ` · 失败 ${candidate.failedImages}` : ''}</Text><Text as="div" size="1" color="gray">pHash {candidate.phashDistance ?? '—'} · dHash {candidate.dhashDistance ?? '—'} · aHash {candidate.ahashDistance ?? '—'}</Text>{candidate.downloadStatus === 'failed' && <Text as="div" size="1" color="red">{candidate.errorMessage || '候选图片下载失败'}</Text>}</div>
                <div className="candidate-score"><Text size="5" weight="bold">{formatScore(candidate.score)}</Text><Text as="div" size="1" color="gray">排名 #{candidate.rank + 1}</Text></div>
              </div>)}{groupCandidates.length === 0 && <div className="empty-result"><Text color="gray">{terminalSearchStatuses.has(request?.status ?? '') ? '没有可展示的候选结果' : '等待设备搜索和图片分析…'}</Text></div>}</Flex></div>
            </Grid>
          </Card>
        })}
      </Flex>

      <Card size="3">
        <Flex justify="between" align="center" mb="3"><div><Heading size="3">设备搜索响应</Heading><Text size="1" color="gray">分别保留标准化模型与小红书原始响应</Text></div><Badge variant="soft">{responses.data?.items.length ?? 0} 条响应</Badge></Flex>
        <Flex direction="column" gap="3">{(responses.data?.items ?? []).map((response, index) => <details key={response.id} className="response-json" open={index === 0}>
          <summary><span>Job {response.jobId}</span><Badge size="1" color={response.parseStatus === 'failed' ? 'red' : 'blue'}>{response.parseStatus}</Badge><Text size="1" color="gray">{response.itemCount} 条候选 · {fmtTime(response.createdAt)}</Text></summary>
          {response.errorMessage && <Text as="div" color="red" size="2" mb="2">{response.errorMessage}</Text>}
          <Text as="div" size="1" weight="bold" mb="1">小红书原始 JSON</Text>
          <pre>{response.rawJson ? prettyJSON(response.rawJson) : '当前设备版本未返回原始响应'}</pre>
          <Text as="div" size="1" weight="bold" mt="3" mb="1">标准化模型</Text>
          <pre>{prettyJSON(response.normalizedJson)}</pre>
        </details>)}{(responses.data?.items.length ?? 0) === 0 && <div className="empty-result"><Text color="gray">设备尚未返回搜索响应</Text></div>}</Flex>
      </Card>
    </Flex>
  )
}

function Metric({ label, value, accent }: { label: string; value: string; accent?: string }) {
  return <div><Text size="1" color="gray">{label}</Text><Text as="div" size="6" weight="bold" color={accent === 'green' ? 'green' : undefined}>{value}</Text></div>
}

function prettyJSON(raw: string) {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}
