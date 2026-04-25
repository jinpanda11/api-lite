import React, { useEffect, useState, useCallback } from 'react'
import {
  Typography,
  Table,
  Input,
  Tag,
  Button,
  Modal,
  RadioGroup,
  Radio,
  Toast,
  Spin,
  Space,
  InputNumber,
} from '@douyinfe/semi-ui'
import { IconSearch, IconEdit } from '@douyinfe/semi-icons'
import { listModels, listModelPricing, updateModelPricing } from '../api'
import type { ModelInfo } from '../types'

const { Title, Text } = Typography

interface PricingConfig {
  id: number
  model_name: string
  billing_mode: 'token' | 'call'
  input_price: number
  output_price: number
  cache_read_price: number
  cache_create_price: number
  call_price: number
}

export default function ModelPricing() {
  const [models, setModels] = useState<ModelInfo[]>([])
  const [pricingMap, setPricingMap] = useState<Map<string, PricingConfig>>(new Map())
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [editModal, setEditModal] = useState(false)
  const [editingModel, setEditingModel] = useState<string | null>(null)
  const [editingPricing, setEditingPricing] = useState<PricingConfig | null>(null)
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [modelsRes, pricingRes] = await Promise.all([listModels(), listModelPricing()])
      const modelList: ModelInfo[] = modelsRes.data.data ?? []
      setModels(modelList)

      const map = new Map<string, PricingConfig>()
      const pricingList: PricingConfig[] = pricingRes.data.data ?? []
      for (const p of pricingList) {
        map.set(p.model_name, p)
      }
      setPricingMap(map)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  // Get unique model names from all channels
  const uniqueModels = React.useMemo(() => {
    const seen = new Set<string>()
    return models.filter((m) => {
      if (seen.has(m.id)) return false
      seen.add(m.id)
      return true
    })
  }, [models])

  const filtered = React.useMemo(() => {
    if (!search) return uniqueModels
    const kw = search.toLowerCase()
    return uniqueModels.filter((m) => m.id.toLowerCase().includes(kw))
  }, [uniqueModels, search])

  const handleEdit = (modelName: string) => {
    const existing = pricingMap.get(modelName)
    setEditingModel(modelName)
    setEditingPricing(
      existing
        ? { ...existing }
        : {
            id: 0,
            model_name: modelName,
            billing_mode: 'token',
            input_price: 0,
            output_price: 0,
            cache_read_price: 0,
            cache_create_price: 0,
            call_price: 0,
          }
    )
    setEditModal(true)
  }

  const handleSave = async () => {
    if (!editingModel || !editingPricing) return
    setSaving(true)
    try {
      await updateModelPricing(editingModel, editingPricing)
      Toast.success('定价保存成功')
      setEditModal(false)
      load()
    } catch {
      Toast.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  const columns = [
    {
      title: '模型名称',
      dataIndex: 'id',
      render: (id: string) => <Text code>{id}</Text>,
    },
    {
      title: '计费模式',
      render: (_: any, row: ModelInfo) => {
        const p = pricingMap.get(row.id)
        if (!p) return <Tag size="small" color="light-blue">未设置</Tag>
        return (
          <Tag color={p.billing_mode === 'call' ? 'purple' : 'blue'} size="small">
            {p.billing_mode === 'call' ? '按次计费' : '按量计费'}
          </Tag>
        )
      },
    },
    {
      title: '价格',
      render: (_: any, row: ModelInfo) => {
        const p = pricingMap.get(row.id)
        if (!p) return <Text type="tertiary">使用渠道默认价格</Text>
        if (p.billing_mode === 'call') {
          return <Text>${p.call_price.toFixed(6)} / 次</Text>
        }
        const parts: string[] = []
        if (p.input_price > 0) parts.push(`输入 $${p.input_price}`)
        if (p.output_price > 0) parts.push(`输出 $${p.output_price}`)
        if (parts.length === 0) return <Text type="tertiary">免费</Text>
        return <Text style={{ fontSize: 12 }}>{parts.join(', ')} / M</Text>
      },
    },
    {
      title: '操作',
      render: (_: any, row: ModelInfo) => (
        <Button size="small" icon={<IconEdit />} onClick={() => handleEdit(row.id)}>
          定价
        </Button>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title heading={4}>模型定价</Title>
        <Input
          prefix={<IconSearch />}
          placeholder="搜索模型名称"
          value={search}
          onChange={(v) => setSearch(v)}
          style={{ width: 260 }}
        />
      </div>

      <Table
        columns={columns}
        dataSource={filtered}
        loading={loading}
        rowKey="id"
        pagination={{ pageSize: 20 }}
        empty={
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Text type="tertiary">暂无模型数据，请先配置渠道</Text>
          </div>
        }
      />

      <Modal
        title={`定价设置 - ${editingModel}`}
        visible={editModal}
        onCancel={() => setEditModal(false)}
        footer={
          <Space>
            <Button onClick={() => setEditModal(false)}>取消</Button>
            <Button type="primary" theme="solid" loading={saving} onClick={handleSave}>
              保存
            </Button>
          </Space>
        }
        width={520}
      >
        {editingPricing ? (
          <div>
            <div style={{ marginBottom: 20 }}>
              <Text style={{ display: 'block', marginBottom: 8, fontWeight: 500, fontSize: 14 }}>计费方式</Text>
              <RadioGroup
                value={editingPricing.billing_mode}
                onChange={(e) =>
                  setEditingPricing({ ...editingPricing, billing_mode: e.target.value as 'token' | 'call' })
                }
              >
                <Radio value="token">按量计费</Radio>
                <Radio value="call">按次计费</Radio>
              </RadioGroup>
            </div>

            {editingPricing.billing_mode === 'token' ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                <div>
                  <Text style={{ display: 'block', marginBottom: 4, fontSize: 13 }}>输入价格 (USD / M tokens)</Text>
                  <InputNumber
                    value={editingPricing.input_price}
                    onChange={(v) =>
                      setEditingPricing({ ...editingPricing, input_price: Number(v) || 0 })
                    }
                    min={0}
                    step={0.001}
                    style={{ width: '100%' }}
                  />
                </div>
                <div>
                  <Text style={{ display: 'block', marginBottom: 4, fontSize: 13 }}>输出价格 (USD / M tokens)</Text>
                  <InputNumber
                    value={editingPricing.output_price}
                    onChange={(v) =>
                      setEditingPricing({ ...editingPricing, output_price: Number(v) || 0 })
                    }
                    min={0}
                    step={0.001}
                    style={{ width: '100%' }}
                  />
                </div>
                <div>
                  <Text style={{ display: 'block', marginBottom: 4, fontSize: 13 }}>缓存读取价格 (USD / M tokens)</Text>
                  <InputNumber
                    value={editingPricing.cache_read_price}
                    onChange={(v) =>
                      setEditingPricing({ ...editingPricing, cache_read_price: Number(v) || 0 })
                    }
                    min={0}
                    step={0.001}
                    style={{ width: '100%' }}
                  />
                </div>
                <div>
                  <Text style={{ display: 'block', marginBottom: 4, fontSize: 13 }}>缓存创建价格 (USD / M tokens)</Text>
                  <InputNumber
                    value={editingPricing.cache_create_price}
                    onChange={(v) =>
                      setEditingPricing({ ...editingPricing, cache_create_price: Number(v) || 0 })
                    }
                    min={0}
                    step={0.001}
                    style={{ width: '100%' }}
                  />
                </div>
              </div>
            ) : (
              <div>
                <Text style={{ display: 'block', marginBottom: 4, fontSize: 13 }}>每次调用价格 (USD / 次)</Text>
                <InputNumber
                  value={editingPricing.call_price}
                  onChange={(v) =>
                    setEditingPricing({ ...editingPricing, call_price: Number(v) || 0 })
                  }
                  min={0}
                  step={0.001}
                  style={{ width: '100%' }}
                />
              </div>
            )}
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: 24 }}><Spin /></div>
        )}
      </Modal>
    </div>
  )
}
