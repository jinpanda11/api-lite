import React, { useEffect, useState } from 'react'
import { Typography, Table, Input, Tag } from '@douyinfe/semi-ui'
import { IconSearch } from '@douyinfe/semi-icons'
import { listModels } from '../api'
import type { ModelInfo } from '../types'

const { Title, Text } = Typography

export default function Models() {
  const [models, setModels] = useState<ModelInfo[]>([])
  const [filtered, setFiltered] = useState<ModelInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')

  useEffect(() => {
    listModels()
      .then((res) => {
        setModels(res.data.data ?? [])
        setFiltered(res.data.data ?? [])
      })
      .finally(() => setLoading(false))
  }, [])

  const handleSearch = (val: string) => {
    setSearch(val)
    const kw = val.toLowerCase()
    setFiltered(models.filter((m) => m.id.toLowerCase().includes(kw)))
  }

  const columns = [
    {
      title: '图标',
      render: (_: any, row: ModelInfo) => {
        const url = row.icon_url
        return url ? (
          <img
            src={url}
            alt={row.id}
            style={{ width: 28, height: 28, borderRadius: 6, objectFit: 'cover' }}
            onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
          />
        ) : (
          <div style={{
            width: 28, height: 28, borderRadius: 6, background: 'var(--semi-color-fill-0)',
            display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, color: 'var(--semi-color-text-2)',
          }}>
            {row.id.charAt(0).toUpperCase()}
          </div>
        )
      },
      width: 60,
    },
    {
      title: '模型 ID',
      dataIndex: 'id',
      render: (id: string) => <Text code>{id}</Text>,
    },
    {
      title: '渠道',
      dataIndex: 'channel_name',
      render: (name: string) => <Tag color="blue">{name}</Tag>,
    },
    {
      title: '计费方式',
      render: (_: any, row: ModelInfo) => {
        if (row.billing_mode === 'call') return <Tag color="purple">按次计费</Tag>
        if (row.billing_mode === 'token') return <Tag color="blue">按量计费</Tag>
        return <Tag size="small" color="light-blue">按量</Tag>
      },
    },
    {
      title: '输入价格 (/ M tokens)',
      render: (_: any, row: ModelInfo) => {
        if (row.billing_mode === 'call') return <Text type="tertiary">-</Text>
        return row.input_price === 0
          ? <Text type="tertiary">免费</Text>
          : <Text>${row.input_price.toFixed(6)}</Text>
      },
    },
    {
      title: '输出价格 (/ M tokens)',
      render: (_: any, row: ModelInfo) => {
        if (row.billing_mode === 'call') return <Text type="tertiary">{row.call_price ? `$${row.call_price.toFixed(6)} / 次` : '-'}</Text>
        return row.output_price === 0
          ? <Text type="tertiary">免费</Text>
          : <Text>${row.output_price.toFixed(6)}</Text>
      },
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>模型市场</Title>
        <Input
          prefix={<IconSearch />}
          placeholder="搜索模型名称"
          value={search}
          onChange={handleSearch}
          style={{ width: 260 }}
        />
      </div>

      <Table
        columns={columns}
        dataSource={filtered}
        loading={loading}
        rowKey={(row) => (row?.id ?? '') + (row?.channel_name ?? '')}
        pagination={{ pageSize: 20 }}
        empty={
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Text type="tertiary">暂无可用模型，请联系管理员配置渠道</Text>
          </div>
        }
      />
    </div>
  )
}
