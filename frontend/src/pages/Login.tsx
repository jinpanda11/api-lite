import React, { useEffect, useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Form, Button, Card, Typography, Toast } from '@douyinfe/semi-ui'
import { login, getBranding } from '../api'
import { useAppStore } from '../store'

const { Title, Text } = Typography

export default function Login() {
  const navigate = useNavigate()
  const { setLoggedIn, setUser } = useAppStore()
  const [loading, setLoading] = useState(false)
  const [branding, setBranding] = useState<Record<string, string>>({})
  const [logoError, setLogoError] = useState(false)

  useEffect(() => {
    getBranding()
      .then((res) => {
        const d = res.data || {}
        setBranding(d)
        setLogoError(false)
        if (d.site_title) document.title = d.site_title
      })
      .catch(() => {})
  }, [])

  const handleSubmit = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const res = await login(values.username, values.password)
      setLoggedIn(true)
      setUser(res.data.user)
      Toast.success('登录成功')
      navigate('/dashboard')
    } catch {
      // Error toast handled by axios interceptor
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'var(--semi-color-bg-0)',
      }}
    >
      <Card style={{ width: 400 }} bodyStyle={{ padding: 32 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          {(() => {
            const logo = (branding.site_logo || '').trim()
            const logoSize = Number(branding.site_logo_size) || 48
            if (!logo) return <span style={{ fontSize: logoSize }}>⚡</span>
            if (logo.startsWith('http') && !logoError) {
              return <img src={logo} alt="logo" style={{ width: logoSize, height: logoSize }}
                onError={() => setLogoError(true)} />
            }
            return <span style={{ fontSize: logoSize }}>{logo.startsWith('http') ? '⚡' : logo}</span>
          })()}
          <Title heading={3} style={{ marginTop: 8, marginBottom: 4, fontSize: Number(branding.site_name_size) || undefined }}>
            {branding.site_name || 'New API Lite'}
          </Title>
          <Text type="tertiary">登录你的账号</Text>
        </div>

        <Form onSubmit={handleSubmit}>
          <Form.Input
            field="username"
            label="用户名"
            placeholder="请输入用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          />
          <Form.Input
            field="password"
            label="密码"
            mode="password"
            placeholder="请输入密码"
            rules={[{ required: true, message: '请输入密码' }]}
          />
          <Button
            htmlType="submit"
            type="primary"
            theme="solid"
            block
            loading={loading}
            style={{ marginTop: 8 }}
          >
            登录
          </Button>
        </Form>

        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Text type="tertiary">还没有账号？</Text>
          <Link to="/register" style={{ marginLeft: 4 }}>
            立即注册
          </Link>
        </div>
      </Card>
    </div>
  )
}
