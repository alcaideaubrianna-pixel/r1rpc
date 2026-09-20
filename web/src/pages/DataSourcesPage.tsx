import { useState } from 'react'
import { Button, Card, Dialog, Flex, Heading, Select, Table, Tabs, Text, TextField } from '@radix-ui/themes'
import { ChevronLeftIcon, ChevronRightIcon, PlusIcon, ReloadIcon } from '@radix-ui/react-icons'
import { get, patch, post } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'

interface Source { id: string; name: string; baseUrl: string; appId: string; accessKey: string; status: string; secretConfigured: boolean }
interface Channel { id: number; title?: string; username?: string; chatType?: string }
interface ScanTask { id: string; channelTitle: string; channelId: number; mode: string; status: string; initialLimit: number; pollIntervalMinutes: number; lastError?: string }
interface ScanPage { items: ScanTask[]; page: number; pageSize: number; total: number; totalPages: number }
interface SourceForm { id?: string; name: string; baseUrl: string; appId: string; accessKey: string; secretKey: string; status: string }
interface ScanForm { mode: 'once' | 'continuous'; initialLimit: number; pollIntervalMinutes: number; priority: number }
const emptySource: SourceForm = { name: '', baseUrl: '', appId: '', accessKey: '', secretKey: '', status: 'enabled' }

