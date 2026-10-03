import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Badge, Button, Card, Flex, Grid, Heading, Progress, Tabs, Text } from '@radix-ui/themes'
import { ArrowLeftIcon, ReloadIcon } from '@radix-ui/react-icons'
import { get, post } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { fmtTime } from '../lib/format'
import { notify } from '../lib/toast'
import type { SearchCandidate, SearchCandidatePage, SearchItem, SearchRequestDetail, SearchResponseRecord, SourceImage } from '../features/image-search/types'
import { formatScore, statusColor, statusLabel, terminalSearchStatuses } from '../features/image-search/status'
import { AuthenticatedImage } from '../features/image-search/AuthenticatedImage'
import { CandidateGallery } from '../features/image-search/CandidateGallery'
import { XHSProfileDialog } from '../features/image-search/XHSProfileDialog'

export default function ImageSearchDetailPage() {
  const { id = '' } = useParams()
	const [retrying, setRetrying] = useState(false)
  const detail = useFetch(() => get<SearchRequestDetail>(`/api/v1/image-search/requests/${encodeURIComponent(id)}`), [id])
  const candidates = useFetch(() => getAllCandidates(id), [id])
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
  const sourceImages = detail.data?.sourceImages ?? []
  const candidateItems = candidates.data?.items ?? []
  const completedGroups = request ? request.matchedCount + request.notMatchedCount + request.failedCount : 0
  const progress = request?.groupCount ? Math.round(completedGroups / request.groupCount * 100) : 0

	async function retryAnalysis() {
		setRetrying(true)
		try {
			await post(`/api/v1/image-search/requests/${encodeURIComponent(id)}/retry-analysis`)
			notify.success('已重新提交设备搜索并获取新图片链接')
			detail.reload(); candidates.reload(); responses.reload()
		} catch (error) {
			notify.error(error, '重新搜索失败')
		} finally {
			setRetrying(false)
		}
	}

  return (
    <Flex direction="column" gap="4" className="image-search-detail">
      <Flex justify="between" align="center" wrap="wrap" gap="3">
        <Flex align="center" gap="3"><Button asChild variant="soft" color="gray"><Link to="/image-search"><ArrowLeftIcon /> 返回任务</Link></Button><div><Heading size="4">任务结果</Heading><Text as="div" size="1" color="gray">{id}</Text></div></Flex>
		<Flex gap="2"><Button loading={retrying} variant="soft" onClick={retryAnalysis}>重新搜索</Button><Button variant="soft" color="gray" onClick={() => { detail.reload(); candidates.reload(); responses.reload() }}><ReloadIcon /> 刷新</Button></Flex>
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
		  const groupSourceImages = sourceImages.filter((image) => image.groupId === group.id)
		  const displayedImages = groupSourceImages.length > 0 ? groupSourceImages : groupItems.map(sourceImageFromItem)
		  const groupCandidates = candidateItems.filter((candidate) => groupItems.some((item) => item.id === candidate.searchItemId))
		  const downloadTotal = groupCandidates.reduce((sum, candidate) => sum + candidate.imageCount, 0)
		  const downloadDone = groupCandidates.reduce((sum, candidate) => sum + candidate.analyzedImages + candidate.failedImages, 0)
          return <Card key={group.id} size="3" className={group.status === 'matched' ? 'matched-group' : undefined}>
            <Flex justify="between" align="start" gap="3" wrap="wrap">
              <Flex gap="3" align="center"><span className="group-index">{String(groupIndex + 1).padStart(2, '0')}</span><div><Flex gap="2" align="center"><Heading size="3">{group.subjectUserId || group.externalId || '未命名图片组'}</Heading><Badge color={statusColor(group.status)}>{statusLabel(group.status)}</Badge></Flex><Text size="1" color="gray">{group.itemCount} 张图片 · 完成 {group.completedCount} · 失败 {group.failedCount}</Text></div></Flex>
              <div className="score-display"><Text size="1" color="gray">最佳相似度</Text><Text as="div" size="6" weight="bold">{formatScore(group.bestScore)}</Text></div>
            </Flex>
			{downloadTotal > 0 && <Flex align="center" gap="3" mt="3"><Progress value={downloadDone / downloadTotal * 100} style={{ flex: 1 }} /><Text size="1" color="gray">下载分析 {downloadDone}/{downloadTotal}</Text></Flex>}

			<Tabs.Root className="source-results" defaultValue={displayedImages[0]?.id}>
			  <Tabs.List className="source-tabs">{displayedImages.map((image, imageIndex) => {
				const bestScore = bestCandidateScore(groupCandidates, image.searchItemId)
				return <Tabs.Trigger key={image.id} value={image.id}>图片 #{(image.imageIndex ?? imageIndex) + 1}<Text size="1" color="gray">{shortID(image.id)}</Text>{bestScore >= 0.9 && <Badge size="1" color="green">命中</Badge>}</Tabs.Trigger>
			  })}</Tabs.List>
			  {displayedImages.map((image, imageIndex) => {
				const imageCandidates = groupCandidates.filter((candidate) => candidate.searchItemId === image.searchItemId).sort((a, b) => b.score - a.score)
				return <Tabs.Content key={image.id} value={image.id}><SourceImageResult image={image} displayIndex={image.imageIndex ?? imageIndex} item={groupItems.find((item) => item.id === image.searchItemId)} candidates={imageCandidates} bestScore={imageCandidates[0]?.score ?? 0} requestStatus={request?.status ?? ''} /></Tabs.Content>
			  })}
			</Tabs.Root>
          </Card>
        })}
      </Flex>

      <Card size="3">
        <Flex justify="between" align="center" mb="3"><div><Heading size="3">设备搜索响应</Heading><Text size="1" color="gray">分别保留标准化模型与小红书原始响应</Text></div><Badge variant="soft">{responses.data?.items.length ?? 0} 条响应</Badge></Flex>
        {(responses.data?.items.length ?? 0) > 0 ? <Tabs.Root defaultValue={responses.data!.items[0].id}>
          <Tabs.List className="response-tabs">{responses.data!.items.map((response, index) => {
            const item = items.find((entry) => entry.jobId === response.jobId)
            const highSimilarity = bestCandidateScore(candidateItems, item?.id) >= 0.9
            return <Tabs.Trigger key={response.id} value={response.id}>响应 #{index + 1}<Text size="1" color="gray">{shortID(response.jobId)}</Text>{highSimilarity && <Badge size="1" color="green">命中</Badge>}</Tabs.Trigger>
          })}</Tabs.List>
          {responses.data!.items.map((response) => <Tabs.Content key={response.id} value={response.id}>
            <div className="response-meta"><Text size="1" color="gray">Job ID</Text><Text size="1">{response.jobId}</Text><Badge size="1" color={response.parseStatus === 'failed' ? 'red' : 'blue'}>{response.parseStatus}</Badge><Text size="1" color="gray">{response.itemCount} 条候选 · {fmtTime(response.createdAt)}</Text></div>
            <div className="response-json response-json-tab">{response.errorMessage && <Text as="div" color="red" size="2" mb="2">{response.errorMessage}</Text>}<Text as="div" size="1" weight="bold" mb="1">小红书原始 JSON</Text><pre>{response.rawJson ? prettyJSON(response.rawJson) : '当前设备版本未返回原始响应'}</pre><Text as="div" size="1" weight="bold" mt="3" mb="1">标准化模型</Text><pre>{prettyJSON(response.normalizedJson)}</pre></div>
          </Tabs.Content>)}
        </Tabs.Root> : <div className="empty-result"><Text color="gray">设备尚未返回搜索响应</Text></div>}
      </Card>
    </Flex>
  )
}

