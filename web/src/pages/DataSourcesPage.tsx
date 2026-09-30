import { useEffect, useState } from 'react'
import { AlertDialog, Badge, Button, Card, Dialog, Flex, Heading, Progress, Select, Table, Tabs, Text, TextField } from '@radix-ui/themes'
import { ChevronLeftIcon, ChevronRightIcon, MagnifyingGlassIcon, PlusIcon, ReloadIcon } from '@radix-ui/react-icons'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { del, get, patch, post } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'
import { statusColor, statusLabel } from '../features/image-search/status'

interface Source { id: string; name: string; baseUrl: string; imageBaseUrl: string; appId: string; accessKey: string; status: string }
interface Channel { id: string; dataSourceId: string; channelId: number; title: string; username: string; chatType: string; lastSyncedAt: string; isPinned: number; cachedNotes?: number; scannedNotes?: number }
interface Note { id: string; externalNoteId: string; title: string; plainText?: string; rawJson?: string; updatedAt: string; scanStatus: string; lastSearchRequestId?: string; bestScore?: number; images?: { id: string; imageIndex: number; sourceUrl: string; videoUrl?: string; assetType: string }[] }
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
  const [notesChannel, setNotesChannel] = useState<Channel | null>(null)
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
      <Tabs.Content value="channels"><Card><Flex justify="between" align="center" mb="3" wrap="wrap" gap="3"><div><Heading size="3">频道列表</Heading><Text size="2" color="gray">数据库已缓存 {channels.data?.total ?? 0} 个频道</Text></div><Flex gap="2"><Select.Root value={sourceFilter} onValueChange={value => { setSourceFilter(value); setChannelPage(1) }}><Select.Trigger /><Select.Content><Select.Item value="all">全部数据源</Select.Item>{(sources.data?.items ?? []).map(source => <Select.Item key={source.id} value={source.id}>{source.name}</Select.Item>)}</Select.Content></Select.Root><TextField.Root placeholder="搜索频道标题" value={keyword} onChange={event => setKeyword(event.target.value)} onKeyDown={event => { if (event.key === 'Enter') search() }}><TextField.Slot><MagnifyingGlassIcon /></TextField.Slot></TextField.Root><Button onClick={search}>搜索</Button></Flex></Flex><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>频道 ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>标题</Table.ColumnHeaderCell><Table.ColumnHeaderCell>资料缓存</Table.ColumnHeaderCell><Table.ColumnHeaderCell>用户名</Table.ColumnHeaderCell><Table.ColumnHeaderCell>类型</Table.ColumnHeaderCell><Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(channels.data?.items ?? []).map(channel => <Table.Row key={channel.id}><Table.Cell>{channel.channelId}</Table.Cell><Table.Cell>{channel.isPinned ? '📌 ' : ''}{channel.title || '—'}</Table.Cell><Table.Cell>{`缓存 ${channel.cachedNotes ?? 0} 条，已扫描 ${channel.scannedNotes ?? 0} / ${channel.cachedNotes ?? 0}`}</Table.Cell><Table.Cell>{channel.username || '—'}</Table.Cell><Table.Cell>{channel.chatType || '—'}</Table.Cell><Table.Cell><Flex gap="2" wrap="nowrap"><Button size="1" variant="soft" onClick={() => setNotesChannel(channel)}>查看资料</Button><Button size="1" variant={channel.isPinned ? 'solid' : 'soft'} color={channel.isPinned ? 'orange' : 'gray'} loading={pinningId === channel.id} onClick={() => togglePin(channel)}>{channel.isPinned ? '取消置顶' : '置顶'}</Button><Button size="1" onClick={() => setSelected(channel)}>创建扫描任务</Button></Flex></Table.Cell></Table.Row>)}</Table.Body></Table.Root><Pager page={channelPage} totalPages={channels.data?.totalPages ?? 0} setPage={setChannelPage} /></Card></Tabs.Content>
    </Tabs.Root>
    <SourceDialog open={sourceDialog} form={sourceForm} setOpen={setSourceDialog} setForm={setSourceForm} save={saveSource} />
    <ScanDialog channel={selected} configs={searchConfigs.data?.items ?? []} form={scanForm} setChannel={setSelected} setForm={setScanForm} create={createTask} />
    <NotesDialog channel={notesChannel} configs={searchConfigs.data?.items ?? []} close={() => setNotesChannel(null)} onCreated={() => { tasks.reload(); setTab('tasks') }} />
  </Flex>
}

