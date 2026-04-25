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
import { IconPlus, IconDelete } from '@douyinfe/semi-icons'
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

  const handleCreate = async (values: { count: string; value: string }) => {
    setSubmitting(true)
    try {
      const res = await request.post('/admin/redeem', {
        count: parseInt(values.count),
        value: parseFloat(values.value),
      })
      Toast.success(`已创建 ${res.data.data?.length} 个兑换码`)
      setModalVisible(false)
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
        <Button icon={<IconPlus />} type="primary" theme="solid" onClick={() => setModalVisible(true)}>
          批量生成
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={codes}
        loading={loading}
        rowKey="id"
        pagination={{ total, pageSize: 20, currentPage: page, onChange: setPage, showTotal: true }}
      />

      <Modal title="批量生成兑换码" visible={modalVisible} onCancel={() => setModalVisible(false)} footer={null}>
        <Form onSubmit={handleCreate} initValues={{ count: '5', value: '5.00' }}>
          <Form.InputNumber
            field="count"
            label="生成数量"
            min={1}
            max={100}
            rules={[{ required: true }]}
          />
          <Form.InputNumber
            field="value"
            label="面值（USD）"
            min={0.01}
            rules={[{ required: true }]}
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16 }}>
            <Button onClick={() => setModalVisible(false)}>取消</Button>
            <Button htmlType="submit" type="primary" theme="solid" loading={submitting}>生成</Button>
          </div>
        </Form>
      </Modal>
    </div>
  )
}