function SourceImageResult({ image, displayIndex, item, candidates, bestScore, requestStatus }: { image: SourceImage; displayIndex: number; item?: SearchItem; candidates: SearchCandidate[]; bestScore: number; requestStatus: string }) {
  const blocked = image.filterDecision === 'blocked'
  const [candidateLimit, setCandidateLimit] = useState(20)
  const visibleCandidates = candidates.slice(0, candidateLimit)
  return <section className="source-result-row">
    <div className="source-result-aside">
      <Text as="div" size="1" weight="bold" color="gray">源图片 #{displayIndex + 1}</Text>
      <div className="source-result-image">
        {image.fileUrl ? <AuthenticatedImage url={image.fileUrl} alt={`源图片 ${displayIndex + 1}`} className="source-image-preview" previewTitle={`源图片 #${displayIndex + 1}`} /> : <div className="source-image-unavailable">图片不可用</div>}
        {blocked && <span className="source-image-overlay source-image-overlay-filtered">OCR 已过滤</span>}
        {!blocked && bestScore >= 0.9 && <span className="source-image-overlay source-image-overlay-match">高相似 {formatScore(bestScore)}</span>}
      </div>
      <Flex justify="between" align="center" mt="2">
        <Badge size="1" color={blocked ? 'orange' : statusColor(image.searchItemId ? 'completed' : image.downloadStatus)}>{blocked ? '未搜索' : image.searchItemId ? '已搜索' : statusLabel(image.downloadStatus)}</Badge>
        <Text size="1" color="gray">{candidates.length} 条候选</Text>
      </Flex>
	  <div className="source-image-ids"><Text size="1" color="gray">SourceImage ID</Text><code>{image.id}</code><Text size="1" color="gray">SearchItem ID</Text><code>{image.searchItemId || '未创建（OCR 已过滤）'}</code>{item?.jobId && <><Text size="1" color="gray">Job ID</Text><code>{item.jobId}</code></>}</div>
      {image.filterReason && <Text as="div" size="1" color={blocked ? 'orange' : 'red'} mt="2">{image.filterReason}</Text>}
    </div>
    <div className="source-result-candidates">
      <Flex justify="between" align="center" mb="2"><Text size="1" weight="bold" color="gray">该图片的搜索结果</Text>{bestScore > 0 && <Text size="1" color="gray">最高 {formatScore(bestScore)}</Text>}</Flex>
      <Flex direction="column" gap="2">
        {visibleCandidates.map((candidate) => <CandidateResult key={candidate.id} candidate={candidate} />)}
        {candidateLimit < candidates.length && <Button variant="soft" color="gray" onClick={() => setCandidateLimit((limit) => limit + 20)}>加载更多（剩余 {candidates.length - candidateLimit} 条）</Button>}
        {candidates.length === 0 && <div className="empty-result"><Text color="gray">{blocked ? '图片命中 OCR 规则，未提交设备搜索' : terminalSearchStatuses.has(requestStatus) ? '该图片没有候选结果' : '等待该图片的搜索结果…'}</Text></div>}
      </Flex>
    </div>
  </section>
}

