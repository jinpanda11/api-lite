import React, { useEffect, useState } from 'react'
import { useNavigate, useLocation, Link } from 'react-router-dom'
import {
  Layout as SemiLayout,
  Nav,
  Avatar,
  Dropdown,
  Button,
  Badge,
  Typography,
} from '@douyinfe/semi-ui'
import {
  IconHome,
  IconImage,
  IconServer,
  IconSetting,
  IconMoon,
  IconSun,
  IconExit,
  IconUser,
  IconGift,
  IconBell,
  IconPriceTag,
  IconHistory,
} from '@douyinfe/semi-icons'
import { useAppStore } from '../store'
import { getUserInfo, getBranding, getDrawQuota } from '../api'
import NoticeModal from './NoticeModal'

const { Header, Sider, Content } = SemiLayout
const { Text } = Typography

interface LayoutProps {
  children: React.ReactNode
}

const NAV_ITEMS = [
  { itemKey: '/draw', text: 'AI 画图', icon: <IconImage /> },
  { itemKey: '/settings', text: '设置', icon: <IconSetting /> },
]

const ADMIN_NAV_ITEMS = [
  { itemKey: '/channels', text: '渠道管理', icon: <IconServer /> },
  { itemKey: '/admin/pricing', text: '模型定价', icon: <IconPriceTag /> },
  { itemKey: '/admin/notice', text: '公告管理', icon: <IconBell /> },
  { itemKey: '/admin/users', text: '用户管理', icon: <IconUser /> },
  { itemKey: '/admin/branding', text: '站点品牌', icon: <IconSetting /> },
  { itemKey: '/admin/audit', text: '审计记录', icon: <IconHistory /> },
]

export default function AppLayout({ children }: LayoutProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, loggedIn, theme, setTheme, setUser, logout, setLoggedIn } = useAppStore()
  const [branding, setBranding] = useState<Record<string, string>>({})
  const [logoError, setLogoError] = useState(false)

  useEffect(() => {
    document.body.setAttribute('theme-mode', theme)
  }, [theme])

  useEffect(() => {
    getUserInfo()
      .then((res) => {
        setUser(res.data)
        setLoggedIn(true)
      })
      .catch(() => {
        setLoggedIn(false)
        setUser(null)
      })
  }, [setUser, setLoggedIn])

  useEffect(() => {
    getBranding()
      .then((res) => { setBranding(res.data || {}); setLogoError(false) })
      .catch(() => {})
  }, [])

  useEffect(() => {
    if (user) {
      getDrawQuota()
        .then((res) => setHeaderQuota({ remaining: res.data.quota_remaining, total: res.data.quota_total }))
        .catch(() => setHeaderQuota(null))
    }
  }, [user, location.pathname])

  const navItems =
    user?.role === 'admin'
      ? [...NAV_ITEMS, ...ADMIN_NAV_ITEMS]
      : NAV_ITEMS

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [headerQuota, setHeaderQuota] = useState<{ remaining: number; total: number } | null>(null)

  const handleNavSelect = (data: any) => {
    const key = String(data.itemKey)
    if (!key.startsWith('divider-')) navigate(key)
  }

  const handleLogout = async () => {
    try {
      await fetch('/api/user/logout', { method: 'POST' })
    } catch { /* ignore network errors */ }
    logout()
    navigate('/login')
  }

  const toggleTheme = () => {
    setTheme(theme === 'dark' ? 'light' : 'dark')
  }

  return (<>
    <SemiLayout style={{ minHeight: '100vh' }}>
      <Sider style={{ background: 'var(--semi-color-bg-1)' }}>
        <Nav
          style={{ height: '100%', borderRight: '1px solid var(--semi-color-border)' }}
          selectedKeys={[location.pathname]}
          onSelect={handleNavSelect}
          items={navItems}
          header={{
            logo: (
              <div style={{ padding: '16px 8px', display: 'flex', alignItems: 'center', gap: 8 }}>
                {(() => {
                  const logo = (branding.site_logo || '').trim()
                  const logoSize = Number(branding.site_logo_size) || 20
                  if (!logo) return <span style={{ fontSize: logoSize }}>⚡</span>
                  if (logo.startsWith('http') && !logoError) {
                    return <img src={logo} alt="logo" style={{ width: logoSize, height: logoSize }}
                      onError={() => setLogoError(true)} />
                  }
                  return <span style={{ fontSize: logoSize }}>{logo.startsWith('http') ? '⚡' : logo}</span>
                })()}
                <Text strong style={{ fontSize: Number(branding.site_name_size) || 16 }}>
                  {branding.site_name || 'New API Lite'}
                </Text>
              </div>
            ),
          }}
          footer={{ collapseButton: true }}
        />
      </Sider>

      <SemiLayout>
        <Header
          style={{
            background: 'var(--semi-color-bg-1)',
            borderBottom: '1px solid var(--semi-color-border)',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: 12,
            height: 56,
          }}
        >
          {user && user.role !== 'admin' && headerQuota != null && (
            <Badge
              count={`画图 ${headerQuota.remaining}/${headerQuota.total}次`}
              type={headerQuota.remaining > 0 ? 'success' : 'danger'}
              style={{ marginRight: 8 }}
            />
          )}
          {user && user.role === 'admin' && (
            <Badge
              count="管理员"
              type="primary"
              style={{ marginRight: 8 }}
            />
          )}

          {!loggedIn && (
            <div style={{ display: 'flex', gap: 8, marginRight: 8 }}>
              <Button theme="borderless" onClick={() => navigate('/login')}>登录</Button>
              <Button type="primary" theme="solid" onClick={() => navigate('/register')}>注册</Button>
            </div>
          )}

          <Button
            icon={theme === 'dark' ? <IconSun /> : <IconMoon />}
            theme="borderless"
            onClick={toggleTheme}
          />

          <Dropdown
            trigger="click"
            position="bottomRight"
            render={
              <Dropdown.Menu>
                <Dropdown.Item icon={<IconSetting />}>
                  <Link to="/settings" style={{ color: 'inherit', textDecoration: 'none' }}>
                    设置
                  </Link>
                </Dropdown.Item>
                <Dropdown.Divider />
                <Dropdown.Item icon={<IconExit />} type="danger" onClick={handleLogout}>
                  退出登录
                </Dropdown.Item>
              </Dropdown.Menu>
            }
          >
            <Avatar size="small" color="indigo" style={{ cursor: 'pointer' }}>
              {user?.username?.charAt(0).toUpperCase() ?? 'U'}
            </Avatar>
          </Dropdown>
        </Header>

        <Content
          style={{
            padding: 24,
            background: 'var(--semi-color-bg-0)',
            minHeight: 'calc(100vh - 56px)',
          }}
        >
          {children}
        </Content>
      </SemiLayout>
    </SemiLayout>
    <NoticeModal />
  </>)
}
