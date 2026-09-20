import { useState } from 'react'
import { Button, Card, Dialog, Flex, Heading, Select, Table, Text, TextField } from '@radix-ui/themes'
import { PlusIcon, ReloadIcon } from '@radix-ui/react-icons'
import { get, post } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'

interface Source { id: string; name: string; baseUrl: string; appId: string; accessKey: string; status: string; secretConfigured: boolean }
interface Channel { id: number; title?: string; username?: string; chatType?: string }
interface ScanTask { id: string; channelTitle: string; channelId: number; mode: string; status: string; initialLimit: number; pollIntervalMinutes: number; lastError?: string }
interface ScanForm { mode: 'once' | 'continuous'; initialLimit: number; pollIntervalMinutes: number; priority: number }
const emptyForm = { name: '', baseUrl: '', appId: '', accessKey: '', secretKey: '' }

export default function DataSourcesPage() {
  const sources = useFetch(() => get<{ items: Source[] }>('/api/v1/data-sources'))
  const tasks = useFetch(() => get<{ items: ScanTask[] }>('/api/v1/channel-scan-tasks'))
  const [form, setForm] = useState(emptyForm)
  const [channels, setChannels] = useState<Channel[]>([])
  const [sourceId, setSourceId] = useState('')
  const [selected, setSelected] = useState<Channel | null>(null)
  const [scanForm, setScanForm] = useState<ScanForm>({ mode: 'once', initialLimit: 10, pollIntervalMinutes: 10, priority: 0 })

  async function save() { try { await post('/api/v1/data-sources', form); notify.success('数据源已保存'); setForm(emptyForm); sources.reload() } catch (error) { notify.error(error, '保存数据源失败') } }
  async function loadChannels(id: string) { try { const data = await get<{ items: Channel[] }>(`/api/v1/data-sources/${id}/channels`); setSourceId(id); setChannels(data.items); notify.success(`已读取 ${data.items.length} 个频道`) } catch (error) { notify.error(error, '读取频道失败') } }
  async function createTask() {
    if (!selected) return
    try { await post('/api/v1/channel-scan-tasks', { dataSourceId: sourceId, channelId: selected.id, channelTitle: selected.title ?? selected.username ?? String(selected.id), ...scanForm }); notify.success('频道扫描任务已创建'); setSelected(null); tasks.reload() } catch (error) { notify.error(error, '创建扫描任务失败') }
  }
  return <Flex direction="column" gap="4">
    <Flex justify="between" align="center"><div><Heading size="5">数据源管理</Heading><Text size="2" color="gray">配置 FeiNiu OpenAPI，并创建频道增量同步任务。</Text></div><Button variant="soft" onClick={() => { sources.reload(); tasks.reload() }}><ReloadIcon />刷新</Button></Flex>
    <Card><Heading size="3" mb="3">新增数据源</Heading><Flex gap="2" wrap="wrap"><TextField.Root placeholder="名称" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} /><TextField.Root placeholder="Base URL" value={form.baseUrl} onChange={e => setForm({ ...form, baseUrl: e.target.value })} /><TextField.Root placeholder="APP_ID" value={form.appId} onChange={e => setForm({ ...form, appId: e.target.value })} /><TextField.Root placeholder="AK" value={form.accessKey} onChange={e => setForm({ ...form, accessKey: e.target.value })} /><TextField.Root type="password" placeholder="SK" value={form.secretKey} onChange={e => setForm({ ...form, secretKey: e.target.value })} /><Button onClick={save}><PlusIcon />保存</Button></Flex></Card>
    <Card><Table.Root variant="surface"><Table.Header><Table.Row><Table.ColumnHeaderCell>名称</Table.ColumnHeaderCell><Table.ColumnHeaderCell>Base URL</Table.ColumnHeaderCell><Table.ColumnHeaderCell>APP_ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell><Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(sources.data?.items ?? []).map(item => <Table.Row key={item.id}><Table.RowHeaderCell>{item.name}</Table.RowHeaderCell><Table.Cell>{item.baseUrl}</Table.Cell><Table.Cell>{item.appId}</Table.Cell><Table.Cell>{item.status}</Table.Cell><Table.Cell><Button size="1" variant="soft" onClick={() => loadChannels(item.id)}>读取频道</Button></Table.Cell></Table.Row>)}</Table.Body></Table.Root></Card>
    {channels.length > 0 && <Card><Heading size="3" mb="3">频道列表</Heading><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>标题</Table.ColumnHeaderCell><Table.ColumnHeaderCell>用户名</Table.ColumnHeaderCell><Table.ColumnHeaderCell>类型</Table.ColumnHeaderCell><Table.ColumnHeaderCell>扫描</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{channels.map(channel => <Table.Row key={channel.id}><Table.Cell>{channel.id}</Table.Cell><Table.Cell>{channel.title || '—'}</Table.Cell><Table.Cell>{channel.username || '—'}</Table.Cell><Table.Cell>{channel.chatType || '—'}</Table.Cell><Table.Cell><Button size="1" onClick={() => { setSelected(channel); setScanForm({ mode: 'once', initialLimit: 10, pollIntervalMinutes: 10, priority: 0 }) }}>创建扫描任务</Button></Table.Cell></Table.Row>)}</Table.Body></Table.Root></Card>}
    <Card><Heading size="3" mb="3">扫描任务</Heading><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>频道</Table.ColumnHeaderCell><Table.ColumnHeaderCell>模式</Table.ColumnHeaderCell><Table.ColumnHeaderCell>条数</Table.ColumnHeaderCell><Table.ColumnHeaderCell>间隔</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell><Table.ColumnHeaderCell>错误</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(tasks.data?.items ?? []).map(task => <Table.Row key={task.id}><Table.Cell>{task.channelTitle || task.channelId}</Table.Cell><Table.Cell>{task.mode === 'continuous' ? '循环' : '单次'}</Table.Cell><Table.Cell>{task.initialLimit}</Table.Cell><Table.Cell>{task.mode === 'continuous' ? `${task.pollIntervalMinutes} 分钟` : '—'}</Table.Cell><Table.Cell>{task.status}</Table.Cell><Table.Cell>{task.lastError || '—'}</Table.Cell></Table.Row>)}</Table.Body></Table.Root></Card>
    <Dialog.Root open={selected !== null} onOpenChange={open => { if (!open) setSelected(null) }}><Dialog.Content maxWidth="460px"><Dialog.Title>创建频道扫描任务</Dialog.Title><Dialog.Description size="2">{selected?.title || selected?.username || selected?.id}</Dialog.Description><Flex direction="column" gap="3" mt="4"><label>模式<Select.Root value={scanForm.mode} onValueChange={mode => setScanForm({ ...scanForm, mode: mode as ScanForm['mode'] })}><Select.Trigger /><Select.Content><Select.Item value="once">单次拉取</Select.Item><Select.Item value="continuous">循环增量同步</Select.Item></Select.Content></Select.Root></label><label>拉取条数<TextField.Root type="number" min="1" max="100" value={String(scanForm.initialLimit)} onChange={e => setScanForm({ ...scanForm, initialLimit: Number(e.target.value) || 1 })} /></label>{scanForm.mode === 'continuous' && <label>同步间隔（5–30 分钟）<TextField.Root type="number" min="5" max="30" value={String(scanForm.pollIntervalMinutes)} onChange={e => setScanForm({ ...scanForm, pollIntervalMinutes: Number(e.target.value) || 10 })} /></label>}<label>优先级<TextField.Root type="number" min="0" max="100" value={String(scanForm.priority)} onChange={e => setScanForm({ ...scanForm, priority: Number(e.target.value) || 0 })} /></label></Flex><Flex justify="end" gap="2" mt="5"><Dialog.Close><Button variant="soft">取消</Button></Dialog.Close><Button onClick={createTask}>创建</Button></Flex></Dialog.Content></Dialog.Root>
  </Flex>
}
