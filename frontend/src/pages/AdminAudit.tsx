import React, { useEffect, useState, useCallback } from 'react'
import {
  Typography,
  Table,
  Tag,
  Input,
} from '@douyinfe/semi-ui'
import { IconSearch } from '@douyinfe/semi-icons'
import { getAuditLogs } from '../api'
import type { AuditLog } from '../types'

const { Title } = Typography

const ACTION_LABELS: Record<string, string> = {
  update_user_status: '修改用户状态',
  update_user: '修改用户',
  create_channel: '创建渠道',
  update_channel: '修改渠道',
  delete_channel: '删除渠道',
  test_channel: '测试渠道',
  toggle_monitor: '切换监控',
  update_monitor_interval: '修改监控间隔',
  create_notice: '创建公告',
  update_notice: '修改公告',
  delete_notice: '删除公告',
  update_settings: '修改设置',
  update_model_pricing: '修改模型定价',
  create_redeem: '创建兑换码',
  import_redeem: '导入兑换码',
}

export default function AdminAudit() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [actionFilter, setActionFilter] = useState('')
  const pageSize = 20

  const load = useCallback(() => {
    setLoading(true)
    const params: Record<string, string | number> = { page, page_size: pageSize }
    if (actionFilter) params.action = actionFilter
    getAuditLogs(params)
      .then((res) => {
        setLogs(res.data.data ?? [])
        setTotal(res.data.total ?? 0)
      })
      .finally(() => setLoading(false))
  }, [page, actionFilter])

  useEffect(() => { load() }, [load])

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '操作人',
      dataIndex: 'admin_name',
      width: 120,
      render: (v: string) => <strong>{v}</strong>,
    },
    {
      title: '操作',
      dataIndex: 'action',
      width: 150,
      render: (v: string) => (
        <Tag color="blue">{ACTION_LABELS[v] || v}</Tag>
      ),
    },
    {
      title: '详情',
      dataIndex: 'detail',
      render: (v: string) => (
        <span style={{ fontSize: 12, opacity: 0.8 }}>{v}</span>
      ),
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 180,
      render: (v: string) => (
        <span style={{ fontSize: 12 }}>{v ? new Date(v).toLocaleString() : '-'}</span>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>审计记录</Title>
        <Input
          prefix={<IconSearch />}
          placeholder="按操作类型过滤..."
          style={{ width: 240 }}
          value={actionFilter}
          onChange={(v) => { setActionFilter(v); setPage(1) }}
          showClear
          onClear={() => { setActionFilter(''); setPage(1) }}
        />
      </div>

      <Table
        columns={columns}
        dataSource={logs}
        loading={loading}
        rowKey="id"
        pagination={{
          currentPage: page,
          pageSize,
          total,
          onPageChange: (p) => setPage(p),
        }}
      />
    </div>
  )
}