export function SourceNotesPage() {
  const { sourceId = '', channelId = '' } = useParams()
  const navigate = useNavigate()
  const channel = { id: `${sourceId}-${channelId}`, dataSourceId: sourceId, channelId: Number(channelId), title: '频道资料', username: '', chatType: '', lastSyncedAt: '', isPinned: 0 }
  return <NotesDialog channel={channel} configs={[]} close={() => navigate('/data-sources')} onCreated={() => navigate('/data-sources')} />
}

function DeleteSourceButton({ source, loading, onDelete }: { source: Source; loading: boolean; onDelete: () => void }) { return <AlertDialog.Root><AlertDialog.Trigger><Button size="1" color="red" variant="soft" loading={loading}>删除</Button></AlertDialog.Trigger><AlertDialog.Content maxWidth="440px"><AlertDialog.Title>删除数据源</AlertDialog.Title><AlertDialog.Description size="2">确认删除「{source.name}」？关联扫描任务会被取消，频道缓存会被清理；已产生的资料和搜索结果会保留。</AlertDialog.Description><Flex justify="end" gap="2" mt="4"><AlertDialog.Cancel><Button variant="soft" color="gray">取消</Button></AlertDialog.Cancel><AlertDialog.Action><Button color="red" onClick={onDelete}>确认删除</Button></AlertDialog.Action></Flex></AlertDialog.Content></AlertDialog.Root> }

function TaskTable({ data, page, setPage }: { data: Page<ScanTask> | null; page: number; setPage: (value: number) => void }) { return <Card><Flex justify="between" mb="3"><Heading size="3">扫描任务</Heading><Text size="2" color="gray">共 {data?.total ?? 0} 个任务</Text></Flex><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>任务 ID / 频道</Table.ColumnHeaderCell><Table.ColumnHeaderCell>模式</Table.ColumnHeaderCell><Table.ColumnHeaderCell>请求 / 实际 / 跳过</Table.ColumnHeaderCell><Table.ColumnHeaderCell>间隔</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态与进度</Table.ColumnHeaderCell><Table.ColumnHeaderCell>搜索任务</Table.ColumnHeaderCell><Table.ColumnHeaderCell>说明</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(data?.items ?? []).map(task => { const total = task.fetchedCount || task.initialLimit; const processed = Math.min(total, (task.completedCount || 0) + (task.skippedCount || 0)); return <Table.Row key={task.id}><Table.Cell><Text as="div" size="1" color="gray">{task.id}</Text><Text as="div">{task.channelTitle || task.channelId}</Text></Table.Cell><Table.Cell>{task.mode === 'continuous' ? '循环' : '单次'}</Table.Cell><Table.Cell>{task.initialLimit} / {task.fetchedCount || '—'} / {task.fetchedCount ? task.skippedCount : '—'}</Table.Cell><Table.Cell>{task.mode === 'continuous' ? `${task.pollIntervalMinutes} 分钟` : '—'}</Table.Cell><Table.Cell><Flex direction="column" gap="1" style={{ minWidth: 150 }}><Flex justify="between"><Badge color={statusColor(task.status)}>{statusLabel(task.status)}</Badge><Text size="1" color="gray">{processed}/{total}</Text></Flex><Progress value={task.progressPercent || 0} /></Flex></Table.Cell><Table.Cell>{task.searchTaskId ? <Button asChild size="1" variant="soft"><Link to={`/image-search/${task.searchTaskId}`}>查看结果</Link></Button> : '—'}</Table.Cell><Table.Cell>{task.lastError || '—'}</Table.Cell></Table.Row>})}</Table.Body></Table.Root><Pager page={page} totalPages={data?.totalPages ?? 0} setPage={setPage} /></Card> }
function Pager({ page, totalPages, setPage }: { page: number; totalPages: number; setPage: (value: number) => void }) { return totalPages > 1 ? <Flex justify="end" align="center" gap="3" mt="3"><Button size="1" variant="soft" disabled={page <= 1} onClick={() => setPage(page - 1)}><ChevronLeftIcon />上一页</Button><Text size="2">第 {page} / {totalPages} 页</Text><Button size="1" variant="soft" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页<ChevronRightIcon /></Button></Flex> : null }
function SourceDialog({ open, form, setOpen, setForm, save }: { open: boolean; form: SourceForm; setOpen: (value: boolean) => void; setForm: (value: SourceForm) => void; save: () => void }) { return <Dialog.Root open={open} onOpenChange={setOpen}><Dialog.Content maxWidth="520px"><Dialog.Title>{form.id ? '编辑数据源' : '新增数据源'}</Dialog.Title><Flex direction="column" gap="3" mt="4"><TextField.Root placeholder="名称" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} /><TextField.Root placeholder="OpenAPI Base URL" value={form.baseUrl} onChange={e => setForm({ ...form, baseUrl: e.target.value })} /><TextField.Root placeholder="图片下载 Base URL（可选，例如内网 COS 地址）" value={form.imageBaseUrl} onChange={e => setForm({ ...form, imageBaseUrl: e.target.value })} /><Text size="1" color="gray">配置后使用该地址加 cos_path 下载图片；留空使用接口返回的 content_url。</Text><TextField.Root placeholder="APP_ID" value={form.appId} onChange={e => setForm({ ...form, appId: e.target.value })} /><TextField.Root placeholder="AK" value={form.accessKey} onChange={e => setForm({ ...form, accessKey: e.target.value })} /><TextField.Root type="password" placeholder={form.id ? 'SK（留空表示不修改）' : 'SK'} value={form.secretKey} onChange={e => setForm({ ...form, secretKey: e.target.value })} /></Flex><Flex justify="end" gap="2" mt="5"><Dialog.Close><Button variant="soft">取消</Button></Dialog.Close><Button onClick={save}>保存</Button></Flex></Dialog.Content></Dialog.Root> }
function ScanDialog({ channel, configs, form, setChannel, setForm, create }: { channel: Channel | null; configs: SearchConfig[]; form: ScanForm; setChannel: (value: Channel | null) => void; setForm: (value: ScanForm) => void; create: () => void }) { return <Dialog.Root open={channel !== null} onOpenChange={open => { if (!open) setChannel(null) }}><Dialog.Content maxWidth="460px"><Dialog.Title>创建频道扫描任务</Dialog.Title><Text size="2" color="gray">{channel?.title || channel?.channelId}</Text><Flex direction="column" gap="3" mt="4"><label>搜索配置<Select.Root value={form.searchConfigId || 'default'} onValueChange={value => setForm({ ...form, searchConfigId: value === 'default' ? '' : value })}><Select.Trigger /><Select.Content><Select.Item value="default">系统默认配置</Select.Item>{configs.filter(config => config.enabled).map(config => <Select.Item key={config.id} value={config.id}>{config.name}</Select.Item>)}</Select.Content></Select.Root></label><label>模式<Select.Root value={form.mode} onValueChange={mode => setForm({ ...form, mode: mode as ScanForm['mode'] })}><Select.Trigger /><Select.Content><Select.Item value="once">单次拉取</Select.Item><Select.Item value="continuous">循环增量同步</Select.Item></Select.Content></Select.Root></label><label>拉取条数<TextField.Root type="number" min="1" max="100" value={String(form.initialLimit)} onChange={e => setForm({ ...form, initialLimit: Number(e.target.value) || 1 })} /></label>{form.mode === 'continuous' && <label>同步间隔（5–30 分钟）<TextField.Root type="number" min="5" max="30" value={String(form.pollIntervalMinutes)} onChange={e => setForm({ ...form, pollIntervalMinutes: Number(e.target.value) || 10 })} /></label>}</Flex><Flex justify="end" gap="2" mt="5"><Dialog.Close><Button variant="soft">取消</Button></Dialog.Close><Button onClick={create}>创建</Button></Flex></Dialog.Content></Dialog.Root> }

