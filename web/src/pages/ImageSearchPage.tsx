import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Badge, Button, Card, Flex, Heading, Table, Text, TextField } from '@radix-ui/themes'
import { ChevronLeftIcon, ChevronRightIcon, PlusIcon, ReloadIcon, RocketIcon, TrashIcon } from '@radix-ui/react-icons'
import { get, post, upload } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'
import { fmtTime } from '../lib/format'
import { randomUUID } from '../lib/id'
import type { StoredFile } from '../types'
import type { SearchRequestPage, SearchRequestSummary } from '../features/image-search/types'
import { statusColor, statusLabel } from '../features/image-search/status'

interface DraftGroup {
  id: string
  subjectUserId: string
  files: File[]
}

const newGroup = (): DraftGroup => ({ id: randomUUID(), subjectUserId: '', files: [] })

export default function ImageSearchPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const requests = useFetch(() => get<SearchRequestPage>(`/api/v1/image-search/requests?page=${page}&pageSize=10`), [page])
  const [groups, setGroups] = useState<DraftGroup[]>([newGroup()])
  const [threshold, setThreshold] = useState('0.82')
  const [maxDistance, setMaxDistance] = useState('12')
  const [creating, setCreating] = useState(false)
  const [progress, setProgress] = useState('')

  useEffect(() => {
    const timer = window.setInterval(requests.reload, 5000)
    return () => window.clearInterval(timer)
  }, [])

  function updateGroup(id: string, patch: Partial<DraftGroup>) {
    setGroups((current) => current.map((group) => group.id === id ? { ...group, ...patch } : group))
  }

  async function createRequest() {
    if (groups.length === 0 || groups.some((group) => group.files.length === 0)) {
      notify.error('每个任务组至少选择一张图片')
      return
    }
    const scoreThreshold = Number(threshold)
    const maxPHashDistance = Number(maxDistance)
    if (!(scoreThreshold > 0 && scoreThreshold <= 1) || !(maxPHashDistance >= 1 && maxPHashDistance <= 64)) {
      notify.error('请检查相似度阈值和 pHash 距离')
      return
    }
    setCreating(true)
    try {
      const payloadGroups = []
      let uploaded = 0
      const total = groups.reduce((sum, group) => sum + group.files.length, 0)
      for (const group of groups) {
        const images = []
        for (const file of group.files) {
          setProgress(`上传图片 ${uploaded + 1}/${total}`)
          const body = new FormData()
          body.append('file', file)
          const stored = await upload<StoredFile>('/api/files', body)
          images.push({ fileId: stored.id })
          uploaded += 1
        }
        payloadGroups.push({
          externalId: group.id,
          subjectUserId: group.subjectUserId.trim(),
          images,
          matchPolicy: { scoreThreshold, maxPHashDistance },
        })
      }
      setProgress('创建并调度任务')
      const created = await post<{ request: SearchRequestSummary }>('/api/v1/image-search/requests', {
        externalId: randomUUID(),
        groups: payloadGroups,
      })
      notify.success('图片搜索任务组已创建')
      navigate(`/image-search/${created.request.id}`)
    } catch (error) {
      notify.error(error, '创建图片搜索任务失败')
    } finally {
      setCreating(false)
      setProgress('')
    }
  }

  return (
    <Flex direction="column" gap="4" className="image-search-page">
      <Card size="3" className="search-create-card">
        <Flex justify="between" align="start" gap="5" wrap="wrap">
          <div>
            <Text size="1" weight="bold" color="blue">VISUAL SEARCH LAB</Text>
            <Heading size="5" mt="1">创建图片搜索任务组</Heading>
            <Text size="2" color="gray">每组任意一张图片匹配成功，该组即完成。</Text>
          </div>
          <Flex gap="3" align="end" wrap="wrap">
            <label><Text as="div" size="1" color="gray" mb="1">最低综合分数</Text><TextField.Root value={threshold} onChange={(e) => setThreshold(e.target.value)} style={{ width: 130 }} /></label>
            <label><Text as="div" size="1" color="gray" mb="1">最大 pHash 距离</Text><TextField.Root value={maxDistance} onChange={(e) => setMaxDistance(e.target.value)} style={{ width: 130 }} /></label>
          </Flex>
        </Flex>

        <Flex direction="column" gap="3" mt="5">
          {groups.map((group, index) => <Card key={group.id} variant="surface" className="search-group-draft">
            <Flex justify="between" align="center" gap="3" wrap="wrap">
              <Flex align="center" gap="3"><span className="group-index">{String(index + 1).padStart(2, '0')}</span><div><Text weight="bold">图片组</Text><Text as="div" size="1" color="gray">{group.files.length} 张图片</Text></div></Flex>
              <Flex gap="2" align="center" wrap="wrap">
                <TextField.Root placeholder="关联用户 ID（可选）" value={group.subjectUserId} onChange={(e) => updateGroup(group.id, { subjectUserId: e.target.value })} style={{ width: 220 }} />
                <input type="file" multiple accept="image/jpeg,image/png,image/webp" disabled={creating} onChange={(e) => updateGroup(group.id, { files: Array.from(e.target.files ?? []).slice(0, 100) })} />
                {groups.length > 1 && <Button size="1" variant="ghost" color="red" onClick={() => setGroups((current) => current.filter((item) => item.id !== group.id))}><TrashIcon /></Button>}
              </Flex>
            </Flex>
            {group.files.length > 0 && <Flex gap="2" mt="3" wrap="wrap">{group.files.slice(0, 8).map((file) => <Badge key={`${file.name}-${file.size}`} variant="soft" color="gray">{file.name}</Badge>)}{group.files.length > 8 && <Badge>+{group.files.length - 8}</Badge>}</Flex>}
          </Card>)}
          <Flex justify="between" align="center" wrap="wrap" gap="3">
            <Button variant="soft" onClick={() => setGroups((current) => [...current, newGroup()])}><PlusIcon /> 增加图片组</Button>
            <Flex align="center" gap="3">{progress && <Text size="2" color="gray">{progress}</Text>}<Button size="3" loading={creating} onClick={createRequest}><RocketIcon /> 上传并开始搜索</Button></Flex>
          </Flex>
        </Flex>
      </Card>

      <Card size="3">
        <Flex justify="between" align="center" mb="3"><div><Heading size="3">搜索任务</Heading><Text size="1" color="gray">自动每 5 秒刷新</Text></div><Button variant="soft" color="gray" onClick={requests.reload}><ReloadIcon /> 刷新</Button></Flex>
        <Table.Root variant="surface"><Table.Header><Table.Row><Table.ColumnHeaderCell>任务</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell><Table.ColumnHeaderCell>图片组</Table.ColumnHeaderCell><Table.ColumnHeaderCell>结果</Table.ColumnHeaderCell><Table.ColumnHeaderCell>创建时间</Table.ColumnHeaderCell><Table.ColumnHeaderCell /></Table.Row></Table.Header>
          <Table.Body>{(requests.data?.items ?? []).map((item) => <Table.Row key={item.id} align="center"><Table.RowHeaderCell><Text size="2" weight="medium">{item.externalId || item.id}</Text><Text as="div" size="1" color="gray">{item.id}</Text></Table.RowHeaderCell><Table.Cell><Badge color={statusColor(item.status)}>{statusLabel(item.status)}</Badge></Table.Cell><Table.Cell>{item.groupCount}</Table.Cell><Table.Cell><Flex gap="2"><Badge color="green" variant="soft">命中 {item.matchedCount}</Badge><Badge color="gray" variant="soft">未命中 {item.notMatchedCount}</Badge>{item.failedCount > 0 && <Badge color="red">失败 {item.failedCount}</Badge>}</Flex></Table.Cell><Table.Cell>{fmtTime(item.createdAt)}</Table.Cell><Table.Cell><Button asChild size="1" variant="soft"><Link to={`/image-search/${item.id}`}>查看结果</Link></Button></Table.Cell></Table.Row>)}</Table.Body>
        </Table.Root>
        {(requests.data?.total ?? 0) > 10 && <Flex align="center" justify="end" gap="3" mt="3"><Button variant="soft" color="gray" size="1" disabled={page <= 1} onClick={() => setPage((v) => v - 1)}><ChevronLeftIcon /> 上一页</Button><Text size="2" color="gray">第 {page} / {Math.ceil((requests.data?.total ?? 0) / 10)} 页</Text><Button variant="soft" color="gray" size="1" disabled={page >= Math.ceil((requests.data?.total ?? 0) / 10)} onClick={() => setPage((v) => v + 1)}>下一页 <ChevronRightIcon /></Button></Flex>}
      </Card>
    </Flex>
  )
}
