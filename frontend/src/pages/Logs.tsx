import React, { useEffect, useState, useCallback } from 'react'
import { Typography, Table, Select, Button, Space, Tag, DatePicker } from '@douyinfe/semi-ui'
import { IconRefresh } from '@douyinfe/semi-icons'
import { getLogs, getDailyCosts } from '../api'
import type { Log } from '../types'
import { useAppStore } from '../store'

const { Title, Text } = Typography

interface DailyCostItem {
  date: string
  total_cost: number
  request_count: number
}

export default function Logs() {
  const [logs, setLogs] = useState<Log[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [modelFilter, setModelFilter] = useState('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)
  const [dailyCosts, setDailyCosts] = useState<DailyCostItem[]>([])
  const user = useAppStore((s) => s.user)
  const isAdmin = user?.role === 'admin'

  const pageSize = 20

  const load = useCallback(() => {
    setLoading(true)
    const params: Record<string, any> = { page, page_size: pageSize }
    if (modelFilter) params.model = modelFilter
    if (dateRange) {
      params.start_time = dateRange[0].toISOString()
      params.end_time = dateRange[1].toISOString()
    }
    getLogs(params)
      .then((res) => {
        setLogs(res.data.data ?? [])
        setTotal(res.data.total ?? 0)
      })
      .finally(() => setLoading(false))
  }, [page, modelFilter, dateRange])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    if (!isAdmin) return
    getDailyCosts()
      .then((res) => setDailyCosts(res.data.data ?? []))
      .catch(() => {})
  }, [isAdmin])

  const statusMap: Record<number, { color: string; text: string }> = {
    1: { color: 'green', text: '成功' },
    2: { color: 'red', text: '失败' },
  }

  const columns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      render: (v: string) => new Date(v).toLocaleString(),
      width: 160,
    },
    {
      title: '模型',
      dataIndex: 'model',
      render: (v: string) => <Text code style={{ fontSize: 12 }}>{v}</Text>,
    },
    {
      title: '令牌',
      dataIndex: 'token_name',
      render: (v: string) => <Text type="tertiary">{v}</Text>,
    },
    {
      title: '渠道',
      dataIndex: 'channel_name',
      render: (v: string) => <Tag color="blue" size="small">{v}</Tag>,
    },
    {
      title: '输入 tokens',
      dataIndex: 'input_tokens',
      render: (v: number) => v.toLocaleString(),
    },
    {
      title: '输出 tokens',
      dataIndex: 'output_tokens',
      render: (v: number) => v.toLocaleString(),
    },
    {
      title: '费用',
      dataIndex: 'cost',
      render: (v: number) => `$${v.toFixed(8)}`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (v: number) => {
        const s = statusMap[v] ?? { color: 'grey', text: '未知' }
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        return <Tag color={s.color as any} size="small">{s.text}</Tag>
      },
    },
  ]

  const todayStr = new Date().toISOString().slice(0, 10)
  const todayRow = dailyCosts.find((d) => d.date === todayStr)

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>使用记录</Title>
        <Space>
          <Select
            placeholder="全部模型"
            showClear
            onChange={(v) => { setModelFilter(v as string); setPage(1) }}
            style={{ width: 200 }}
          >
            {[...new Set(logs.map((l) => l.model))].map((m) => (
              <Select.Option key={m} value={m}>{m}</Select.Option>
            ))}
          </Select>
          <DatePicker
            type="dateRange"
            placeholder={['开始日期', '结束日期']}
            onChange={(v: any) => { setDateRange(v); setPage(1) }}
            style={{ width: 260 }}
          />
          <Button icon={<IconRefresh />} onClick={() => { load(); if (isAdmin) getDailyCosts().then((res) => setDailyCosts(res.data.data ?? [])).catch(() => {}) }}>
            刷新
          </Button>
        </Space>
      </div>

      {isAdmin && dailyCosts.length > 0 && (
        <div style={{
          background: 'var(--semi-color-bg-1)',
          borderRadius: 8,
          padding: 16,
          marginBottom: 16,
          border: '1px solid var(--semi-color-border)',
        }}>
          <Text strong style={{ fontSize: 14 }}>每日消耗总览</Text>
          <div style={{ display: 'flex', gap: 24, marginTop: 12, flexWrap: 'wrap' }}>
            <div>
              <Text type="tertiary" size="small">今日消耗</Text>
              <div>
                <Text
                  strong
                  style={{ fontSize: 24, color: todayRow && todayRow.total_cost > 0 ? 'var(--semi-color-danger)' : undefined }}
                >
                  ${(todayRow?.total_cost ?? 0).toFixed(4)}
                </Text>
              </div>
              <Text type="tertiary" size="small">{todayRow?.request_count ?? 0} 次请求</Text>
            </div>
            {dailyCosts.slice(0, 7).map((d) => (
              <div key={d.date} style={{ opacity: d.date === todayStr ? 1 : 0.7 }}>
                <Text type="tertiary" size="small">{d.date.slice(5)}</Text>
                <div>
                  <Text strong style={{ fontSize: 16 }}>${d.total_cost.toFixed(2)}</Text>
                </div>
                <Text type="tertiary" size="small">{d.request_count} 次</Text>
              </div>
            ))}
          </div>
        </div>
      )}

      <Table
        columns={columns}
        dataSource={logs}
        loading={loading}
        rowKey="id"
        pagination={{
          total,
          pageSize,
          currentPage: page,
          onChange: (p) => setPage(p),
          showTotal: true,
        }}
      />
    </div>
  )
}
