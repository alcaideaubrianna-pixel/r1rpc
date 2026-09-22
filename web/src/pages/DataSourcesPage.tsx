import { useEffect, useState } from 'react'
import { AlertDialog, Badge, Button, Card, Dialog, Flex, Heading, Progress, Select, Table, Tabs, Text, TextField } from '@radix-ui/themes'
import { ChevronLeftIcon, ChevronRightIcon, MagnifyingGlassIcon, PlusIcon, ReloadIcon } from '@radix-ui/react-icons'
import { Link } from 'react-router-dom'
import { del, get, patch, post } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'
import { statusColor, statusLabel } from '../features/image-search/status'

interface Source { id: string; name: string; baseUrl: string; imageBaseUrl: string; appId: string; accessKey: string; status: string }
interface Channel { id: string; dataSourceId: string; channelId: number; title: string; username: string; chatType: string; lastSyncedAt: string; isPinned: number }
interface ScanTask { id: string; channelTitle: string; channelId: number; mode: string; status: string; initialLimit: number; pollIntervalMinutes: number; fetchedCount: number; skippedCount: number; searchTaskId?: string; completedCount: number; progressPercent: number; lastError?: string }
interface SearchConfig { id: string; name: string; enabled: number }
interface Page<T> { items: T[]; page: number; pageSize: number; total: number; totalPages: number }
interface SourceForm { id?: string; name: string; baseUrl: string; imageBaseUrl: string; appId: string; accessKey: string; secretKey: string; status: string }
interface ScanForm { mode: 'once' | 'continuous'; initialLimit: number; pollIntervalMinutes: number; priority: number; searchConfigId: string }
const emptySource: SourceForm = { name: '', baseUrl: '', imageBaseUrl: '', appId: '', accessKey: '', secretKey: '', status: 'enabled' }