export default function DataSourcesPage() {
  const sources = useFetch(() => get<{ items: Source[] }>('/api/v1/data-sources'))
  const [page, setPage] = useState(1)
  const tasks = useFetch(() => get<ScanPage>(`/api/v1/channel-scan-tasks?page=${page}&pageSize=10`), [page])
  const [sourceForm, setSourceForm] = useState<SourceForm>(emptySource)
  const [sourceDialog, setSourceDialog] = useState(false)
  const [channels, setChannels] = useState<Channel[]>([])
  const [sourceId, setSourceId] = useState('')
  const [selected, setSelected] = useState<Channel | null>(null)
  const [scanForm, setScanForm] = useState<ScanForm>({ mode: 'once', initialLimit: 10, pollIntervalMinutes: 10, priority: 0 })

  async function saveSource() {
    try {
      const payload = { ...sourceForm }
      if (sourceForm.id) await patch(`/api/v1/data-sources/${sourceForm.id}`, payload)
      else await post('/api/v1/data-sources', payload)
      notify.success(sourceForm.id ? '数据源已更新' : '数据源已保存')
      setSourceDialog(false); setSourceForm(emptySource); sources.reload()
    } catch (error) { notify.error(error, '保存数据源失败') }
  }
  async function loadChannels(id: string) {
    try { const data = await get<{ items: Channel[] }>(`/api/v1/data-sources/${id}/channels`); setSourceId(id); setChannels(data.items); notify.success(`已读取 ${data.items.length} 个频道`) }
    catch (error) { notify.error(error, '读取频道失败') }
  }
  async function createTask() {
    if (!selected) return
    try { await post('/api/v1/channel-scan-tasks', { dataSourceId: sourceId, channelId: selected.id, channelTitle: selected.title ?? selected.username ?? String(selected.id), ...scanForm }); notify.success('频道扫描任务已创建'); setSelected(null); tasks.reload() }
    catch (error) { notify.error(error, '创建扫描任务失败') }
  }
  return <Flex direction="column" gap="4">
    <Flex justify="between" align="center"><div><Heading size="5">数据源与扫描任务</Heading><Text size="2" color="gray">扫描任务和数据源配置分开管理。</Text></div><Button variant="soft" onClick={() => { tasks.reload(); sources.reload() }}><ReloadIcon />刷新</Button></Flex>
    <Tabs.Root defaultValue="tasks"><Tabs.List><Tabs.Trigger value="tasks">扫描任务</Tabs.Trigger><Tabs.Trigger value="sources">数据源管理</Tabs.Trigger></Tabs.List>
      <Tabs.Content value="tasks"><Card><Flex justify="between" align="center" mb="3"><Heading size="3">扫描任务</Heading><Text size="2" color="gray">共 {tasks.data?.total ?? 0} 个任务</Text></Flex><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>频道</Table.ColumnHeaderCell><Table.ColumnHeaderCell>模式</Table.ColumnHeaderCell><Table.ColumnHeaderCell>条数</Table.ColumnHeaderCell><Table.ColumnHeaderCell>间隔</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell><Table.ColumnHeaderCell>错误</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(tasks.data?.items ?? []).map(task => <Table.Row key={task.id}><Table.Cell>{task.channelTitle || task.channelId}</Table.Cell><Table.Cell>{task.mode === 'continuous' ? '循环' : '单次'}</Table.Cell><Table.Cell>{task.initialLimit}</Table.Cell><Table.Cell>{task.mode === 'continuous' ? `${task.pollIntervalMinutes} 分钟` : '—'}</Table.Cell><Table.Cell>{task.status}</Table.Cell><Table.Cell>{task.lastError || '—'}</Table.Cell></Table.Row>)}</Table.Body></Table.Root>{(tasks.data?.totalPages ?? 0) > 1 && <Flex justify="end" align="center" gap="3" mt="3"><Button size="1" variant="soft" disabled={page <= 1} onClick={() => setPage(v => v - 1)}><ChevronLeftIcon />上一页</Button><Text size="2">第 {page} / {tasks.data?.totalPages} 页</Text><Button size="1" variant="soft" disabled={page >= (tasks.data?.totalPages ?? 0)} onClick={() => setPage(v => v + 1)}>下一页<ChevronRightIcon /></Button></Flex>}</Card></Tabs.Content>
      <Tabs.Content value="sources"><Card><Flex justify="between" align="center" mb="3"><div><Heading size="3">数据源管理</Heading><Text size="2" color="gray">配置 FeiNiu OpenAPI 数据源。</Text></div><Button onClick={() => { setSourceForm(emptySource); setSourceDialog(true) }}><PlusIcon />新增数据源</Button></Flex><Table.Root variant="surface"><Table.Header><Table.Row><Table.ColumnHeaderCell>名称</Table.ColumnHeaderCell><Table.ColumnHeaderCell>Base URL</Table.ColumnHeaderCell><Table.ColumnHeaderCell>APP_ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell><Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(sources.data?.items ?? []).map(item => <Table.Row key={item.id}><Table.RowHeaderCell>{item.name}</Table.RowHeaderCell><Table.Cell>{item.baseUrl}</Table.Cell><Table.Cell>{item.appId}</Table.Cell><Table.Cell>{item.status}</Table.Cell><Table.Cell><Flex gap="2"><Button size="1" variant="soft" onClick={() => { setSourceForm({ id: item.id, name: item.name, baseUrl: item.baseUrl, appId: item.appId, accessKey: item.accessKey, secretKey: '', status: item.status }); setSourceDialog(true) }}>编辑</Button><Button size="1" variant="soft" onClick={() => loadChannels(item.id)}>读取频道</Button></Flex></Table.Cell></Table.Row>)}</Table.Body></Table.Root></Card>{channels.length > 0 && <Card><Heading size="3" mb="3">频道列表</Heading><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>标题</Table.ColumnHeaderCell><Table.ColumnHeaderCell>用户名</Table.ColumnHeaderCell><Table.ColumnHeaderCell>扫描</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{channels.map(channel => <Table.Row key={channel.id}><Table.Cell>{channel.id}</Table.Cell><Table.Cell>{channel.title || '—'}</Table.Cell><Table.Cell>{channel.username || '—'}</Table.Cell><Table.Cell><Button size="1" onClick={() => setSelected(channel)}>创建扫描任务</Button></Table.Cell></Table.Row>)}</Table.Body></Table.Root></Card>}</Tabs.Content>
    </Tabs.Root>
    <Dialog.Root open={sourceDialog} onOpenChange={setSourceDialog}><Dialog.Content maxWidth="520px"><Dialog.Title>{sourceForm.id ? '编辑数据源' : '新增数据源'}</Dialog.Title><Flex direction="column" gap="3" mt="4"><TextField.Root placeholder="名称" value={sourceForm.name} onChange={e => setSourceForm({ ...sourceForm, name: e.target.value })} /><TextField.Root placeholder="Base URL" value={sourceForm.baseUrl} onChange={e => setSourceForm({ ...sourceForm, baseUrl: e.target.value })} /><TextField.Root placeholder="APP_ID" value={sourceForm.appId} onChange={e => setSourceForm({ ...sourceForm, appId: e.target.value })} /><TextField.Root placeholder="AK" value={sourceForm.accessKey} onChange={e => setSourceForm({ ...sourceForm, accessKey: e.target.value })} /><TextField.Root type="password" placeholder={sourceForm.id ? 'SK（留空表示不修改）' : 'SK'} value={sourceForm.secretKey} onChange={e => setSourceForm({ ...sourceForm, secretKey: e.target.value })} /></Flex><Flex justify="end" gap="2" mt="5"><Dialog.Close><Button variant="soft">取消</Button></Dialog.Close><Button onClick={saveSource}>保存</Button></Flex></Dialog.Content></Dialog.Root>
    <Dialog.Root open={selected !== null} onOpenChange={open => { if (!open) setSelected(null) }}><Dialog.Content maxWidth="460px"><Dialog.Title>创建频道扫描任务</Dialog.Title><Text size="2" color="gray">{selected?.title || selected?.username || selected?.id}</Text><Flex direction="column" gap="3" mt="4"><label>模式<Select.Root value={scanForm.mode} onValueChange={mode => setScanForm({ ...scanForm, mode: mode as ScanForm['mode'] })}><Select.Trigger /><Select.Content><Select.Item value="once">单次拉取</Select.Item><Select.Item value="continuous">循环增量同步</Select.Item></Select.Content></Select.Root></label><label>拉取条数<TextField.Root type="number" min="1" max="100" value={String(scanForm.initialLimit)} onChange={e => setScanForm({ ...scanForm, initialLimit: Number(e.target.value) || 1 })} /></label>{scanForm.mode === 'continuous' && <label>同步间隔（5–30 分钟）<TextField.Root type="number" min="5" max="30" value={String(scanForm.pollIntervalMinutes)} onChange={e => setScanForm({ ...scanForm, pollIntervalMinutes: Number(e.target.value) || 10 })} /></label>}<label>优先级<TextField.Root type="number" min="0" max="100" value={String(scanForm.priority)} onChange={e => setScanForm({ ...scanForm, priority: Number(e.target.value) || 0 })} /></label></Flex><Flex justify="end" gap="2" mt="5"><Dialog.Close><Button variant="soft">取消</Button></Dialog.Close><Button onClick={createTask}>创建</Button></Flex></Dialog.Content></Dialog.Root>
  </Flex>
}
