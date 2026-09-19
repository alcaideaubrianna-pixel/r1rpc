import { useEffect, useMemo, useState } from 'react'
import { Flex, Card, Table, Button, Badge, Text, Select, Spinner, Code, AlertDialog, Dialog, TextField, Callout } from '@radix-ui/themes'
import { MobileIcon, ReloadIcon, TrashIcon } from '@radix-ui/react-icons'
import { QRCodeSVG } from 'qrcode.react'
import { get, del, post } from '../api/client'
import { useFetch } from '../lib/useFetch'
import { notify } from '../lib/toast'
import { fmtTime } from '../lib/format'
import type { Device } from '../types'

export default function DevicesPage() {
  const [status, setStatus] = useState('all')
  const query = status === 'all' ? '' : `?status=${status}`
  const { data, loading, reload } = useFetch(
    () => get<{ items: Device[] }>(`/api/devices${query}`),
    [status],
  )
  const devices = data?.items ?? []

  function refresh() {
    reload()
    notify.success('已刷新')
  }

  return (
    <Flex direction="column" gap="4">
      <Flex justify="between" align="center">
        <Text size="2" color="gray">
          在线状态由 Hub 实时会话决定
        </Text>
        <Flex gap="2" align="center">
          <Select.Root value={status} onValueChange={setStatus}>
            <Select.Trigger />
            <Select.Content>
              <Select.Item value="all">全部状态</Select.Item>
              <Select.Item value="online">在线</Select.Item>
              <Select.Item value="offline">离线</Select.Item>
            </Select.Content>
          </Select.Root>
          <DebugPairingButton onPaired={reload} />
          <Button variant="soft" color="gray" onClick={refresh}>
            <ReloadIcon /> 刷新
          </Button>
        </Flex>
      </Flex>

      <Card size="2">
        {loading ? (
          <Flex justify="center" p="6">
            <Spinner size="3" />
          </Flex>
        ) : (
          <Table.Root variant="surface">
            <Table.Header>
              <Table.Row>
                <Table.ColumnHeaderCell>客户端 ID</Table.ColumnHeaderCell>
                <Table.ColumnHeaderCell>分组</Table.ColumnHeaderCell>
                <Table.ColumnHeaderCell>设备型号</Table.ColumnHeaderCell>
                <Table.ColumnHeaderCell>系统版本</Table.ColumnHeaderCell>
                <Table.ColumnHeaderCell>SDK 版本</Table.ColumnHeaderCell>
                <Table.ColumnHeaderCell>状态</Table.ColumnHeaderCell>
                <Table.ColumnHeaderCell>最后在线</Table.ColumnHeaderCell>
                <Table.ColumnHeaderCell>IP</Table.ColumnHeaderCell>
                <Table.ColumnHeaderCell>操作</Table.ColumnHeaderCell>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {devices.map((d) => (
                <Table.Row key={d.clientId} align="center">
                  <Table.RowHeaderCell>
                    <Code variant="ghost">{d.clientId}</Code>
                  </Table.RowHeaderCell>
                  <Table.Cell>{d.group}</Table.Cell>
                  <Table.Cell>
                    <Flex direction="column" gap="1">
                      <Text size="2">{d.deviceModel || d.deviceMachine || '—'}</Text>
                      {d.deviceMachine && d.deviceModel !== d.deviceMachine && (
                        <Text size="1" color="gray">{d.deviceMachine}</Text>
                      )}
                    </Flex>
                  </Table.Cell>
                  <Table.Cell>{[d.osName, d.osVersion].filter(Boolean).join(' ') || '—'}</Table.Cell>
                  <Table.Cell>{[d.sdkName, d.sdkVersion].filter(Boolean).join(' ') || '—'}</Table.Cell>
                  <Table.Cell>
                    <Badge color={d.status === 'online' ? 'green' : 'gray'} variant="soft">
                      {d.status === 'online' ? '在线' : '离线'}
                    </Badge>
                  </Table.Cell>
                  <Table.Cell>
                    <Text size="2" color="gray">
                      {fmtTime(d.lastSeenAt)}
                    </Text>
                  </Table.Cell>
                  <Table.Cell>
                    <Text size="2" color="gray">
                      {d.lastIp || '—'}
                    </Text>
                  </Table.Cell>
                  <Table.Cell>
                    <DeleteDeviceButton device={d} onDeleted={reload} />
                  </Table.Cell>
                </Table.Row>
              ))}
              {devices.length === 0 && (
                <Table.Row>
                  <Table.Cell colSpan={9}>
                    <Text color="gray">暂无设备</Text>
                  </Table.Cell>
                </Table.Row>
              )}
            </Table.Body>
          </Table.Root>
        )}
      </Card>
    </Flex>
  )
}

interface DebugPairing {
  id: string
  code: string
  expiresAt: string
}

interface DebugPairingStatus {
  claimed: boolean
  clientId?: string
}