function NotesDialog({ channel, configs, close, onCreated }: { channel: Channel | null; configs: SearchConfig[]; close: () => void; onCreated: () => void }) {
  const [page, setPage] = useState(1); const [pageSize, setPageSize] = useState(10); const [query, setQuery] = useState(''); const [selected, setSelected] = useState<string[]>([]); const [config, setConfig] = useState(''); const [busy, setBusy] = useState(false); const [syncTask, setSyncTask] = useState<ScanTask | null>(null); const [detail, setDetail] = useState<Note | null>(null)
  const notes = useFetch(() => channel ? get<Page<Note>>(`/api/v1/data-sources/${channel.dataSourceId}/channels/${channel.channelId}/notes?page=${page}&pageSize=${pageSize}${query ? `&q=${encodeURIComponent(query)}` : ''}`) : Promise.resolve(null), [channel?.id, page, pageSize, query])
  useEffect(() => { if (channel) { setPage(1); setQuery(''); setSelected([]) } }, [channel?.id])
  useEffect(() => { if (!syncTask || !channel || ['completed', 'failed', 'cancelled'].includes(syncTask.status)) return; const timer = window.setInterval(async () => { const result = await get<Page<ScanTask>>(`/api/v1/channel-scan-tasks?page=1&pageSize=100`); const current = result.items.find(item => item.id === syncTask.id); if (current) { setSyncTask(current); notes.reload() } }, 2000); return () => window.clearInterval(timer) }, [syncTask?.id, syncTask?.status, channel?.id])
  function toggle(id: string) { setSelected(items => items.includes(id) ? items.filter(item => item !== id) : items.length >= 10 ? items : [...items, id]) }
  const pageItems = notes.data?.items ?? []; const allSelected = pageItems.length > 0 && pageItems.every(item => selected.includes(item.id))
  function togglePage() { if (allSelected) setSelected(items => items.filter(id => !pageItems.some(item => item.id === id))); else setSelected(items => [...items, ...pageItems.map(item => item.id).filter(id => !items.includes(id))].slice(0, 10)) }
  function scanStatus(value: string) { return value === 'completed' || value === 'success' ? '扫描成功' : value === 'running' || value === 'preparing' ? '扫描中' : value === 'failed' ? '扫描失败' : '未扫描' }
  async function create(allNotes = false) { if (!channel || (!allNotes && selected.length === 0)) return; setBusy(true); try { await post(`/api/v1/data-sources/${channel.dataSourceId}/channels/${channel.channelId}/note-batches`, { noteIds: allNotes ? [] : selected, allNotes, searchConfigId: config, mode: 'once', initialLimit: allNotes ? 10000 : selected.length, priority: 0 }); notify.success(allNotes ? '已创建全部资料扫描任务' : `已创建 ${selected.length} 个资料扫描任务`); close(); onCreated() } catch (error) { notify.error(error, '创建资料扫描任务失败') } finally { setBusy(false) } }
  async function syncAll() { if (!channel) return; try { const task = await post<ScanTask>(`/api/v1/data-sources/${channel.dataSourceId}/channels/${channel.channelId}/notes/sync?title=${encodeURIComponent(channel.title)}`); setSyncTask(task); notify.success('已提交全部资料同步任务') } catch (error) { notify.error(error, '同步资料失败') } }
  const processed = (syncTask?.completedCount ?? 0) + (syncTask?.skippedCount ?? 0); const syncTotal = syncTask?.fetchedCount || syncTask?.initialLimit || 0
  return <><Dialog.Root open={channel !== null} onOpenChange={open => { if (!open) close() }}><Dialog.Content maxWidth="980px"><Dialog.Title>频道资料</Dialog.Title><Text size="2" color="gray">{channel?.title}，当前页最多选择 10 条</Text>{syncTask && <Card mt="3"><Flex justify="between"><Text>资料同步：{syncStatus(syncTask.status)}</Text><Text>{processed}/{syncTotal}</Text></Flex><Progress value={syncTotal ? processed / syncTotal * 100 : 0} /><Text size="1" color="gray">{syncTask.lastError || `拉取 ${syncTask.fetchedCount || 0} 条，新增 ${syncTask.completedCount || 0} 条，跳过 ${syncTask.skippedCount || 0} 条已扫描资料`}</Text></Card>}<Flex gap="2" mt="3" wrap="wrap"><TextField.Root placeholder="搜索资料标题" value={query} onChange={e => setQuery(e.target.value)} /><Button onClick={() => setPage(1)}>搜索</Button><Button variant="soft" onClick={syncAll}>同步全部资料</Button><Button variant="soft" color="orange" onClick={() => create(true)}>全部资料扫描</Button><Select.Root value={String(pageSize)} onValueChange={v => { setPageSize(Number(v)); setPage(1) }}><Select.Trigger /><Select.Content>{[10, 20, 50, 100].map(size => <Select.Item key={size} value={String(size)}>{size} 条/页</Select.Item>)}</Select.Content></Select.Root><Select.Root value={config || 'default'} onValueChange={v => setConfig(v === 'default' ? '' : v)}><Select.Trigger /><Select.Content><Select.Item value="default">默认配置</Select.Item>{configs.filter(item => item.enabled).map(item => <Select.Item key={item.id} value={item.id}>{item.name}</Select.Item>)}</Select.Content></Select.Root></Flex><Table.Root mt="3"><Table.Header><Table.Row><Table.ColumnHeaderCell><label><input type="checkbox" checked={allSelected} onChange={togglePage} /> 全选</label></Table.ColumnHeaderCell><Table.ColumnHeaderCell>资料编号 / ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>标题</Table.ColumnHeaderCell><Table.ColumnHeaderCell>更新时间</Table.ColumnHeaderCell><Table.ColumnHeaderCell>扫描状态</Table.ColumnHeaderCell><Table.ColumnHeaderCell>结果</Table.ColumnHeaderCell><Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{pageItems.map(note => <Table.Row key={note.id}><Table.Cell><input type="checkbox" checked={selected.includes(note.id)} onChange={() => toggle(note.id)} /></Table.Cell><Table.Cell><Text as="div">{note.externalNoteId}</Text><Text size="1" color="gray">{note.id}</Text></Table.Cell><Table.Cell>{note.title || '—'}</Table.Cell><Table.Cell>{note.updatedAt || '—'}</Table.Cell><Table.Cell><Badge color={note.scanStatus === 'completed' ? 'green' : note.scanStatus === 'running' ? 'blue' : note.scanStatus === 'failed' ? 'red' : 'gray'}>{scanStatus(note.scanStatus)}</Badge></Table.Cell><Table.Cell style={{ whiteSpace: 'nowrap' }}>{note.lastSearchRequestId ? <Button asChild size="1" variant="soft"><Link to={`/image-search/results/${note.lastSearchRequestId}`}>查看结果</Link></Button> : '—'}{note.bestScore ? ` ${(note.bestScore * 100).toFixed(1)}%` : ''}</Table.Cell><Table.Cell><Button size="1" variant="soft" onClick={() => setDetail(note)}>详情</Button></Table.Cell></Table.Row>)}</Table.Body></Table.Root><Flex justify="between" align="center" mt="3"><Text size="2">已选择 {selected.length}/10，共 {notes.data?.total ?? 0} 条</Text><Pager page={page} totalPages={notes.data?.totalPages ?? 0} setPage={setPage} /></Flex><Flex justify="end" gap="2" mt="4"><Button variant="soft" onClick={close}>关闭</Button><Button loading={busy} disabled={selected.length === 0} onClick={() => create(false)}>开始扫描</Button></Flex></Dialog.Content></Dialog.Root><NoteDetailDialog note={detail} close={() => setDetail(null)} /></>
}

