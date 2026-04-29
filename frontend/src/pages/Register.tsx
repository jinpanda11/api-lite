import React, { useState, useRef, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Form, Button, Card, Typography, Toast } from '@douyinfe/semi-ui'
import { register, sendVerificationCode, getEmailVerificationStatus, getBranding } from '../api'

const { Title, Text } = Typography

export default function Register() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [sending, setSending] = useState(false)
  const [countdown, setCountdown] = useState(0)
  const [emailVerificationEnabled, setEmailVerificationEnabled] = useState(true)
  const [branding, setBranding] = useState<Record<string, string>>({})
  const [logoError, setLogoError] = useState(false)
  const formRef = useRef<any>(null)

  useEffect(() => {
    getEmailVerificationStatus()
      .then((res) => setEmailVerificationEnabled(res.data.enabled !== false))
      .catch(() => {})
    getBranding()
      .then((res) => {
        const d = res.data || {}
        setBranding(d)
        setLogoError(false)
        if (d.site_title) document.title = d.site_title
      })
      .catch(() => {})
  }, [])

  const startCountdown = () => {
    setCountdown(60)
    const timer = setInterval(() => {
      setCountdown((c) => {
        if (c <= 1) {
          clearInterval(timer)
          return 0
        }
        return c - 1
      })
    }, 1000)
  }

  const handleSendCode = async () => {
    const email = formRef.current?.formApi?.getValue('email')
    if (!email) {
      Toast.warning('请先输入邮箱地址')
      return
    }
    setSending(true)
    try {
      await sendVerificationCode(email)
      Toast.success('验证码已发送，请查收邮件')
      startCountdown()
    } catch {
      // handled by interceptor
    } finally {
      setSending(false)
    }
  }

  const handleSubmit = async (values: {
    username: string
    email: string
    password: string
    confirm_password: string
    code: string
  }) => {
    if (values.password !== values.confirm_password) {
      Toast.error('两次密码输入不一致')
      return
    }
    setLoading(true)
    try {
      await register({
        username: values.username,
        email: values.email,
        password: values.password,
        code: values.code,
      })
      Toast.success('注册成功，请登录')
      navigate('/login')
    } catch {
      // handled by interceptor
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
      <Card style={{ width: 440 }} bodyStyle={{ padding: 32 }}>
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
          <Title heading={3} style={{ marginTop: 8, marginBottom: 4 }}>
            创建账号
          </Title>
          <Text type="tertiary" style={{ fontSize: Number(branding.site_name_size) || undefined }}>注册{branding.site_name ? ' ' + branding.site_name : ' New API Lite'}</Text>
        </div>

        <Form ref={formRef} onSubmit={handleSubmit}>
          <Form.Input
            field="username"
            label="用户名"
            placeholder="3-32 个字符"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少 3 个字符' },
            ]}
          />
          <Form.Input
            field="email"
            label="邮箱"
            placeholder="your@email.com"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          />
          <Form.Input
            field="password"
            label="密码"
            mode="password"
            placeholder="至少 8 个字符"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 8, message: '密码至少 8 个字符' },
            ]}
          />
          <Form.Input
            field="confirm_password"
            label="确认密码"
            mode="password"
            placeholder="再次输入密码"
            rules={[{ required: true, message: '请确认密码' }]}
          />

          {emailVerificationEnabled && (
            <Form.Input
              field="code"
              label="邮箱验证码"
              placeholder="6 位验证码"
              rules={[{ required: true, message: '请输入验证码' }]}
              suffix={
                <Button
                  size="small"
                  disabled={countdown > 0 || sending}
                  loading={sending}
                  onClick={handleSendCode}
                  style={{ marginRight: -8 }}
                >
                  {countdown > 0 ? `${countdown}s` : '发送验证码'}
                </Button>
              }
            />
          )}

          <Button
            htmlType="submit"
            type="primary"
            theme="solid"
            block
            loading={loading}
            style={{ marginTop: 8 }}
          >
            注册
          </Button>
        </Form>

        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Text type="tertiary">已有账号？</Text>
          <Link to="/login" style={{ marginLeft: 4 }}>
            立即登录
          </Link>
        </div>
      </Card>
    </div>
  )
}
