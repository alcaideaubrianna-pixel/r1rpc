import { useState } from 'react'
import { Avatar, Button, Dialog, Flex, Grid, Text } from '@radix-ui/themes'
import { get } from '../../api/client'
import { notify } from '../../lib/toast'
import type { SearchCandidate } from './types'
import { AuthenticatedImage } from './AuthenticatedImage'

interface Profile {
  userId: string; redId: string; nickname: string; avatarUrl?: string; description?: string
  gender: number; ipLocation?: string; fansCount: number; likedCount: number
  collectedCount: number; noteCount: number; cached: boolean
}

export function XHSProfileDialog({ candidate }: { candidate: SearchCandidate }) {
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [profile, setProfile] = useState<Profile>()

  async function changeOpen(next: boolean) {
    setOpen(next)
    if (!next || profile || !candidate.authorId) return
    setLoading(true)
    try { setProfile(await get<Profile>(`/api/v1/xhs/users/${encodeURIComponent(candidate.authorId)}?candidateId=${encodeURIComponent(candidate.id)}`)) }
    catch (error) { notify.error(error, '获取小红书用户信息失败') } finally { setLoading(false) }
  }

  return <Dialog.Root open={open} onOpenChange={changeOpen}>
    <Dialog.Trigger><Button variant="ghost" size="1" disabled={!candidate.authorId}>{candidate.authorName || '未知作者'}</Button></Dialog.Trigger>
    <Dialog.Content maxWidth="520px">
      <Dialog.Title>小红书用户详情</Dialog.Title>
      {loading && <Text color="gray">正在通过搜索设备查询用户资料…</Text>}
      {!loading && profile && <Flex direction="column" gap="4">
        <Flex gap="3" align="center">
          {profile.avatarUrl ? <div className="profile-avatar"><AuthenticatedImage url={profile.avatarUrl} alt={profile.nickname} previewTitle={profile.nickname} /></div> : <Avatar fallback={profile.nickname.slice(0, 1)} size="6" />}
          <div><Text as="div" size="4" weight="bold">{profile.nickname}</Text><Text as="div" size="2" color="gray">小红书号：{profile.redId || '—'}</Text><Text as="div" size="1" color="gray">用户 ID：{profile.userId}</Text></div>
        </Flex>
        {profile.description && <Text size="2">{profile.description}</Text>}
        <Grid columns="4" gap="3"><Metric label="粉丝" value={profile.fansCount} /><Metric label="获赞" value={profile.likedCount} /><Metric label="收藏" value={profile.collectedCount} /><Metric label="笔记" value={profile.noteCount} /></Grid>
        <Text size="1" color="gray">IP 属地：{profile.ipLocation || '—'} · {profile.cached ? '来自本地缓存' : '刚刚通过设备更新'}</Text>
      </Flex>}
      <Flex justify="end" mt="5"><Dialog.Close><Button variant="soft" color="gray">关闭</Button></Dialog.Close></Flex>
    </Dialog.Content>
  </Dialog.Root>
}

function Metric({ label, value }: { label: string; value: number }) { return <div><Text as="div" size="1" color="gray">{label}</Text><Text size="4" weight="bold">{value}</Text></div> }
