import { useState } from 'react'
import { Button, Card, Flex, Heading, Table, Text, TextField } from '@radix-ui/themes'
import { PlusIcon, ReloadIcon } from '@radix-ui/react-icons'
import { get, post } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'

interface Source { id:string; name:string; baseUrl:string; appId:string; accessKey:string; status:string; secretConfigured:boolean }
interface Channel { id:number; title?:string; username?:string; chatType?:string }

export default function DataSourcesPage() {
  const list = useFetch(() => get<{items:Source[]}>('/api/v1/data-sources'))
  const [form,setForm]=useState({name:'',baseUrl:'',appId:'',accessKey:'',secretKey:''})
  const [channels,setChannels]=useState<Channel[]>([])
  async function save(){try{await post('/api/v1/data-sources',form);notify.success('数据源已保存');setForm({name:'',baseUrl:'',appId:'',accessKey:'',secretKey:''});list.reload()}catch(e){notify.error(e,'保存数据源失败')}}
  async function loadChannels(id:string){try{const data=await get<{items:Channel[]}>(`/api/v1/data-sources/${id}/channels`);setChannels(data.items);notify.success(`已读取 ${data.items.length} 个频道`)}catch(e){notify.error(e,'读取频道失败')}}
  return <Flex direction="column" gap="4"><Flex justify="between" align="center"><div><Heading size="5">数据源管理</Heading><Text size="2" color="gray">配置 FeiNiu OpenAPI，频道同步任务将在此基础上创建。</Text></div><Button variant="soft" onClick={list.reload}><ReloadIcon/>刷新</Button></Flex><Card><Heading size="3" mb="3">新增数据源</Heading><Flex gap="2" wrap="wrap"><TextField.Root placeholder="名称" value={form.name} onChange={e=>setForm({...form,name:e.target.value})}/><TextField.Root placeholder="Base URL" value={form.baseUrl} onChange={e=>setForm({...form,baseUrl:e.target.value})}/><TextField.Root placeholder="APP_ID" value={form.appId} onChange={e=>setForm({...form,appId:e.target.value})}/><TextField.Root placeholder="AK" value={form.accessKey} onChange={e=>setForm({...form,accessKey:e.target.value})}/><TextField.Root type="password" placeholder="SK" value={form.secretKey} onChange={e=>setForm({...form,secretKey:e.target.value})}/><Button onClick={save}><PlusIcon/>保存</Button></Flex></Card><Card><Table.Root variant="surface"><Table.Header><Table.Row><Table.ColumnHeaderCell>名称</Table.ColumnHeaderCell><Table.ColumnHeaderCell>Base URL</Table.ColumnHeaderCell><Table.ColumnHeaderCell>APP_ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell><Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{(list.data?.items??[]).map(item=><Table.Row key={item.id}><Table.RowHeaderCell>{item.name}</Table.RowHeaderCell><Table.Cell>{item.baseUrl}</Table.Cell><Table.Cell>{item.appId}</Table.Cell><Table.Cell>{item.status}</Table.Cell><Table.Cell><Button size="1" variant="soft" onClick={()=>loadChannels(item.id)}>读取频道</Button></Table.Cell></Table.Row>)}</Table.Body></Table.Root></Card>{channels.length>0&&<Card><Heading size="3" mb="3">频道列表</Heading><Table.Root><Table.Header><Table.Row><Table.ColumnHeaderCell>ID</Table.ColumnHeaderCell><Table.ColumnHeaderCell>标题</Table.ColumnHeaderCell><Table.ColumnHeaderCell>用户名</Table.ColumnHeaderCell><Table.ColumnHeaderCell>类型</Table.ColumnHeaderCell></Table.Row></Table.Header><Table.Body>{channels.map(channel=><Table.Row key={channel.id}><Table.Cell>{channel.id}</Table.Cell><Table.Cell>{channel.title||'—'}</Table.Cell><Table.Cell>{channel.username||'—'}</Table.Cell><Table.Cell>{channel.chatType||'—'}</Table.Cell></Table.Row>)}</Table.Body></Table.Root></Card>}</Flex>
}
