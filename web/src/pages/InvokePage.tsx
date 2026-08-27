import { useEffect, useMemo, useRef, useState } from 'react'
import {
  Flex,
  Box,
  Grid,
  Card,
  Heading,
  Text,
  Button,
  Badge,
  TextField,
  TextArea,
  Select,
  Callout,
} from '@radix-ui/themes'
import { RocketIcon } from '@radix-ui/react-icons'
import { get, postWithHeaders, upload, ApiError } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'
import { cnError } from '../lib/errors'
import { prettyJson } from '../lib/format'
import type { ActionDefinition, GroupInfo, Device, StoredFile } from '../types'

interface Result {
  ok: boolean
  data?: unknown
  error?: string
}

export default function InvokePage() {
  const imageInputRef = useRef<HTMLInputElement>(null)
  const groupsR = useFetch(() => get<{ items: GroupInfo[] }>('/api/groups'))
  const groups = groupsR.data?.items ?? []
  const devicesR = useFetch(() => get<{ items: Device[] }>('/api/devices'))
  const devices = devicesR.data?.items ?? []

  const [group, setGroup] = useState('')
  const [action, setAction] = useState('')
  const [clientId, setClientId] = useState('')
  const [timeout, setTimeoutS] = useState('15')
  const [payload, setPayload] = useState('{}')
  const [running, setRunning] = useState(false)
  const [result, setResult] = useState<Result | null>(null)
  const [actionOptions, setActionOptions] = useState<string[]>([])
  const [definitions, setDefinitions] = useState<Record<string, ActionDefinition>>({})
  const [customAction, setCustomAction] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [selectedFile, setSelectedFile] = useState<StoredFile | null>(null)
  const [traceId, setTraceId] = useState('')
  const [uploadTraceId, setUploadTraceId] = useState('')

  const effGroup = group || groups[0]?.group || ''

  // 选定分组后，拉取该分组扫描到的 action 列表
  useEffect(() => {
    if (!effGroup) {
      setActionOptions([])
      return
    }
    let alive = true
    setAction('')
    setCustomAction(false)
    setClientId('')
    get<{ actions: string[]; definitions: ActionDefinition[] }>(`/api/groups/${encodeURIComponent(effGroup)}/actions`)
      .then((d) => {
        if (alive) {
          setActionOptions(d.actions || [])
          setDefinitions(Object.fromEntries((d.definitions || []).map((item) => [item.name, item])))
        }
      })
      .catch(() => {
        if (alive) { setActionOptions([]); setDefinitions({}) }
      })
    return () => {
      alive = false
    }
  }, [effGroup])

  const definition = definitions[action]
  const fileInput = definition?.inputs.find((input) => input.type === 'file')

  function selectAction(value: string) {
    setAction(value)
    setSelectedFile(null)
    setUploadTraceId('')
    setPayload(JSON.stringify(definitions[value]?.payloadTemplate ?? {}, null, 2))
  }

  async function uploadImage(file?: File) {
    if (!file || !fileInput) return
    const body = new FormData()
    body.append('file', file)
    setUploading(true)
    try {
      const stored = await upload<StoredFile>('/api/files', body)
      let next: Record<string, unknown> = {}
      try { next = JSON.parse(payload || '{}') as Record<string, unknown> } catch { /* reset below */ }
      setPath(next, fileInput.path, stored.id)
      setPayload(JSON.stringify(next, null, 2))
      setSelectedFile(stored)
      notify.success('图片上传成功')
    } catch (error) {
      notify.error(error, '图片上传失败')
    } finally {
      setUploading(false)
    }
  }

  // 鉴权模式取自所选分组
  const curGroup = groups.find((g) => g.group === effGroup)
  const mode = curGroup?.authMode === 'apikey' ? 'apikey' : 'none'
  const apiKey = curGroup?.apiKey || ''
  const modeBadge =
    mode === 'apikey'
      ? { text: '需要 API Key', color: 'amber' as const }
      : { text: '免鉴权', color: 'green' as const }

  const curl = useMemo(() => {
    const base = window.location.origin
    const g = effGroup || '{group}'
    const a = action || '{action}'
    const keyHeader = mode === 'apikey' ? `\\\n  -H "X-API-Key: ${apiKey || '你的key'}"` : ''
    let parsed: unknown = {}
    try { parsed = JSON.parse(payload || '{}') } catch { /* 不合法时用空对象 */ }
    const body = JSON.stringify({ payload: parsed, timeoutSeconds: Number(timeout) || 15 })
    return `curl -X POST "${base}/rpc/${g}/${a}"${keyHeader} \\\n  -H "Content-Type: application/json" \\\n  -d '${body}'`
  }, [effGroup, action, payload, mode, apiKey, timeout, traceId])

  async function invoke() {
    if (!effGroup) {
      notify.error('请选择分组')
      return
    }
    if (!action.trim()) {
      notify.error('请输入 action')
      return
    }
    let parsed: Record<string, unknown> = {}
    if (payload.trim()) {
      try {
        parsed = JSON.parse(payload) as Record<string, unknown>
      } catch {
        notify.error('payload 不是合法 JSON')
        return
      }
    }
    setRunning(true)
    setResult(null)
    const currentTraceId = createTraceId()
    setTraceId(currentTraceId)
    setUploadTraceId('')
    try {
      if (action === 'content.search_by_image') {
        const existingHandle = typeof parsed.uploadHandle === 'string' ? parsed.uploadHandle.trim() : ''
        const image = isRecord(parsed.image) ? parsed.image : {}
        const fileId = typeof image.fileId === 'string' ? image.fileId.trim() : ''
        if (!existingHandle) {
          if (!fileId) throw new Error('请选择图片，或填写已有的 uploadHandle')
          const currentUploadTraceId = createTraceId()
          setUploadTraceId(currentUploadTraceId)
          const uploadData = await invokeAction<Record<string, unknown>>(
            'media.upload_image',
            { image: { fileId }, purpose: 'image_search' },
            currentUploadTraceId,
          )
          const uploadHandle = typeof uploadData.uploadHandle === 'string' ? uploadData.uploadHandle.trim() : ''
          if (!uploadHandle) throw new Error('media.upload_image 未返回有效 uploadHandle')
          parsed = { ...parsed, uploadHandle }
          setPayload(JSON.stringify(parsed, null, 2))
        }
      }
      const data = await invokeAction<unknown>(action.trim(), parsed, currentTraceId)
      setResult({ ok: true, data })
      notify.success('调用成功')
    } catch (e) {
      const msg = cnError(e instanceof ApiError ? e.detail : '请求异常')
      setResult({ ok: false, error: msg })
      notify.error(e, '请求异常')
    } finally {
      setRunning(false)
    }
  }

  async function invokeAction<T>(actionName: string, actionPayload: unknown, requestId: string) {
    const body: Record<string, unknown> = { payload: actionPayload, timeoutSeconds: Number(timeout) || 15 }
    if (clientId.trim()) body.clientId = clientId.trim()
    return postWithHeaders<T>(
      `/rpc/${encodeURIComponent(effGroup)}/${encodeURIComponent(actionName)}`,
      body,
      { 'X-Request-ID': requestId },
    )
  }

  return (
    <Flex direction="column" gap="4">
      <Flex justify="end" align="center" gap="2">
        <Text size="2" color="gray">
          对外鉴权模式
        </Text>
        <Badge color={modeBadge.color} variant="soft">
          {modeBadge.text}
        </Badge>
      </Flex>

      <Grid columns={{ initial: '1', md: '2' }} gap="4">
        {/* 请求 */}
        <Card size="3">
          <Heading size="3" mb="4">
            请求
          </Heading>
          <Flex direction="column" gap="3">
            <Grid columns="2" gap="3">
              <label>
                <Text size="2" mb="1" as="div" weight="medium">
                  分组
                </Text>
                <Select.Root value={effGroup} onValueChange={setGroup} disabled={groups.length === 0}>
                  <Select.Trigger placeholder="选择分组" style={{ width: '100%' }} />
                  <Select.Content>
                    {groups.map((g) => (
                      <Select.Item key={g.group} value={g.group}>
                        {g.displayName ? `${g.displayName} (${g.group})` : g.group}
                      </Select.Item>
                    ))}
                  </Select.Content>
                </Select.Root>
              </label>
              <label>
                <Text size="2" mb="1" as="div" weight="medium">
                  Action
                </Text>
                {actionOptions.length > 0 && !customAction ? (
                  <Select.Root
                    value={action}
                    onValueChange={(v) => {
                      if (v === '__custom__') {
                        setCustomAction(true)
                        setAction('')
                        setPayload('{}')
                        setSelectedFile(null)
                      } else {
                        selectAction(v)
                      }
                    }}
                  >
                    <Select.Trigger placeholder="选择 action" style={{ width: '100%' }} />
                    <Select.Content>
                      {actionOptions.map((a) => (
                        <Select.Item key={a} value={a}>
                          {a}
                        </Select.Item>
                      ))}
                      <Select.Separator />
                      <Select.Item value="__custom__">自定义…</Select.Item>
                    </Select.Content>
                  </Select.Root>
                ) : (
                  <Flex gap="1" align="center">
                    <TextField.Root
                      style={{ flex: 1 }}
                      value={action}
                      onChange={(e) => setAction(e.target.value)}
                      placeholder="例如 getToken"
                    />
                    {actionOptions.length > 0 && (
                      <Button variant="soft" color="gray" size="1" onClick={() => setCustomAction(false)}>
                        列表
                      </Button>
                    )}
                  </Flex>
                )}
              </label>
            </Grid>
            <Grid columns="2" gap="3">
              <label>
                <Text size="2" mb="1" as="div" weight="medium">
                  指定客户端（可选）
                </Text>
                <Select.Root value={clientId} onValueChange={(v) => setClientId(v === '__auto__' ? '' : v)}>
                  <Select.Trigger placeholder="自动调度" style={{ width: '100%' }} />
                  <Select.Content>
                    <Select.Item value="__auto__">自动调度</Select.Item>
                    {devices.filter((d) => d.group === effGroup && d.status === 'online').length > 0 && <Select.Separator />}
                    {devices.filter((d) => d.group === effGroup && d.status === 'online').map((d) => (
                      <Select.Item key={d.clientId} value={d.clientId}>
                        {d.clientId}
                      </Select.Item>
                    ))}
                  </Select.Content>
                </Select.Root>
              </label>
              <label>
                <Text size="2" mb="1" as="div" weight="medium">
                  超时（秒）
                </Text>
                <TextField.Root type="number" value={timeout} onChange={(e) => setTimeoutS(e.target.value)} />
              </label>
            </Grid>
            {fileInput && (
              <label>
                <Text size="2" mb="1" as="div" weight="medium">{fileInput.label}</Text>
                <Flex gap="3" align="center">
                  <input ref={imageInputRef} hidden type="file" accept={(fileInput.accept || []).join(',')} onChange={(e) => uploadImage(e.target.files?.[0])} />
                  <Button variant="soft" loading={uploading} onClick={() => imageInputRef.current?.click()}>选择图片</Button>
                  <Text size="2" color="gray">{selectedFile ? `${selectedFile.originalName} · ${formatBytes(selectedFile.sizeBytes)}` : '尚未选择'}</Text>
                </Flex>
              </label>
            )}
            <label>
              <Text size="2" mb="1" as="div" weight="medium">
                Payload (JSON)
              </Text>
              <TextArea
                value={payload}
                onChange={(e) => setPayload(e.target.value)}
                rows={6}
                style={{ fontFamily: 'var(--code-font-family)' }}
              />
            </label>
            <Button onClick={invoke} loading={running}>
              <RocketIcon /> 发起调用
            </Button>
            {uploadTraceId && <Text size="1" color="gray">图片上传 Trace ID: <code>{uploadTraceId}</code></Text>}
            {traceId && <Text size="1" color="gray">调用 Trace ID: <code>{traceId}</code></Text>}
            <Box>
              <Text size="1" color="gray" mb="1" as="div">
                curl 示例
              </Text>
              <Box
                p="3"
                style={{ background: 'var(--gray-2)', borderRadius: 'var(--radius-3)', border: '1px solid var(--gray-4)' }}
              >
                <pre style={{ margin: 0, fontSize: 12, fontFamily: 'var(--code-font-family)', whiteSpace: 'pre-wrap' }}>
                  {curl}
                </pre>
              </Box>
            </Box>
          </Flex>
        </Card>

        {/* 响应 */}
        <Card size="3">
          <Flex justify="between" align="center" mb="4">
            <Heading size="3">响应</Heading>
            {result && (
              <Badge color={result.ok ? 'green' : 'red'} variant="soft">
                {result.ok ? '成功' : '失败'}
              </Badge>
            )}
          </Flex>
          {!result ? (
            <Flex align="center" justify="center" py="8">
              <Text size="2" color="gray">
                发起调用后在此查看结果
              </Text>
            </Flex>
          ) : (
            <Flex direction="column" gap="3">
              {result.error && (
                <Callout.Root color="red" size="1">
                  <Callout.Text>{result.error}</Callout.Text>
                </Callout.Root>
              )}
              {result.ok && (
                <Box>
                  <Text size="1" color="gray" mb="1" as="div">
                    data
                  </Text>
                  <Box
                    p="3"
                    style={{
                      background: 'var(--gray-2)',
                      borderRadius: 'var(--radius-3)',
                      border: '1px solid var(--gray-4)',
                      maxHeight: 360,
                      overflow: 'auto',
                    }}
                  >
                    <pre
                      style={{ margin: 0, fontSize: 12, fontFamily: 'var(--code-font-family)', whiteSpace: 'pre-wrap' }}
                    >
                      {prettyJson(result.data)}
                    </pre>
                  </Box>
                </Box>
              )}
            </Flex>
          )}
        </Card>
      </Grid>
    </Flex>
  )
}

function setPath(target: Record<string, unknown>, path: string, value: unknown) {
  const parts = path.split('.')
  let current = target
  parts.forEach((part, index) => {
    if (index === parts.length - 1) { current[part] = value; return }
    const next = current[part]
    if (!next || typeof next !== 'object' || Array.isArray(next)) current[part] = {}
    current = current[part] as Record<string, unknown>
  })
}

function formatBytes(bytes: number) {
  return bytes < 1024 * 1024 ? `${(bytes / 1024).toFixed(1)} KiB` : `${(bytes / 1024 / 1024).toFixed(1)} MiB`
}

function createTraceId() {
  return crypto.randomUUID().replaceAll('-', '')
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}
