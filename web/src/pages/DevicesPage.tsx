import { useState } from 'react'
import { Flex, Card, Table, Button, Badge, Text, Select, Spinner, Code, AlertDialog } from '@radix-ui/themes'
import { ReloadIcon, TrashIcon } from '@radix-ui/react-icons'
import { get, del } from '../api/client'
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
                  <Table.Cell colSpan={8}>
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
