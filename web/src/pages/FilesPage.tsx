import { useRef, useState } from 'react'
import { Badge, Button, Card, Flex, Table, Text } from '@radix-ui/themes'
import { DownloadIcon, ReloadIcon, TrashIcon, UploadIcon } from '@radix-ui/react-icons'
import { del, get, upload } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'
import { fmtTime } from '../lib/format'
import type { StoredFile } from '../types'

const accept = 'image/jpeg,image/png,image/heic,image/heif'

export default function FilesPage() {
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const { data, loading, reload } = useFetch(
    () => get<{ items: StoredFile[] }>('/api/files'),
  )
  const files = data?.items ?? []

  async function uploadFile(file?: File) {
    if (!file) return
    const body = new FormData()
    body.append('file', file)
    setUploading(true)
    try {
      await upload<StoredFile>('/api/files', body)
      notify.success('图片上传成功')
      reload()
    } catch (error) {
      notify.error(error, '上传失败')
    } finally {
      setUploading(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  async function remove(id: string) {
    try {
      await del(`/api/files/${encodeURIComponent(id)}`)
      notify.success('文件已删除')
      reload()
    } catch (error) {
      notify.error(error, '删除失败')
    }
  }

  return (
    <Flex direction="column" gap="4">
      <Flex justify="between" align="center">
        <Text size="2" color="gray">本地图片最大 12 MiB，支持 JPEG、PNG、HEIC、HEIF</Text>
        <Flex gap="2">
          <input ref={inputRef} hidden type="file" accept={accept} onChange={(e) => uploadFile(e.target.files?.[0])} />
          <Button loading={uploading} onClick={() => inputRef.current?.click()}><UploadIcon /> 上传图片</Button>
          <Button variant="soft" color="gray" onClick={reload}><ReloadIcon /> 刷新</Button>
        </Flex>
      </Flex>
      <Card size="2">
        <Table.Root variant="surface">
          <Table.Header><Table.Row>
            <Table.ColumnHeaderCell>文件</Table.ColumnHeaderCell><Table.ColumnHeaderCell>类型</Table.ColumnHeaderCell>
            <Table.ColumnHeaderCell>大小</Table.ColumnHeaderCell><Table.ColumnHeaderCell>上传时间</Table.ColumnHeaderCell>
            <Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell>
          </Table.Row></Table.Header>
          <Table.Body>
            {files.map((file) => <Table.Row key={file.id} align="center">
              <Table.RowHeaderCell><Flex direction="column"><Text size="2">{file.originalName}</Text><Text size="1" color="gray">{file.id}</Text></Flex></Table.RowHeaderCell>
              <Table.Cell><Badge variant="soft">{file.contentType}</Badge></Table.Cell>
              <Table.Cell>{formatBytes(file.sizeBytes)}</Table.Cell>
              <Table.Cell>{fmtTime(file.createdAt)}</Table.Cell>
              <Table.Cell><Flex gap="2">
                <Button asChild size="1" variant="soft"><a href={`/api/files/${encodeURIComponent(file.id)}/content`} target="_blank" rel="noreferrer"><DownloadIcon /> 预览</a></Button>
                <Button size="1" variant="ghost" color="red" onClick={() => remove(file.id)}><TrashIcon /> 删除</Button>
              </Flex></Table.Cell>
            </Table.Row>)}
            {!loading && files.length === 0 && <Table.Row><Table.Cell colSpan={5}><Text color="gray">暂无文件</Text></Table.Cell></Table.Row>}
          </Table.Body>
        </Table.Root>
      </Card>
    </Flex>
  )
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MiB`
}
