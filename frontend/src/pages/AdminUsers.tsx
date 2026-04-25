import React, { useEffect, useState, useCallback } from 'react'
import {
  Typography,
  Table,
  Tag,
  Button,
  Modal,
  Form,
  Toast,
  Space,
  Input,
} from '@douyinfe/semi-ui'
import { IconEdit } from '@douyinfe/semi-icons'
import request from '../api/request'

const { Title, Text } = Typography

interface AdminUser {
  id: number
  username: string
  email: string
  role: string
  balance: number
  status: number
  created_at: string
}

export default function AdminUsers() {
  const [users, setUsers] = useState<AdminUser[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [editing, setEditing] = useState<AdminUser | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const load = useCallback(() => {
    setLoading(true)
    request.get('/admin/user', { params: { page } })
      .then((res) => { setUsers(res.data.data ?? []); setTotal(res.data.total ?? 0) })
      .finally(() => setLoading(false))
  }, [page])

  useEffect(() => { load() }, [load])

  const handleSubmit = async (values: {
    role: string
    balance: string
    status: string
  }) => {
    if (!editing) return
    setSubmitting(true)
    try {
      await request.put(`/admin/user/${editing.id}`, {
        role: values.role,
        balance: parseFloat(values.balance),
        status: parseInt(values.status),
      })
      Toast.success('已更新')
      setEditing(null)
      load()
    } finally {
      setSubmitting(false)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: '用户名',
      dataIndex: 'username',
      render: (v: string) => <Text strong>{v}</Text>,
    },
    { title: '邮箱', dataIndex: 'email' },
    {
      title: '角色',
      dataIndex: 'role',
      render: (v: string) => (
        <Tag color={v === 'admin' ? 'purple' : 'blue'}>{v === 'admin' ? '管理员' : '用户'}</Tag>
      ),
    },
    {
      title: '余额',
      dataIndex: 'balance',
      render: (v: number) => `$${v.toFixed(4)}`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (s: number) => <Tag color={s === 1 ? 'green' : 'red'}>{s === 1 ? '正常' : '禁用'}</Tag>,
    },
    {
      title: '注册时间',
      dataIndex: 'created_at',
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: '操作',
      render: (_: any, row: AdminUser) => (
        <Button size="small" icon={<IconEdit />} onClick={() => setEditing(row)}>编辑</Button>
      ),
    },
  ]

  return (
    <div>
      <Title heading={4} style={{ marginBottom: 16 }}>用户管理</Title>

      <Table
        columns={columns}
        dataSource={users}
        loading={loading}
        rowKey="id"
        pagination={{ total, pageSize: 20, currentPage: page, onChange: setPage, showTotal: true }}
      />

      <Modal
        title={`编辑用户：${editing?.username}`}
        visible={!!editing}
        onCancel={() => setEditing(null)}
        footer={null}
      >
        {editing && (
          <Form
            onSubmit={handleSubmit}
            initValues={{
              role: editing.role,
              balance: String(editing.balance),
              status: String(editing.status),
            }}
          >
            <Form.Select field="role" label="角色" style={{ width: '100%' }}>
              <Form.Select.Option value="user">普通用户</Form.Select.Option>
              <Form.Select.Option value="admin">管理员</Form.Select.Option>
            </Form.Select>
            <Form.InputNumber field="balance" label="余额 (USD)" min={0} />
            <Form.Select field="status" label="状态" style={{ width: '100%' }}>
              <Form.Select.Option value="1">正常</Form.Select.Option>
              <Form.Select.Option value="0">禁用</Form.Select.Option>
            </Form.Select>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16 }}>
              <Button onClick={() => setEditing(null)}>取消</Button>
              <Button htmlType="submit" type="primary" theme="solid" loading={submitting}>保存</Button>
            </div>
          </Form>
        )}
      </Modal>
    </div>
  )
}
