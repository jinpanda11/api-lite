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
  DatePicker,
} from '@douyinfe/semi-ui'
import { IconPlus, IconCopy, IconDelete, IconEdit } from '@douyinfe/semi-icons'
import { listTokens, createToken, updateToken, deleteToken } from '../api'
import type { Token } from '../types'

const { Title, Text } = Typography

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).then(() => Toast.success('已复制到剪贴板'))
}

export default function Tokens() {
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)
  const [modalVisible, setModalVisible] = useState(false)
  const [editing, setEditing] = useState<Token | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const load = useCallback(() => {
    setLoading(true)
    listTokens()
      .then((res) => setTokens(res.data.data))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { load() }, [load])

  const handleCreate = () => {
    setEditing(null)
    setModalVisible(true)
  }

  const handleEdit = (token: Token) => {
    setEditing(token)
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    await deleteToken(id)
    Toast.success('已删除')
    load()
  }

  const handleToggleStatus = async (token: Token) => {
    const newStatus = token.status === 1 ? 0 : 1
    await updateToken(token.id, { status: newStatus })
    Toast.success(newStatus === 1 ? '已启用' : '已禁用')
    load()
  }

  const handleSubmit = async (values: any) => {
    setSubmitting(true)
    try {
      const payload = {
        name: values.name,
        remark: values.remark || '',
        expired_at: values.expired_at
          ? new Date(values.expired_at).toISOString()
          : null,
      }
      if (editing) {
        await updateToken(editing.id, payload)
        Toast.success('已更新')
      } else {
        await createToken(payload)
        Toast.success('创建成功')
      }
      setModalVisible(false)
      load()
    } finally {
      setSubmitting(false)
    }
  }

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      render: (name: string, row: Token) => (
        <div>
          <Text strong>{name}</Text>
          {row.remark && <Text type="tertiary" size="small" style={{ display: 'block' }}>{row.remark}</Text>}
        </div>
      ),
    },
    {
      title: 'Key',
      dataIndex: 'key',
      render: (key: string) => (
        <Space>
          <Text code style={{ fontSize: 12 }}>
            {key.slice(0, 12)}...
          </Text>
          <Button
            icon={<IconCopy />}
            size="small"
            theme="borderless"
            onClick={() => copyToClipboard(key)}
          />
        </Space>
      ),
    },
    {
      title: '过期时间',
      dataIndex: 'expired_at',
      render: (v: string | null) =>
        v ? new Date(v).toLocaleDateString() : <Text type="tertiary">永不过期</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (s: number) => (
        <Tag color={s === 1 ? 'green' : 'red'}>{s === 1 ? '启用' : '禁用'}</Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: '操作',
      render: (_: any, row: Token) => (
        <Space>
          <Button size="small" icon={<IconEdit />} onClick={() => handleEdit(row)}>编辑</Button>
          <Button
            size="small"
            theme="borderless"
            type={row.status === 1 ? 'warning' : 'primary'}
            onClick={() => handleToggleStatus(row)}
          >
            {row.status === 1 ? '禁用' : '启用'}
          </Button>
          <Popconfirm title="确认删除该令牌？" onConfirm={() => handleDelete(row.id)}>
            <Button size="small" type="danger" icon={<IconDelete />} theme="borderless" />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>我的令牌</Title>
        <Button icon={<IconPlus />} type="primary" theme="solid" onClick={handleCreate}>
          新建令牌
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={tokens}
        loading={loading}
        rowKey="id"
        pagination={{ pageSize: 15 }}
      />

      <Modal
        title={editing ? '编辑令牌' : '新建令牌'}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
      >
        <Form
          onSubmit={handleSubmit}
          initValues={
            editing
              ? { name: editing.name, remark: editing.remark }
              : {}
          }
        >
          <Form.Input
            field="name"
            label="名称"
            placeholder="令牌名称"
            rules={[{ required: true, message: '请输入名称' }]}
          />
          <Form.Input field="remark" label="备注" placeholder="可选备注" />
          <Form.DatePicker
            field="expired_at"
            label="过期时间"
            placeholder="留空表示永不过期"
            style={{ width: '100%' }}
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16 }}>
            <Button onClick={() => setModalVisible(false)}>取消</Button>
            <Button htmlType="submit" type="primary" theme="solid" loading={submitting}>
              {editing ? '保存' : '创建'}
            </Button>
          </div>
        </Form>
      </Modal>
    </div>
  )
}
