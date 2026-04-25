import React, { useEffect, useState, useCallback } from 'react'
import { Typography, Table, Select, Button, Space, Tag, DatePicker } from '@douyinfe/semi-ui'
import { IconRefresh } from '@douyinfe/semi-icons'
import { getLogs } from '../api'
import type { Log } from '../types'

const { Title, Text } = Typography

export default function Logs() {
  const [logs, setLogs] = useState<Log[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [modelFilter, setModelFilter] = useState('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)

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
          <Button icon={<IconRefresh />} onClick={load}>刷新</Button>
        </Space>
      </div>

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