function DebugPairingButton({ onPaired }: { onPaired: () => void }) {
  const [open, setOpen] = useState(false)
  const [server, setServer] = useState(window.location.origin)
  const [pairing, setPairing] = useState<DebugPairing | null>(null)
  const [pairedClient, setPairedClient] = useState('')
  const [busy, setBusy] = useState(false)

  const payload = useMemo(() => pairing ? JSON.stringify({
    type: 'r1rpc-debug-pair',
    version: 1,
    server: server.replace(/\/$/, ''),
    code: pairing.code,
  }) : '', [pairing, server])

  const localOnly = /^(https?:\/\/)?(localhost|127\.0\.0\.1)(:|\/|$)/i.test(server)

  async function createPairing() {
    setBusy(true)
    setPairedClient('')
    try {
      setPairing(await post<DebugPairing>('/api/devices/debug-pairings', {}))
    } catch (error) {
      notify.error(error, '创建配对码失败')
    } finally {
      setBusy(false)
    }
  }

  function changeOpen(next: boolean) {
    setOpen(next)
    if (next) void createPairing()
    else {
      setPairing(null)
      setPairedClient('')
    }
  }

  useEffect(() => {
    if (!open || !pairing || pairedClient) return
    const timer = window.setInterval(async () => {
      try {
        const status = await get<DebugPairingStatus>(`/api/devices/debug-pairings/${encodeURIComponent(pairing.id)}`)
        if (status.claimed) {
          setPairedClient(status.clientId || '已注册设备')
          onPaired()
          notify.success(`设备 ${status.clientId || ''} 注册成功`)
        }
      } catch {
        window.clearInterval(timer)
      }
    }, 1500)
    return () => window.clearInterval(timer)
  }, [open, pairing, pairedClient, onPaired])

  return (
    <Dialog.Root open={open} onOpenChange={changeOpen}>
      <Dialog.Trigger>
        <Button><MobileIcon /> 调试设备注册</Button>
      </Dialog.Trigger>
      <Dialog.Content maxWidth="440px">
        <Dialog.Title>调试设备注册</Dialog.Title>
        <Flex direction="column" gap="3" mt="3">
          <label>
            <Text as="div" size="2" weight="medium" mb="1">设备可访问地址</Text>
            <TextField.Root value={server} onChange={(event) => setServer(event.target.value)} placeholder="http://192.168.1.10:9876" />
          </label>
          {localOnly && (
            <Callout.Root color="amber" size="1">
              <Callout.Text>手机无法访问 localhost，请改为电脑的局域网 IP。</Callout.Text>
            </Callout.Root>
          )}
          <Flex justify="center" align="center" style={{ minHeight: 248 }}>
            {busy && <Spinner size="3" />}
            {!busy && pairing && !pairedClient && (
              <QRCodeSVG value={payload} size={232} level="M" marginSize={2} />
            )}
            {pairedClient && (
              <Flex direction="column" align="center" gap="2">
                <Badge color="green" size="2">注册成功</Badge>
                <Code>{pairedClient}</Code>
              </Flex>
            )}
          </Flex>
          {pairing && !pairedClient && (
            <Text size="1" color="gray" align="center">
              配对码将在 {fmtTime(pairing.expiresAt)} 过期
            </Text>
          )}
        </Flex>
        <Flex gap="3" mt="4" justify="end">
          <Button variant="soft" color="gray" onClick={createPairing} loading={busy}>刷新二维码</Button>
          <Dialog.Close><Button variant="soft">关闭</Button></Dialog.Close>
        </Flex>
      </Dialog.Content>
    </Dialog.Root>
  )
}

function DeleteDeviceButton({ device, onDeleted }: { device: Device; onDeleted: () => void }) {
  const [busy, setBusy] = useState(false)
  const online = device.status === 'online'

  async function remove() {
    setBusy(true)
    try {
      await del(`/api/devices/${encodeURIComponent(device.clientId)}`)
      notify.success(`已删除设备 ${device.clientId}`)
      onDeleted()
    } catch (e) {
      notify.error(e)
    } finally {
      setBusy(false)
    }
  }

  if (online) {
    return (
      <Button size="1" variant="ghost" color="gray" disabled title="在线设备无法删除">
        <TrashIcon /> 删除
      </Button>
    )
  }

  return (
    <AlertDialog.Root>
      <AlertDialog.Trigger>
        <Button size="1" variant="ghost" color="red">
          <TrashIcon /> 删除
        </Button>
      </AlertDialog.Trigger>
      <AlertDialog.Content maxWidth="400px">
        <AlertDialog.Title>删除设备</AlertDialog.Title>
        <AlertDialog.Description size="2">
          确认从名册删除离线设备「{device.clientId}」？该设备若重新登录会再次出现。
        </AlertDialog.Description>
        <Flex gap="3" mt="4" justify="end">
          <AlertDialog.Cancel>
            <Button variant="soft" color="gray">
              取消
            </Button>
          </AlertDialog.Cancel>
          <AlertDialog.Action>
            <Button color="red" loading={busy} onClick={remove}>
              删除
            </Button>
          </AlertDialog.Action>
        </Flex>
      </AlertDialog.Content>
    </AlertDialog.Root>
  )
}
