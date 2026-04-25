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
  InputNumber,
} from '@douyinfe/semi-ui'
import { IconPlus, IconEdit, IconDelete } from '@douyinfe/semi-icons'
import { listNotices, createNotice, updateNotice, deleteNotice } from '../api'
import type { NoticeItem } from '../components/NoticeModal'

const { Title } = Typography

export default function AdminNotice() {
  const [notices, setNotices] = useState<NoticeItem[]>([])
  const [loading, setLoading] = useState(true)
  const [modalVisible, setModalVisible] = useState(false)
  const [editing, setEditing] = useState<NoticeItem | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const load = useCallback(() => {
    setLoading(true)
    listNotices()
      .then((res) => setNotices(res.data.data ?? []))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { load() }, [load])

  const handleCreate = () => { setEditing(null); setModalVisible(true) }
  const handleEdit = (n: NoticeItem) => { setEditing(n); setModalVisible(true) }

  const handleDelete = async (id: number) => {
    await deleteNotice(id)
    Toast.success('已删除')
    load()
  }

  const handleSubmit = async (values: any) => {
    setSubmitting(true)
    try {
      const payload = {
        title: values.title,
        content: values.content,
        priority: Number(values.priority) || 0,
        status: values.status ?? 1,
      }
      if (editing) {
        await updateNotice(editing.id, payload)
        Toast.success('已更新')
      } else {
        await createNotice(payload)
        Toast.success('创建成功')
      }
      setModalVisible(false)
      load()
    } finally {
      setSubmitting(false)
    }
  }

  const columns = [
    { title: '标题', dataIndex: 'title', render: (v: string) => <strong>{v}</strong> },
    {
      title: '内容预览',
      dataIndex: 'content',
      render: (v: string) => (
        <span style={{ fontSize: 12, opacity: 0.7 }}>
          {v.replace(/<[^>]+>/g, '').slice(0, 60)}{v.length > 60 ? '…' : ''}
        </span>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      render: (v: number) => <Tag color="blue">{v}</Tag>,
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
      render: (v: string) => (
        <span style={{ fontSize: 12 }}>{v ? new Date(v).toLocaleString() : '-'}</span>
      ),
    },
    {
      title: '操作',
      render: (_: any, row: NoticeItem) => (
        <Space>
          <Button size="small" icon={<IconEdit />} onClick={() => handleEdit(row)}>编辑</Button>
          <Popconfirm title="确认删除该公告？" onConfirm={() => handleDelete(row.id)}>
            <Button size="small" type="danger" icon={<IconDelete />} theme="borderless" />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>公告管理</Title>
        <Button icon={<IconPlus />} type="primary" theme="solid" onClick={handleCreate}>
          添加公告
        </Button>
      </div>

      <Table columns={columns} dataSource={notices} loading={loading} rowKey="id" pagination={{ pageSize: 15 }} />

      <Modal
        title={editing ? '编辑公告' : '添加公告'}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={640}
      >
        <Form
          onSubmit={handleSubmit}
          initValues={
            editing
              ? {
                  title: editing.title,
                  content: editing.content,
                  priority: editing.priority,
                  status: editing.status,
                }
              : { priority: 0, status: 1 }
          }
        >
          <Form.Input field="title" label="标题" placeholder="公告标题" rules={[{ required: true }]} />
          <Form.TextArea
            field="content"
            label="内容 (支持 HTML)"
            placeholder={'<p>公告内容</p>\n<a href="https://example.com">链接</a>'}
            autosize
            style={{ minHeight: 120 }}
            rules={[{ required: true }]}
          />
          <Form.InputNumber field="priority" label="优先级" placeholder="数字越大越优先" />
          <Form.Switch field="status" label="启用" />

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
