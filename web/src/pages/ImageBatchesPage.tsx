import { useState } from 'react'
import { Badge, Button, Card, Flex, Heading, Table, Text } from '@radix-ui/themes'
import { get, post, upload } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'
import { randomUUID } from '../lib/id'
import type { StoredFile } from '../types'
import type { ImageBatch, ImageBatchPage } from '../features/image-task/types'

export default function ImageBatchesPage() {
  const batches = useFetch(() => get<ImageBatchPage>('/api/v1/image-recognition/batches?page=1&pageSize=50'))
  const [files, setFiles] = useState<File[]>([])
  const [creating, setCreating] = useState(false)
  const [progress, setProgress] = useState('')

  async function createBatch() {
    if (files.length < 1 || files.length > 100) {
      notify.error('请选择 1～100 张图片')
      return
    }
    setCreating(true)
    try {
      const fileIds: string[] = []
      for (let index = 0; index < files.length; index += 1) {
        setProgress(`正在上传 ${index + 1}/${files.length}`)
        const body = new FormData()
        body.append('file', files[index])
        const stored = await upload<StoredFile>('/api/files', body)
        fileIds.push(stored.id)
      }
      setProgress('正在创建批次')
      await post('/api/v1/image-recognition/batches', {
        fileIds,
        externalId: randomUUID(),
      })
      setFiles([])
      notify.success('批量识图任务已创建')
      await batches.reload()
    } catch (error) {
      notify.error(error, '创建批次失败')
    } finally {
      setCreating(false)
      setProgress('')
    }
  }

  return (
    <Flex direction="column" gap="4">
      <Card size="3">
        <Heading size="3" mb="3">创建批量识图任务</Heading>
        <Flex direction="column" gap="3">
          <input
            type="file"
            multiple
            accept="image/jpeg,image/png,image/heic,image/heif"
            disabled={creating}
            onChange={(event) => setFiles(Array.from(event.target.files ?? []).slice(0, 100))}
          />
          <Text size="2" color="gray">已选择 {files.length} 张，单次最多 100 张</Text>
          <Flex align="center" gap="3">
            <Button loading={creating} onClick={createBatch}>上传并创建任务</Button>
            {progress && <Text size="2">{progress}</Text>}
          </Flex>
        </Flex>
      </Card>

      <Card size="3">
        <Flex justify="between" align="center" mb="3">
          <Heading size="3">批量任务</Heading>
          <Button variant="soft" onClick={() => batches.reload()}>刷新</Button>
        </Flex>
        <Table.Root variant="surface">
          <Table.Header><Table.Row>
            <Table.ColumnHeaderCell>批次 ID</Table.ColumnHeaderCell>
            <Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell>
            <Table.ColumnHeaderCell>进度</Table.ColumnHeaderCell>
            <Table.ColumnHeaderCell>缓存命中</Table.ColumnHeaderCell>
            <Table.ColumnHeaderCell>创建时间</Table.ColumnHeaderCell>
          </Table.Row></Table.Header>
          <Table.Body>{(batches.data?.items ?? []).map((item: ImageBatch) => (
            <Table.Row key={item.id}>
              <Table.Cell><code>{item.id}</code></Table.Cell>
              <Table.Cell><Badge>{item.status}</Badge></Table.Cell>
              <Table.Cell>{item.completed + item.failed}/{item.total}（排队 {item.queued}，运行 {item.running}）</Table.Cell>
              <Table.Cell>{item.cacheHit}</Table.Cell>
              <Table.Cell>{new Date(item.createdAt).toLocaleString()}</Table.Cell>
            </Table.Row>
          ))}</Table.Body>
        </Table.Root>
      </Card>
    </Flex>
  )
}