export default function DataSourcesPage() {
  const [tab, setTab] = useState('tasks')
  const sources = useFetch(() => get<{ items: Source[] }>('/api/v1/data-sources'))
  const searchConfigs = useFetch(() => get<{ items: SearchConfig[] }>('/api/v1/search-configs'))
  const [taskPage, setTaskPage] = useState(1)
  const tasks = useFetch(() => get<Page<ScanTask>>(`/api/v1/channel-scan-tasks?page=${taskPage}&pageSize=10`), [taskPage])
  const [channelPage, setChannelPage] = useState(1)
  const [sourceFilter, setSourceFilter] = useState('all')
  const [keyword, setKeyword] = useState('')
  const [query, setQuery] = useState('')
  const channels = useFetch(() => get<Page<Channel>>(`/api/v1/source-channels?page=${channelPage}&pageSize=20${sourceFilter === 'all' ? '' : `&dataSourceId=${encodeURIComponent(sourceFilter)}`}${query ? `&q=${encodeURIComponent(query)}` : ''}`), [channelPage, sourceFilter, query])
  const [sourceForm, setSourceForm] = useState<SourceForm>(emptySource)
  const [sourceDialog, setSourceDialog] = useState(false)
  const [syncingId, setSyncingId] = useState('')
  const [deletingId, setDeletingId] = useState('')
  const [selected, setSelected] = useState<Channel | null>(null)
  const [pinningId, setPinningId] = useState('')
  const [scanForm, setScanForm] = useState<ScanForm>({ mode: 'once', initialLimit: 10, pollIntervalMinutes: 10, priority: 0, searchConfigId: '' })

  useEffect(() => {
    const timer = window.setInterval(tasks.reload, 5000)
    return () => window.clearInterval(timer)
  }, [taskPage])

  async function saveSource() {
    try {
      if (sourceForm.id) await patch(`/api/v1/data-sources/${sourceForm.id}`, sourceForm)
      else await post('/api/v1/data-sources', sourceForm)
      notify.success(sourceForm.id ? '数据源已更新' : '数据源已保存')
      setSourceDialog(false); setSourceForm(emptySource); sources.reload()
    } catch (error) { notify.error(error, '保存数据源失败') }
  }
  async function syncChannels(source: Source) {
    setSyncingId(source.id)
    try {
      const result = await post<{ syncedCount: number }>(`/api/v1/data-sources/${source.id}/channels/sync`)
      notify.success(`已同步 ${result.syncedCount} 个频道到数据库`)
      setSourceFilter(source.id); setChannelPage(1); setTab('channels')
    } catch (error) { notify.error(error, '同步频道失败') }
    finally { setSyncingId('') }
  }
  async function deleteSource(source: Source) {
    setDeletingId(source.id)
    try {
      await del(`/api/v1/data-sources/${source.id}`)
      notify.success('数据源已删除，关联扫描任务已取消')
      if (sourceFilter === source.id) setSourceFilter('all')
      sources.reload(); tasks.reload(); channels.reload()
    } catch (error) { notify.error(error, '删除数据源失败') }
    finally { setDeletingId('') }
  }
  async function createTask() {
    if (!selected) return
    try {
      await post('/api/v1/channel-scan-tasks', { dataSourceId: selected.dataSourceId, channelId: selected.channelId, channelTitle: selected.title, ...scanForm })
      notify.success('频道扫描任务已创建'); setSelected(null); tasks.reload()
    } catch (error) { notify.error(error, '创建扫描任务失败') }
  }
  async function togglePin(channel: Channel) {
    setPinningId(channel.id)
    try {
      await patch(`/api/v1/source-channels/${channel.id}/pin`, { pinned: !channel.isPinned })
      notify.success(channel.isPinned ? '已取消置顶' : '频道已置顶')
      channels.reload()
    } catch (error) { notify.error(error, '更新频道置顶失败') }
    finally { setPinningId('') }
  }
  function search() { setChannelPage(1); setQuery(keyword.trim()) }

  return <Flex direction="column" gap="4">
    <Flex justify="between" align="center"><div><Heading size="5">数据源与扫描任务</Heading><Text size="2" color="gray">管理数据源、持久化频道目录和扫描任务。</Text></div><Button variant="soft" onClick={() => { sources.reload(); tasks.reload(); channels.reload() }}><ReloadIcon />刷新</Button></Flex>
    <Tabs.Root value={tab} onValueChange={setTab}><Tabs.List><Tabs.Trigger value="tasks">扫描任务</Tabs.Trigger><Tabs.Trigger value="sources">数据源管理</Tabs.Trigger><Tabs.Trigger value="channels">频道列表</Tabs.Trigger></Tabs.List>
      <Tabs.Content value="tasks"><TaskTable data={tasks.data} page={taskPage} setPage={setTaskPage} /></Tabs.Content>
      <Tabs.Content value="sources"><Card><Flex justify="between" align="center" mb="3"><Heading size="3">数据源管理</Heading><Button onClick={() => { setSourceForm(emptySource); setSourceDialog(true) }}><PlusIcon />新增数据源</Button></Flex><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>名称</Table.ColumnHeaderCell><Table.ColumnHeaderCell>Base URL</Table.ColumnHeaderCell><Table.ColumnHeaderCell>APP_ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell><Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(sources.data?.items ?? []).map(source => <Table.Row key={source.id}><Table.RowHeaderCell>{source.name}</Table.RowHeaderCell><Table.Cell>{source.baseUrl}</Table.Cell><Table.Cell>{source.appId}</Table.Cell><Table.Cell>{source.status}</Table.Cell><Table.Cell><Flex gap="2"><Button size="1" variant="soft" onClick={() => { setSourceForm({ ...source, secretKey: '' }); setSourceDialog(true) }}>编辑</Button><Button size="1" loading={syncingId === source.id} onClick={() => syncChannels(source)}>读取频道</Button><DeleteSourceButton source={source} loading={deletingId === source.id} onDelete={() => deleteSource(source)} /></Flex></Table.Cell></Table.Row>)}</Table.Body></Table.Root></Card></Tabs.Content>
      <Tabs.Content value="channels"><Card><Flex justify="between" align="center" mb="3" wrap="wrap" gap="3"><div><Heading size="3">频道列表</Heading><Text size="2" color="gray">数据库已缓存 {channels.data?.total ?? 0} 个频道</Text></div><Flex gap="2"><Select.Root value={sourceFilter} onValueChange={value => { setSourceFilter(value); setChannelPage(1) }}><Select.Trigger /><Select.Content><Select.Item value="all">全部数据源</Select.Item>{(sources.data?.items ?? []).map(source => <Select.Item key={source.id} value={source.id}>{source.name}</Select.Item>)}</Select.Content></Select.Root><TextField.Root placeholder="搜索频道标题" value={keyword} onChange={event => setKeyword(event.target.value)} onKeyDown={event => { if (event.key === 'Enter') search() }}><TextField.Slot><MagnifyingGlassIcon /></TextField.Slot></TextField.Root><Button onClick={search}>搜索</Button></Flex></Flex><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>频道 ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>标题</Table.ColumnHeaderCell><Table.ColumnHeaderCell>用户名</Table.ColumnHeaderCell><Table.ColumnHeaderCell>类型</Table.ColumnHeaderCell><Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(channels.data?.items ?? []).map(channel => <Table.Row key={channel.id}><Table.Cell>{channel.channelId}</Table.Cell><Table.Cell>{channel.isPinned ? '📌 ' : ''}{channel.title || '—'}</Table.Cell><Table.Cell>{channel.username || '—'}</Table.Cell><Table.Cell>{channel.chatType || '—'}</Table.Cell><Table.Cell><Flex gap="2"><Button size="1" variant={channel.isPinned ? 'solid' : 'soft'} color={channel.isPinned ? 'orange' : 'gray'} loading={pinningId === channel.id} onClick={() => togglePin(channel)}>{channel.isPinned ? '取消置顶' : '置顶'}</Button><Button size="1" onClick={() => setSelected(channel)}>创建扫描任务</Button></Flex></Table.Cell></Table.Row>)}</Table.Body></Table.Root><Pager page={channelPage} totalPages={channels.data?.totalPages ?? 0} setPage={setChannelPage} /></Card></Tabs.Content>
    </Tabs.Root>
    <SourceDialog open={sourceDialog} form={sourceForm} setOpen={setSourceDialog} setForm={setSourceForm} save={saveSource} />
    <ScanDialog channel={selected} configs={searchConfigs.data?.items ?? []} form={scanForm} setChannel={setSelected} setForm={setScanForm} create={createTask} />
  </Flex>
}

