import React, { useState, useEffect } from 'react'
import {
  Typography,
  Card,
  Form,
  Button,
  Switch,
  RadioGroup,
  Radio,
  Toast,
  Divider,
  Spin,
  Input,
} from '@douyinfe/semi-ui'
import { updatePassword, getSettings, updateSettings } from '../api'
import { useAppStore } from '../store'

const { Title, Text } = Typography

export default function Settings() {
  const { user, theme, setTheme, language, setLanguage, logout } = useAppStore()
  const [pwLoading, setPwLoading] = useState(false)
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const isAdmin = user?.role === 'admin'

  useEffect(() => {
    if (isAdmin) {
      loadSettings()
    }
  }, [isAdmin])

  const loadSettings = async () => {
    setSettingsLoading(true)
    try {
      const res = await getSettings()
      setSettings(res.data.data || {})
    } catch {
      // handled by interceptor
    } finally {
      setSettingsLoading(false)
    }
  }

  const handleSaveSettings = async () => {
    setSaving(true)
    try {
      // Don't send masked sensitive values back
      const payload = { ...settings }
      if (payload.smtp_password === '****') delete payload.smtp_password
      await updateSettings(payload)
      Toast.success('设置已保存')
    } catch {
      // handled by interceptor
    } finally {
      setSaving(false)
    }
  }

  const updateSetting = (key: string, value: string) => {
    setSettings((prev) => ({ ...prev, [key]: value }))
  }

  const handlePasswordSubmit = async (values: {
    old_password: string
    new_password: string
    confirm_password: string
  }) => {
    if (values.new_password !== values.confirm_password) {
      Toast.error('两次密码输入不一致')
      return
    }
    setPwLoading(true)
    try {
      await updatePassword(values.old_password, values.new_password)
      Toast.success('密码已更新')
    } catch {
      // handled by interceptor
    } finally {
      setPwLoading(false)
    }
  }

  return (
    <div style={{ maxWidth: 640 }}>
      <Title heading={4} style={{ marginBottom: 24 }}>设置</Title>

      {/* Profile */}
      <Card title="个人信息" style={{ marginBottom: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <div>
            <Text type="tertiary" size="small">用户名</Text>
            <Text strong style={{ display: 'block', marginTop: 4 }}>{user?.username}</Text>
          </div>
          <div>
            <Text type="tertiary" size="small">邮箱</Text>
            <Text strong style={{ display: 'block', marginTop: 4 }}>{user?.email || '未绑定'}</Text>
          </div>
          <div>
            <Text type="tertiary" size="small">角色</Text>
            <Text strong style={{ display: 'block', marginTop: 4 }}>
              {user?.role === 'admin' ? '管理员' : '普通用户'}
            </Text>
          </div>
          <div>
            <Text type="tertiary" size="small">用户 ID</Text>
            <Text strong style={{ display: 'block', marginTop: 4 }}>#{user?.id}</Text>
          </div>
        </div>
      </Card>

      {/* Change Password */}
      <Card title="修改密码" style={{ marginBottom: 16 }}>
        <Form onSubmit={handlePasswordSubmit}>
          <Form.Input
            field="old_password"
            label="当前密码"
            mode="password"
            placeholder="输入当前密码"
            rules={[{ required: true }]}
          />
          <Form.Input
            field="new_password"
            label="新密码"
            mode="password"
            placeholder="至少 8 个字符"
            rules={[{ required: true }, { min: 8, message: '至少 8 个字符' }]}
          />
          <Form.Input
            field="confirm_password"
            label="确认新密码"
            mode="password"
            placeholder="再次输入新密码"
            rules={[{ required: true }]}
          />
          <Button htmlType="submit" type="primary" theme="solid" loading={pwLoading}>
            更新密码
          </Button>
        </Form>
      </Card>

      {/* Admin Settings */}
      {isAdmin && (
        <Card title="系统设置" style={{ marginBottom: 16 }}>
          {settingsLoading ? (
            <div style={{ textAlign: 'center', padding: 24 }}>
              <Spin />
            </div>
          ) : (
            <>
              {/* Email Verification Toggle */}
              <div style={{ marginBottom: 24 }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <div>
                    <Text strong>邮箱验证</Text>
                    <Text type="tertiary" style={{ display: 'block', fontSize: 12 }}>
                      注册时是否需要验证邮箱
                    </Text>
                  </div>
                  <Switch
                    checked={settings['email_verification_enabled'] !== 'false'}
                    onChange={(checked) => updateSetting('email_verification_enabled', checked ? 'true' : 'false')}
                  />
                </div>
              </div>

              {/* Register Bonus */}
              <div style={{ marginTop: 24, marginBottom: 24 }}>
                <Text strong>新用户注册赠送</Text>
                <Text type="tertiary" style={{ display: 'block', fontSize: 12, marginBottom: 8 }}>
                  新注册用户自动获得的初始余额，设置为 0 则不赠送
                </Text>
                <Input
                  placeholder="0.01"
                  value={settings['register_bonus_balance'] || '0'}
                  onChange={(v) => updateSetting('register_bonus_balance', v)}
                  style={{ width: 200 }}
                  type="number"
                  suffix={<Text type="tertiary">$</Text>}
                />
              </div>

              {/* Redeem Purchase URL */}
              <div style={{ marginTop: 24, marginBottom: 24 }}>
                <Text strong>购买兑换码链接</Text>
                <Text type="tertiary" style={{ display: 'block', fontSize: 12, marginBottom: 8 }}>
                  用户在钱包页面点击「购买兑换码」时跳转的链接
                </Text>
                <Input
                  placeholder="https://example.com/buy"
                  value={settings['redeem_purchase_url'] || ''}
                  onChange={(v) => updateSetting('redeem_purchase_url', v)}
                  style={{ width: '100%' }}
                />
              </div>

              <Divider />

              {/* SMTP Configuration */}
              <div style={{ marginTop: 24 }}>
                <Title heading={6} style={{ marginBottom: 16 }}>SMTP 邮件配置</Title>

                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                  <div>
                    <Text strong size="small">SMTP 主机</Text>
                    <Input
                      placeholder="smtp.example.com"
                      value={settings['smtp_host'] || ''}
                      onChange={(v) => updateSetting('smtp_host', v)}
                      style={{ width: '100%', marginTop: 4 }}
                    />
                  </div>
                  <div>
                    <Text strong size="small">端口</Text>
                    <Input
                      placeholder="465"
                      value={settings['smtp_port'] || ''}
                      onChange={(v) => updateSetting('smtp_port', v)}
                      style={{ width: '100%', marginTop: 4 }}
                    />
                  </div>
                  <div>
                    <Text strong size="small">用户名</Text>
                    <Input
                      placeholder="noreply@example.com"
                      value={settings['smtp_username'] || ''}
                      onChange={(v) => updateSetting('smtp_username', v)}
                      style={{ width: '100%', marginTop: 4 }}
                    />
                  </div>
                  <div>
                    <Text strong size="small">密码</Text>
                    <Input
                      type="password"
                      placeholder="SMTP 密码"
                      value={settings['smtp_password'] || ''}
                      onChange={(v) => updateSetting('smtp_password', v)}
                      style={{ width: '100%', marginTop: 4 }}
                    />
                  </div>
                  <div style={{ gridColumn: '1 / -1' }}>
                    <Text strong size="small">发件人地址</Text>
                    <Input
                      placeholder='New API Lite <noreply@example.com>'
                      value={settings['smtp_from'] || ''}
                      onChange={(v) => updateSetting('smtp_from', v)}
                      style={{ width: '100%', marginTop: 4 }}
                    />
                  </div>
                </div>

                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 12 }}>
                  <div>
                    <Text strong size="small">SSL</Text>
                    <Text type="tertiary" style={{ display: 'block', fontSize: 12 }}>
                      使用 SSL/TLS 加密连接
                    </Text>
                  </div>
                  <Switch
                    checked={settings['smtp_ssl'] === 'true'}
                    onChange={(checked) => updateSetting('smtp_ssl', checked ? 'true' : 'false')}
                  />
                </div>
              </div>

              <Divider />

              <Button
                type="primary"
                theme="solid"
                loading={saving}
                onClick={handleSaveSettings}
                style={{ marginTop: 8 }}
              >
                保存设置
              </Button>
            </>
          )}
        </Card>
      )}

      {/* Appearance */}
      <Card title="外观" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
          <div>
            <Text strong>深色模式</Text>
            <Text type="tertiary" style={{ display: 'block', fontSize: 12 }}>
              切换界面明暗主题
            </Text>
          </div>
          <Switch
            checked={theme === 'dark'}
            onChange={(checked) => setTheme(checked ? 'dark' : 'light')}
          />
        </div>

        <Divider />

        <div style={{ marginTop: 16 }}>
          <Text strong>语言</Text>
          <RadioGroup
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            style={{ marginTop: 8, display: 'flex', gap: 16 }}
          >
            <Radio value="zh">中文</Radio>
            <Radio value="en">English</Radio>
          </RadioGroup>
        </div>
      </Card>

      {/* Danger Zone */}
      <Card title="账号操作">
        <Button
          type="danger"
          theme="light"
          onClick={async () => {
            try { await fetch('/api/user/logout', { method: 'POST' }) } catch {}
            logout()
            window.location.href = '/login'
          }}
        >
          退出登录
        </Button>
      </Card>
    </div>
  )
}