function syncStatus(value: string) { return value === 'completed' ? '已完成' : value === 'failed' ? '失败' : value === 'running' || value === 'preparing' || value === 'queued' ? '同步中' : value }
function NoteDetailDialog({ note, close }: { note: Note | null; close: () => void }) { const [video, setVideo] = useState<{ url: string; poster: string } | null>(null); return <><Dialog.Root open={note !== null} onOpenChange={open => { if (!open) close() }}><Dialog.Content maxWidth="760px"><Dialog.Title>{note?.title || '资料详情'}</Dialog.Title><Text size="1" color="gray">资料编号：{note?.externalNoteId} · ID：{note?.id}</Text>{note?.images && note.images.length > 0 && <Flex gap="2" wrap="wrap" mt="4">{note.images.map(image => image.assetType === 'video_preview' ? <button key={image.id} type="button" onClick={() => image.videoUrl && setVideo({ url: image.videoUrl, poster: image.sourceUrl })} disabled={!image.videoUrl} style={{ position: 'relative', width: 150, height: 150, padding: 0, border: 0, background: '#111', borderRadius: 6, overflow: 'hidden', cursor: image.videoUrl ? 'pointer' : 'default' }}><img src={image.sourceUrl} alt={`视频预览 ${image.imageIndex + 1}`} style={{ width: '100%', height: '100%', objectFit: 'cover' }} /><span style={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center', color: '#fff', fontSize: 32, textShadow: '0 1px 3px #000' }}>▶</span></button> : <img key={image.id} src={image.sourceUrl} alt={`图片 ${image.imageIndex + 1}`} style={{ width: 150, height: 150, objectFit: 'cover', borderRadius: 6 }} />)}</Flex>}<Text as="div" size="2" mt="4" style={{ whiteSpace: 'pre-wrap' }}>{note?.plainText || '暂无原文'}</Text>{note?.rawJson && <details><summary>查看原始 JSON</summary><pre style={{ maxHeight: 260, overflow: 'auto', whiteSpace: 'pre-wrap' }}>{note.rawJson}</pre></details>}<Flex justify="end" mt="4"><Button variant="soft" onClick={close}>关闭</Button></Flex></Dialog.Content></Dialog.Root><Dialog.Root open={video !== null} onOpenChange={open => { if (!open) setVideo(null) }}><Dialog.Content maxWidth="900px"><Dialog.Title>视频播放</Dialog.Title>{video && <video src={video.url} poster={video.poster} controls autoPlay playsInline style={{ width: '100%', maxHeight: '70vh' }} />}<Flex justify="end" mt="3"><Button variant="soft" onClick={() => setVideo(null)}>关闭</Button></Flex></Dialog.Content></Dialog.Root></> }
