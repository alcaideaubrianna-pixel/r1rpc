import { NavLink, Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Box, Flex, Text, Heading, Button, Avatar, Separator } from '@radix-ui/themes'
import {
  DashboardIcon,
  LayersIcon,
  MobileIcon,
  TargetIcon,
  ReaderIcon,
  RocketIcon,
  PersonIcon,
  ExitIcon,
  FileIcon,
  ImageIcon,
} from '@radix-ui/react-icons'
import { auth } from './auth'

const NAV = [
  { to: '/overview', label: '概览', icon: <DashboardIcon /> },
  { to: '/groups', label: '分组', icon: <LayersIcon /> },
  { to: '/devices', label: '设备', icon: <MobileIcon /> },
  { to: '/clients', label: '在线客户端', icon: <TargetIcon /> },
  { to: '/requests', label: '调用记录', icon: <ReaderIcon /> },
  { to: '/invoke', label: 'RPC 调用', icon: <RocketIcon /> },
  { to: '/files', label: '文件管理', icon: <FileIcon /> },
  { to: '/image-batches', label: '批量识图', icon: <ImageIcon /> },
  { to: '/image-search', label: '图片搜索', icon: <ImageIcon /> },
  { to: '/users', label: '账号', icon: <PersonIcon /> },
]

const panel = { background: 'var(--color-panel-solid)' }

export default function Shell() {
  const nav = useNavigate()
  const loc = useLocation()
  const user = auth.user
  const current = NAV.find((n) => loc.pathname.startsWith(n.to))

  return (
    <Flex style={{ height: '100vh' }}>
      {/* 侧边栏 */}
      <Flex
        direction="column"
        style={{ width: 232, flexShrink: 0, borderRight: '1px solid var(--gray-4)', ...panel }}
      >
        <Flex align="center" gap="1" px="4" style={{ height: 60, flexShrink: 0 }}>
          <img src={`${import.meta.env.BASE_URL}logo.svg`} alt="r1rpc" width={32} height={32} />
          <Heading size="4" weight="bold">
            r1rpc
          </Heading>
        </Flex>
        <Separator size="4" />
        <Box p="3" style={{ flex: 1, overflow: 'auto' }}>
          <Flex direction="column" gap="1">
            {NAV.map((item) => (
              <NavLink key={item.to} to={item.to}>
                {({ isActive }) => (
                  <Flex
                    align="center"
                    gap="3"
                    px="3"
                    py="2"
                    style={{
                      borderRadius: 'var(--radius-3)',
                      background: isActive ? 'var(--accent-3)' : 'transparent',
                    }}
                  >
                    <Flex
                      align="center"
                      style={{ color: isActive ? 'var(--accent-11)' : 'var(--gray-9)' }}
                    >
                      {item.icon}
                    </Flex>
                    <Text
                      size="2"
                      weight={isActive ? 'medium' : 'regular'}
                      style={{ color: isActive ? 'var(--accent-11)' : 'var(--gray-11)' }}
                    >
                      {item.label}
                    </Text>
                  </Flex>
                )}
              </NavLink>
            ))}
          </Flex>
        </Box>
      </Flex>

      {/* 主区 */}
      <Flex direction="column" style={{ flex: 1, minWidth: 0 }}>
        <Flex
          align="center"
          justify="between"
          px="5"
          style={{ height: 60, flexShrink: 0, borderBottom: '1px solid var(--gray-4)', ...panel }}
        >
          <Heading size="4">{current?.label ?? ''}</Heading>
          <Flex align="center" gap="4">
            <Flex align="center" gap="2">
              <Avatar
                size="1"
                radius="full"
                fallback={(user?.username ?? '?').slice(0, 1).toUpperCase()}
              />
              <Text size="2" color="gray">
                {user?.username ?? ''}
              </Text>
            </Flex>
            <Button
              variant="soft"
              color="gray"
              size="1"
              onClick={() => {
                auth.logout()
                nav('/login', { replace: true })
              }}
            >
              <ExitIcon /> 退出
            </Button>
          </Flex>
        </Flex>

        <Box p="5" style={{ flex: 1, overflow: 'auto', background: 'var(--gray-2)' }}>
          <div key={loc.pathname} className="page-enter">
            <Outlet />
          </div>
        </Box>
      </Flex>
    </Flex>
  )
}
