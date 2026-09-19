import { useEffect, useState } from 'react'
import { Button, Callout, Dialog, Flex, Select, Switch, Text, TextField } from '@radix-ui/themes'
import { GearIcon, InfoCircledIcon } from '@radix-ui/react-icons'
import { get, put } from '../../api/client'
import { notify } from '../../lib/toast'

interface StorageSetting {
  backend: 'local' | 'cos' | 'oss'
  localPath: string
  endpoint: string
  region: string
  bucket: string
  pathStyle: boolean
  accessKeyConfigured: boolean
  secretKeyConfigured: boolean
  driverAvailable: boolean
}

const empty: StorageSetting = { backend: 'local', localPath: './data/files', endpoint: '', region: '', bucket: '', pathStyle: true, accessKeyConfigured: false, secretKeyConfigured: false, driverAvailable: true }

export function StorageSettingsDialog() {
  const [open, setOpen] = useState(false)
  const [value, setValue] = useState<StorageSetting>(empty)
  const [accessKey, setAccessKey] = useState('')
  const [secretKey, setSecretKey] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => { if (open) get<StorageSetting>('/api/v1/storage/settings').then(setValue).catch((e) => notify.error(e, '读取存储配置失败')) }, [open])
  const field = (key: keyof StorageSetting, next: string | boolean) => setValue((current) => ({ ...current, [key]: next }))

  async function save() {
    setSaving(true)
    try {
      const result = await put<StorageSetting>('/api/v1/storage/settings', { ...value, accessKey, secretKey })
      setValue(result); setAccessKey(''); setSecretKey('')
      notify.success('配置已保存，重启服务后应用新后端')
      setOpen(false)
    } catch (error) { notify.error(error, '保存存储配置失败') } finally { setSaving(false) }
  }

  return <Dialog.Root open={open} onOpenChange={setOpen}>
    <Dialog.Trigger><Button variant="soft" color="gray"><GearIcon /> 存储配置</Button></Dialog.Trigger>
    <Dialog.Content maxWidth="560px">
      <Dialog.Title>存储配置</Dialog.Title>
      <Dialog.Description size="2" color="gray">配置只保存连接参数；切换后端需重启服务。</Dialog.Description>
      <Flex direction="column" gap="3" mt="4">
        <label><Text as="div" size="2" mb="1">存储类型</Text><Select.Root value={value.backend} onValueChange={(v) => field('backend', v)}><Select.Trigger style={{ width: '100%' }} /><Select.Content><Select.Item value="local">Local 本地存储</Select.Item><Select.Item value="cos">腾讯云 COS（待启用驱动）</Select.Item><Select.Item value="oss">阿里云 OSS（待启用驱动）</Select.Item></Select.Content></Select.Root></label>
        {value.backend === 'local' ? <Field label="本地目录" value={value.localPath} onChange={(v) => field('localPath', v)} placeholder="./data/files" /> : <>
          <Callout.Root color="amber" size="1"><Callout.Icon><InfoCircledIcon /></Callout.Icon><Callout.Text>当前版本尚未加载 {value.backend.toUpperCase()} 驱动，配置可提前保存，但不能立即用于文件读写。</Callout.Text></Callout.Root>
          <Field label="Endpoint" value={value.endpoint} onChange={(v) => field('endpoint', v)} placeholder="https://cos.ap-guangzhou.myqcloud.com" />
          <Flex gap="3"><div style={{ flex: 1 }}><Field label="Region" value={value.region} onChange={(v) => field('region', v)} placeholder="ap-guangzhou" /></div><div style={{ flex: 1 }}><Field label="Bucket" value={value.bucket} onChange={(v) => field('bucket', v)} placeholder="bucket-name" /></div></Flex>
          <Field label={`Access Key${value.accessKeyConfigured ? '（已配置，留空不修改）' : ''}`} value={accessKey} onChange={setAccessKey} password />
          <Field label={`Secret Key${value.secretKeyConfigured ? '（已配置，留空不修改）' : ''}`} value={secretKey} onChange={setSecretKey} password />
          <Flex align="center" gap="2"><Switch checked={value.pathStyle} onCheckedChange={(v) => field('pathStyle', v)} /><Text size="2">Path Style 地址</Text></Flex>
        </>}
      </Flex>
      <Flex gap="3" mt="5" justify="end"><Dialog.Close><Button variant="soft" color="gray">取消</Button></Dialog.Close><Button loading={saving} onClick={save}>保存配置</Button></Flex>
    </Dialog.Content>
  </Dialog.Root>
}

function Field({ label, value, onChange, placeholder, password }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string; password?: boolean }) {
  return <label><Text as="div" size="2" mb="1">{label}</Text><TextField.Root type={password ? 'password' : 'text'} value={value} placeholder={placeholder} onChange={(e) => onChange(e.target.value)} /></label>
}
