import React, { useEffect, useState, useCallback, useRef } from 'react'
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
  Spin,
  Input,
} from '@douyinfe/semi-ui'
import { IconPlus, IconEdit, IconDelete, IconCheckCircleStroked } from '@douyinfe/semi-icons'
import { listChannels, createChannel, updateChannel, deleteChannel, testChannel } from '../api'
import type { Channel } from '../types'

const { Title, Text } = Typography

export default function Channels() {
  const [channels, setChannels] = useState<Channel[]>([])
  const [loading, setLoading] = useState(true)
  const [modalVisible, setModalVisible] = useState(false)
  const [editing, setEditing] = useState<Channel | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [testResult, setTestResult] = useState<any>(null)
  const [testModalVisible, setTestModalVisible] = useState(false)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [newModel, setNewModel] = useState('')
  const formApiRef = useRef<any>(null)

  const load = useCallback(() => {
    setLoading(true)
    listChannels()
      .then((res) => setChannels(res.data.data ?? []))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { load() }, [load])

  const handleCreate = () => { setEditing(null); setModalVisible(true) }
  const handleEdit = (ch: Channel) => { setEditing(ch); setModalVisible(true) }

  const handleDelete = async (id: number) => {
    await deleteChannel(id)
    Toast.success('已删除')
    load()
  }

  const handleToggle = async (ch: Channel) => {
    await updateChannel(ch.id, { ...ch, status: ch.status === 1 ? 0 : 1 })
    Toast.success(ch.status === 1 ? '已禁用' : '已启用')
    load()
  }

  const handleTest = async (id: number) => {
    setTestingId(id)
    try {
      const res = await testChannel(id)
      setTestResult(res.data)
    } catch (err: any) {
      setTestResult(err.response?.data || { error: 'test failed' })
    }
    setTestingId(null)
    setTestModalVisible(true)
  }

  const handleAddModel = () => {
    const name = newModel.trim()
    if (!name) return
    const api = formApiRef.current
    if (!api) return
    const current = api.getValue('models') || ''
    api.setValue('models', current ? `${current},${name}` : name)
    setNewModel('')
  }

  const handleSubmit = async (values: any) => {
    setSubmitting(true)
    try {
      const payload = {
        name: values.name,
        type: values.type || 'openai',
        base_url: values.base_url,
        api_key: values.api_key,
        models: values.models || '',
        priority: Number(values.priority) || 0,
        status: 1,
      }
      if (editing) {
        await updateChannel(editing.id, payload)
        Toast.success('已更新')
      } else {
        await createChannel(payload)
        Toast.success('创建成功')
      }
      setModalVisible(false)
      load()
    } finally {
      setSubmitting(false)
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name', render: (v: string) => <strong>{v}</strong> },
    { title: '类型', dataIndex: 'type', render: (v: string) => <Tag>{v}</Tag> },
    {
      title: 'Base URL',
      dataIndex: 'base_url',
      render: (v: string) => (
        <span style={{ fontSize: 12, opacity: 0.7, wordBreak: 'break-all' }}>{v}</span>
      ),
    },
    {
      title: '支持模型',
      dataIndex: 'models',
      render: (v: string) =>
        v ? (
          <span style={{ fontSize: 11, opacity: 0.7 }}>{v.length > 40 ? v.slice(0, 40) + '…' : v}</span>
        ) : (
          <Tag size="small" color="green">全部</Tag>
        ),
    },
    { title: '优先级', dataIndex: 'priority' },
    {
      title: '状态',
      dataIndex: 'status',
      render: (s: number) => <Tag color={s === 1 ? 'green' : 'red'}>{s === 1 ? '启用' : '禁用'}</Tag>,
    },
    {
      title: '操作',
      render: (_: any, row: Channel) => (
        <Space>
          <Button size="small" icon={<IconEdit />} onClick={() => handleEdit(row)}>编辑</Button>
          <Button
            size="small"
            type="tertiary"
            loading={testingId === row.id}
            icon={<IconCheckCircleStroked />}
            onClick={() => handleTest(row.id)}
          >测试</Button>
          <Button
            size="small"
            theme="borderless"
            type={row.status === 1 ? 'warning' : 'primary'}
            onClick={() => handleToggle(row)}
          >
            {row.status === 1 ? '禁用' : '启用'}
          </Button>
          <Popconfirm title="确认删除该渠道？" onConfirm={() => handleDelete(row.id)}>
            <Button size="small" type="danger" icon={<IconDelete />} theme="borderless" />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>渠道管理</Title>
        <Button icon={<IconPlus />} type="primary" theme="solid" onClick={handleCreate}>
          添加渠道
        </Button>
      </div>

      <Table columns={columns} dataSource={channels} loading={loading} rowKey="id" pagination={{ pageSize: 15 }} />

      <Modal
        title={editing ? '编辑渠道' : '添加渠道'}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={560}
      >
        <Form
          onSubmit={handleSubmit}
          getFormApi={(api: any) => { formApiRef.current = api }}
          initValues={
            editing
              ? {
                  name: editing.name,
                  type: editing.type,
                  base_url: editing.base_url,
                  api_key: editing.api_key,
                  models: editing.models,
                  priority: editing.priority,
                }
              : { type: 'openai', priority: 0 }
          }
        >
          <Form.Input field="name" label="名称" placeholder="渠道名称" rules={[{ required: true }]} />
          <Form.Input field="type" label="类型" placeholder="openai / azure / custom" />
          <Form.Input
            field="base_url"
            label="Base URL"
            placeholder="https://api.openai.com"
            rules={[{ required: true }]}
          />
          <Form.Input
            field="api_key"
            label="API Key"
            mode="password"
            placeholder="sk-..."
            rules={[{ required: true }]}
          />
          <Form.Input
            field="models"
            label="支持的模型"
            placeholder="gpt-4o,gpt-3.5-turbo（留空表示全部）"
          />
          <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
            <Input
              placeholder="输入模型名称"
              value={newModel}
              onChange={(v) => setNewModel(v)}
              style={{ flex: 1 }}
            />
            <Button onClick={handleAddModel}>填入</Button>
          </div>
          <Form.InputNumber field="priority" label="优先级" placeholder="数字越大越优先" />

          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16 }}>
            <Button onClick={() => setModalVisible(false)}>取消</Button>
            <Button htmlType="submit" type="primary" theme="solid" loading={submitting}>
              {editing ? '保存' : '创建'}
            </Button>
          </div>
        </Form>
      </Modal>

      {/* Test Result Modal */}
      <Modal
        title="测试结果"
        visible={testModalVisible}
        onCancel={() => setTestModalVisible(false)}
        footer={<Button onClick={() => setTestModalVisible(false)}>关闭</Button>}
        width={640}
      >
        {testResult ? (
          <div>
            <Tag color={testResult.error ? 'red' : 'green'} size="large" style={{ marginBottom: 12 }}>
              {testResult.error ? '失败' : '成功'}
            </Tag>
            {testResult.elapsed_ms != null && (
              <Text type="tertiary" style={{ marginLeft: 8 }}>延迟: {testResult.elapsed_ms}ms</Text>
            )}
            <pre style={{
              background: 'var(--semi-color-bg-1)',
              padding: 12,
              borderRadius: 6,
              fontSize: 12,
              overflow: 'auto',
              maxHeight: 400,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
            }}>
              {JSON.stringify(testResult, null, 2)}
            </pre>
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: 24 }}><Spin /></div>
        )}
      </Modal>
    </div>
  )
}
