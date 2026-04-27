import React, { useEffect, useState, useCallback } from 'react'
import {
  Typography,
  Button,
  Table,
  Modal,
  Form,
  Tag,
  Toast,
  Popconfirm,
  Space,
} from '@douyinfe/semi-ui'
import { IconPlus, IconDelete, IconDownload } from '@douyinfe/semi-icons'
import request from '../api/request'

const { Title, Text } = Typography

interface RedeemCode {
  id: number
  code: string
  value: number
  status: number
  used_by?: number
  used_at?: string
  created_at: string
}

export default function AdminRedeem() {
  const [codes, setCodes] = useState<RedeemCode[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [modalVisible, setModalVisible] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [page, setPage] = useState(1)
  const [resultCodes, setResultCodes] = useState<{ codes: string[]; value: number } | null>(null)
  const [clearingUsed, setClearingUsed] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([])

  const load = useCallback(() => {
    setLoading(true)
    request.get('/admin/redeem', { params: { page } })
      .then((res) => { setCodes(res.data.data ?? []); setTotal(res.data.total ?? 0) })
      .finally(() => setLoading(false))
  }, [page])

  useEffect(() => { load() }, [load])

  const handleDelete = async (id: number) => {
    await request.delete(`/admin/redeem/${id}`)
    Toast.success('已删除')
    load()
  }

  const handleClearUsed = async () => {
    setClearingUsed(true)
    try {
      const res = await request.delete('/admin/redeem/used')
      Toast.success(`已删除 ${res.data.count} 个已使用的兑换码`)
      load()
    } finally {
      setClearingUsed(false)
    }
  }

  const handleBatchDelete = async () => {
    await Promise.all(selectedRowKeys.map((id) => request.delete(`/admin/redeem/${id}`)))
    Toast.success(`已删除 ${selectedRowKeys.length} 个兑换码`)
    setSelectedRowKeys([])
    load()
  }

  const downloadTxt = (codes: string[], value: number) => {
    const content = codes.join('\n')
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${value}.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  const handleCreate = async (values: { count: string; value: string }) => {
    setSubmitting(true)
    try {
      const valueNum = parseFloat(values.value)
      const res = await request.post('/admin/redeem', {
        count: parseInt(values.count),
        value: valueNum,
      })
      const created: string[] = (res.data.data ?? []).map((c: RedeemCode) => c.code)
      setResultCodes({ codes: created, value: valueNum })
      Toast.success(`已创建 ${created.length} 个兑换码`)
      load()
    } finally {
      setSubmitting(false)
    }
  }

  const columns = [
    {
      title: '兑换码',
      dataIndex: 'code',
      render: (v: string) => <Text code style={{ letterSpacing: 1 }}>{v}</Text>,
    },
    {
      title: '面值',
      dataIndex: 'value',
      render: (v: number) => <Text strong>${v.toFixed(2)}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (s: number, row: RedeemCode) => (
        <Tag color={s === 1 ? 'green' : 'grey'}>
          {s === 1 ? '未使用' : `已使用 (UID: ${row.used_by})`}
        </Tag>
      ),
    },
    {
      title: '使用时间',
      dataIndex: 'used_at',
      render: (v?: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: '操作',
      render: (_: any, row: RedeemCode) => (
        <Popconfirm title="确认删除该兑换码？" onConfirm={() => handleDelete(row.id)}>
          <Button size="small" type="danger" icon={<IconDelete />} theme="borderless" />
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>兑换码管理</Title>
        <Space>
          {selectedRowKeys.length > 0 && (
            <Popconfirm
              title={`确认删除选中的 ${selectedRowKeys.length} 个兑换码？`}
              onConfirm={handleBatchDelete}
            >
              <Button icon={<IconDelete />} type="danger" theme="solid">
                删除选中 ({selectedRowKeys.length})
              </Button>
            </Popconfirm>
          )}
          <Popconfirm title="确认删除所有已使用的兑换码？此操作不可恢复。" onConfirm={handleClearUsed}>
            <Button icon={<IconDelete />} type="danger" theme="light" loading={clearingUsed}>
              清除已使用
            </Button>
          </Popconfirm>
          <Button icon={<IconPlus />} type="primary" theme="solid" onClick={() => setModalVisible(true)}>
            批量生成
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={codes}
        loading={loading}
        rowKey="id"
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as number[]),
        }}
        pagination={{ total, pageSize: 20, currentPage: page, onChange: setPage, showTotal: true }}
      />

      <Modal
        title={resultCodes ? '生成完成' : '批量生成兑换码'}
        visible={modalVisible}
        onCancel={() => { setModalVisible(false); setResultCodes(null) }}
        footer={null}
      >
        {resultCodes ? (
          <div>
            <Text>
              已生成 <strong>{resultCodes.codes.length}</strong> 个面值 <strong>${resultCodes.value.toFixed(2)}</strong> 的兑换码
            </Text>
            <pre style={{
              background: 'var(--semi-color-bg-1)',
              padding: 12,
              borderRadius: 6,
              fontSize: 12,
              maxHeight: 300,
              overflow: 'auto',
              marginTop: 12,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
            }}>
              {resultCodes.codes.join('\n')}
            </pre>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16 }}>
              <Button onClick={() => { setModalVisible(false); setResultCodes(null) }}>关闭</Button>
              <Button
                icon={<IconDownload />}
                type="primary"
                theme="solid"
                onClick={() => downloadTxt(resultCodes.codes, resultCodes.value)}
              >
                下载 TXT
              </Button>
            </div>
          </div>
        ) : (
          <Form onSubmit={handleCreate} initValues={{ count: '5', value: '5.00' }}>
            <Form.InputNumber field="count" label="生成数量" min={1} max={100} rules={[{ required: true }]} />
            <Form.InputNumber field="value" label="面值（USD）" min={0.01} rules={[{ required: true }]} />
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16 }}>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
              <Button htmlType="submit" type="primary" theme="solid" loading={submitting}>生成</Button>
            </div>
          </Form>
        )}
      </Modal>
    </div>
  )
}
