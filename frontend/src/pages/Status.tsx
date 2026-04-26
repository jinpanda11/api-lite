import React, { useEffect, useState, useCallback } from 'react'
import { Card, Row, Col, Typography, Tag, Button, Switch, Toast, InputNumber, Space, Spin } from '@douyinfe/semi-ui'
import { IconRefresh } from '@douyinfe/semi-icons'
import { getStatus, getMonitorConfig, updateMonitorConfig, toggleChannelMonitor } from '../api'
import type { ChannelStatus, MonitorConfig } from '../types'
import { useAppStore } from '../store'

const { Title, Text } = Typography

export default function StatusPage() {
  const user = useAppStore((s) => s.user)
  const isAdmin = user?.role === 'admin'
  const [data, setData] = useState<ChannelStatus[]>([])
  const [interval, setInterval] = useState(5)
  const [loading, setLoading] = useState(true)
  const [config, setConfig] = useState<MonitorConfig | null>(null)
  const [editingInterval, setEditingInterval] = useState(5)

  const load = useCallback(() => {
    getStatus()
      .then((res) => {
        setData(res.data.data ?? [])
        setInterval(res.data.interval ?? 5)
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
    const timer = setInterval(load, 30000)
    return () => clearInterval(timer)
  }, [load])

  useEffect(() => {
    if (isAdmin) {
      getMonitorConfig()
        .then((res) => {
          setConfig(res.data)
          setEditingInterval(res.data.interval)
        })
        .catch(() => {})
    }
  }, [isAdmin])

  const handleToggle = async (id: number, monitored: boolean) => {
    await toggleChannelMonitor(id, !monitored)
    if (config) {
      const channels = config.channels.map((c) =>
        c.id === id ? { ...c, monitor_enabled: !c.monitor_enabled } : c
      )
      setConfig({ ...config, channels })
    }
  }

  const handleSaveInterval = async () => {
    await updateMonitorConfig(editingInterval)
    setInterval(editingInterval)
    Toast.success(`检测间隔已设为 ${editingInterval} 分钟`)
  }

  const disabledChannels = new Set(
    (config?.channels ?? []).filter((c) => !c.monitor_enabled).map((c) => c.id)
  )

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>模型连通性监控</Title>
        <Space>
          {isAdmin && (
            <>
              <InputNumber
                value={editingInterval}
                onChange={(v) => setEditingInterval(v as number)}
                min={1}
                max={1440}
                suffix="分钟"
                style={{ width: 140 }}
              />
              <Button onClick={handleSaveInterval}>保存间隔</Button>
            </>
          )}
          <Button icon={<IconRefresh />} onClick={load}>刷新</Button>
        </Space>
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', paddingTop: 80 }}><Spin size="large" /></div>
      ) : (
        <>
          <Row gutter={[12, 12]}>
            {data.map((s) => {
              const key = `${s.model}|${s.channel_id}`
              const disabled = disabledChannels.has(s.channel_id)
              return (
                <Col key={key} xs={12} sm={8} md={6} lg={4} xl={3}>
                  <Card
                    bodyStyle={{ padding: '16px 12px', textAlign: 'center' }}
                  >
                    <div
                      style={{
                        width: 10,
                        height: 10,
                        borderRadius: '50%',
                        backgroundColor: disabled
                          ? '#ccc'
                          : s.online
                            ? 'var(--semi-color-success)'
                            : 'var(--semi-color-danger)',
                        margin: '0 auto 8px',
                      }}
                    />
                    <Text strong style={{ fontSize: 13, display: 'block', marginBottom: 6 }}>{s.model}</Text>
                    {disabled ? (
                      <Text type="tertiary" size="small">已关闭</Text>
                    ) : s.online ? (
                      <Text type="success" size="small">在线</Text>
                    ) : (
                      <Text type="danger" size="small">异常</Text>
                    )}
                  </Card>
                </Col>
              )
            })}
          </Row>

          {data.length === 0 && (
            <div style={{ textAlign: 'center', paddingTop: 40 }}>
              <Text type="tertiary">暂无数据，等待首次检测...</Text>
            </div>
          )}
        </>
      )}

      {isAdmin && (config?.channels.length ?? 0) > 0 && (
        <Card style={{ marginTop: 24 }}>
          <Text strong style={{ display: 'block', marginBottom: 12 }}>渠道检测开关</Text>
          <Row gutter={[16, 8]}>
            {config?.channels.map((ch) => (
              <Col key={ch.id} xs={24} sm={12} lg={8} xl={6}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text>{ch.name}</Text>
                  <Switch
                    checked={ch.monitor_enabled}
                    onChange={() => handleToggle(ch.id, ch.monitor_enabled)}
                  />
                </div>
              </Col>
            ))}
          </Row>
        </Card>
      )}
    </div>
  )
}