function DeleteSourceButton({ source, loading, onDelete }: { source: Source; loading: boolean; onDelete: () => void }) { return <AlertDialog.Root><AlertDialog.Trigger><Button size="1" color="red" variant="soft" loading={loading}>删除</Button></AlertDialog.Trigger><AlertDialog.Content maxWidth="440px"><AlertDialog.Title>删除数据源</AlertDialog.Title><AlertDialog.Description size="2">确认删除「{source.name}」？关联扫描任务会被取消，频道缓存会被清理；已产生的资料和搜索结果会保留。</AlertDialog.Description><Flex justify="end" gap="2" mt="4"><AlertDialog.Cancel><Button variant="soft" color="gray">取消</Button></AlertDialog.Cancel><AlertDialog.Action><Button color="red" onClick={onDelete}>确认删除</Button></AlertDialog.Action></Flex></AlertDialog.Content></AlertDialog.Root> }

function TaskTable({ data, page, setPage }: { data: Page<ScanTask> | null; page: number; setPage: (value: number) => void }) { return <Card><Flex justify="between" mb="3"><Heading size="3">扫描任务</Heading><Text size="2" color="gray">共 {data?.total ?? 0} 个任务</Text></Flex><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>任务 ID / 频道</Table.ColumnHeaderCell><Table.ColumnHeaderCell>模式</Table.ColumnHeaderCell><Table.ColumnHeaderCell>请求 / 实际 / 跳过</Table.ColumnHeaderCell><Table.ColumnHeaderCell>间隔</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态与进度</Table.ColumnHeaderCell><Table.ColumnHeaderCell>搜索任务</Table.ColumnHeaderCell><Table.ColumnHeaderCell>说明</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(data?.items ?? []).map(task => { const total = task.fetchedCount || task.initialLimit; const processed = Math.min(total, (task.completedCount || 0) + (task.skippedCount || 0)); return <Table.Row key={task.id}><Table.Cell><Text as="div" size="1" color="gray">{task.id}</Text><Text as="div">{task.channelTitle || task.channelId}</Text></Table.Cell><Table.Cell>{task.mode === 'continuous' ? '循环' : '单次'}</Table.Cell><Table.Cell>{task.initialLimit} / {task.fetchedCount || '—'} / {task.fetchedCount ? task.skippedCount : '—'}</Table.Cell><Table.Cell>{task.mode === 'continuous' ? `${task.pollIntervalMinutes} 分钟` : '—'}</Table.Cell><Table.Cell><Flex direction="column" gap="1" style={{ minWidth: 150 }}><Flex justify="between"><Badge color={statusColor(task.status)}>{statusLabel(task.status)}</Badge><Text size="1" color="gray">{processed}/{total}</Text></Flex><Progress value={task.progressPercent || 0} /></Flex></Table.Cell><Table.Cell>{task.searchTaskId ? <Button asChild size="1" variant="soft"><Link to={`/image-search/${task.searchTaskId}`}>查看结果</Link></Button> : '—'}</Table.Cell><Table.Cell>{task.lastError || '—'}</Table.Cell></Table.Row>})}</Table.Body></Table.Root><Pager page={page} totalPages={data?.totalPages ?? 0} setPage={setPage} /></Card> }
function Pager({ page, totalPages, setPage }: { page: number; totalPages: number; setPage: (value: number) => void }) { return totalPages > 1 ? <Flex justify="end" align="center" gap="3" mt="3"><Button size="1" variant="soft" disabled={page <= 1} onClick={() => setPage(page - 1)}><ChevronLeftIcon />上一页</Button><Text size="2">第 {page} / {totalPages} 页</Text><Button size="1" variant="soft" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页<ChevronRightIcon /></Button></Flex> : null }
function SourceDialog({ open, form, setOpen, setForm, save }: { open: boolean; form: SourceForm; setOpen: (value: boolean) => void; setForm: (value: SourceForm) => void; save: () => void }) { return <Dialog.Root open={open} onOpenChange={setOpen}><Dialog.Content maxWidth="520px"><Dialog.Title>{form.id ? '编辑数据源' : '新增数据源'}</Dialog.Title><Flex direction="column" gap="3" mt="4"><TextField.Root placeholder="名称" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} /><TextField.Root placeholder="OpenAPI Base URL" value={form.baseUrl} onChange={e => setForm({ ...form, baseUrl: e.target.value })} /><TextField.Root placeholder="图片下载 Base URL（可选，例如内网 COS 地址）" value={form.imageBaseUrl} onChange={e => setForm({ ...form, imageBaseUrl: e.target.value })} /><Text size="1" color="gray">配置后使用该地址加 cos_path 下载图片；留空使用接口返回的 content_url。</Text><TextField.Root placeholder="APP_ID" value={form.appId} onChange={e => setForm({ ...form, appId: e.target.value })} /><TextField.Root placeholder="AK" value={form.accessKey} onChange={e => setForm({ ...form, accessKey: e.target.value })} /><TextField.Root type="password" placeholder={form.id ? 'SK（留空表示不修改）' : 'SK'} value={form.secretKey} onChange={e => setForm({ ...form, secretKey: e.target.value })} /></Flex><Flex justify="end" gap="2" mt="5"><Dialog.Close><Button variant="soft">取消</Button></Dialog.Close><Button onClick={save}>保存</Button></Flex></Dialog.Content></Dialog.Root> }
function ScanDialog({ channel, configs, form, setChannel, setForm, create }: { channel: Channel | null; configs: SearchConfig[]; form: ScanForm; setChannel: (value: Channel | null) => void; setForm: (value: ScanForm) => void; create: () => void }) { return <Dialog.Root open={channel !== null} onOpenChange={open => { if (!open) setChannel(null) }}><Dialog.Content maxWidth="460px"><Dialog.Title>创建频道扫描任务</Dialog.Title><Text size="2" color="gray">{channel?.title || channel?.channelId}</Text><Flex direction="column" gap="3" mt="4"><label>搜索配置<Select.Root value={form.searchConfigId || 'default'} onValueChange={value => setForm({ ...form, searchConfigId: value === 'default' ? '' : value })}><Select.Trigger /><Select.Content><Select.Item value="default">系统默认配置</Select.Item>{configs.filter(config => config.enabled).map(config => <Select.Item key={config.id} value={config.id}>{config.name}</Select.Item>)}</Select.Content></Select.Root></label><label>模式<Select.Root value={form.mode} onValueChange={mode => setForm({ ...form, mode: mode as ScanForm['mode'] })}><Select.Trigger /><Select.Content><Select.Item value="once">单次拉取</Select.Item><Select.Item value="continuous">循环增量同步</Select.Item></Select.Content></Select.Root></label><label>拉取条数<TextField.Root type="number" min="1" max="100" value={String(form.initialLimit)} onChange={e => setForm({ ...form, initialLimit: Number(e.target.value) || 1 })} /></label>{form.mode === 'continuous' && <label>同步间隔（5–30 分钟）<TextField.Root type="number" min="5" max="30" value={String(form.pollIntervalMinutes)} onChange={e => setForm({ ...form, pollIntervalMinutes: Number(e.target.value) || 10 })} /></label>}</Flex><Flex justify="end" gap="2" mt="5"><Dialog.Close><Button variant="soft">取消</Button></Dialog.Close><Button onClick={create}>创建</Button></Flex></Dialog.Content></Dialog.Root> }