function CandidateResult({ candidate }: { candidate: SearchCandidate }) {
  return <div className={`candidate-row ${candidate.matched ? 'candidate-matched' : ''}`}>
    <CandidateGallery candidate={candidate} />
    <div className="candidate-copy"><Flex align="center" gap="2"><Text weight="medium" truncate>{candidate.title || candidate.contentId}</Text>{candidate.matched && <Badge color="green">MATCH</Badge>}</Flex><Flex align="center" gap="1"><XHSProfileDialog candidate={candidate} /><Text size="1" color="gray">· {candidate.authorId || '无用户 ID'}</Text></Flex><Text as="div" size="1" color="gray">图片进度 {candidate.analyzedImages}/{candidate.imageCount}{candidate.failedImages > 0 ? ` · 失败 ${candidate.failedImages}` : ''}</Text><Text as="div" size="1" color="gray">pHash {candidate.phashDistance ?? '—'} · dHash {candidate.dhashDistance ?? '—'} · aHash {candidate.ahashDistance ?? '—'}</Text>{candidate.downloadStatus === 'failed' && <Text as="div" size="1" color="red">{candidate.errorMessage || '候选图片下载失败'}</Text>}</div>
    <div className="candidate-score"><Text size="5" weight="bold">{formatScore(candidate.score)}</Text><Text as="div" size="1" color="gray">排名 #{candidate.rank + 1}</Text></div>
  </div>
}

function sourceImageFromItem(item: SearchItem): SourceImage {
  return { id: item.id, groupId: item.groupId, searchItemId: item.id, fileId: item.fileId, fileUrl: item.fileUrl, imageIndex: item.ordinal, downloadStatus: item.status }
}

function bestCandidateScore(candidates: SearchCandidate[], searchItemId?: string) {
  if (!searchItemId) return 0
  return candidates.reduce((best, candidate) => candidate.searchItemId === searchItemId ? Math.max(best, candidate.score) : best, 0)
}

function shortID(id: string) {
  return id.length > 8 ? id.slice(-8) : id
}

async function getAllCandidates(requestId: string): Promise<SearchCandidatePage> {
  const pageSize = 100
  const baseURL = `/api/v1/image-search/requests/${encodeURIComponent(requestId)}/candidates`
  const first = await get<SearchCandidatePage>(`${baseURL}?page=1&pageSize=${pageSize}`)
  const pageCount = Math.ceil(first.total / pageSize)
  if (pageCount <= 1) return first
  const remaining = await Promise.all(Array.from({ length: pageCount - 1 }, (_, index) =>
    get<SearchCandidatePage>(`${baseURL}?page=${index + 2}&pageSize=${pageSize}`),
  ))
  return { ...first, items: [first, ...remaining].flatMap((page) => page.items), pageSize: first.total }
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
