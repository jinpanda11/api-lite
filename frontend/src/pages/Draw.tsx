import React, { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Typography,
  Button,
  Select,
  TextArea,
  Input,
  Spin,
  Tag,
  Toast,
} from '@douyinfe/semi-ui'
import { IconImage, IconCopy, IconDownload, IconUser, IconPlus } from '@douyinfe/semi-icons'
import { generateImage, getDrawQuota, listModels, getNotices, getBranding } from '../api'
import { useAppStore } from '../store'
import type { NoticeItem } from '../components/NoticeModal'

const { Title, Text } = Typography

const SIZES = ['auto', '256x256', '512x512', '1024x1024', '1792x1024', '1024x1792']
const QUALITIES = ['standard', 'hd']

function AdSlot({ code }: { code: string }) {
  if (!code) return null
  return (
    <div style={{ marginBottom: 16, textAlign: 'center', overflow: 'hidden' }}>
      <div dangerouslySetInnerHTML={{ __html: code }} />
    </div>
  )
}

export default function Draw() {
  const navigate = useNavigate()
  const loggedIn = useAppStore((s) => s.loggedIn)
  const [models, setModels] = useState<string[]>([])
  const [selectedModel, setSelectedModel] = useState('')
  const [prompt, setPrompt] = useState('')
  const [size, setSize] = useState('auto')
  const [customW, setCustomW] = useState('1024')
  const [customH, setCustomH] = useState('1024')
  const [sizeMode, setSizeMode] = useState<'preset' | 'custom'>('preset')
  const [quality, setQuality] = useState('standard')
  const [generating, setGenerating] = useState(false)
  const [images, setImages] = useState<string[]>([])
  const [quota, setQuota] = useState({ remaining: 0, total: 10, isAdmin: false })
  const [error, setError] = useState('')
  const [debug, setDebug] = useState('')
  const [notices, setNotices] = useState<NoticeItem[]>([])
  const [showNotices, setShowNotices] = useState(true)
  const [adCode, setAdCode] = useState('')

  const loadQuota = useCallback(() => {
    getDrawQuota()
      .then((res) => {
        const d = res?.data
        if (d) setQuota({ remaining: d.quota_remaining, total: d.quota_total, isAdmin: d.is_admin })
      })
      .catch((err) => {
        console.error('quota error', err)
      })
  }, [])

  useEffect(() => {
    loadQuota()
    listModels()
      .then((res) => {
        const list = (res.data?.data || []) as any[]
        const names = list
          .filter((m: any) => m.channel_type === 'image')
          .map((m: any) => m.id || m.model_name || '')
          .filter((v: string) => v)
        setDebug((prev) => prev + '\nmodels: ' + JSON.stringify(names))
        if (names.length > 0) {
          setModels(names)
          setSelectedModel(names[0])
        }
      })
      .catch((err) => {
        setDebug((prev) => prev + '\nmodels error: ' + String(err))
      })
    getNotices()
      .then((res) => {
        const list: NoticeItem[] = res.data.data ?? []
        if (list.length > 0) setNotices(list)
      })
      .catch(() => {})
    getBranding()
      .then((res) => {
        setAdCode(res.data?.draw_ad_code || '')
      })
      .catch(() => {})
  }, [loadQuota])

  const handleGenerate = async () => {
    if (!loggedIn) {
      Toast.warning('请先登录')
      navigate('/login')
      return
    }
    setError('')
    setDebug('')
    if (!prompt.trim()) {
      Toast.warning('请输入画图提示词')
      return
    }
    if (!selectedModel) {
      Toast.warning('请选择模型')
      return
    }
    const effectiveSize = sizeMode === 'custom' ? `${customW}x${customH}` : size

    setGenerating(true)
    setImages([])
    try {
      const res = await generateImage({
        model: selectedModel,
        prompt: prompt.trim(),
        size: effectiveSize,
        quality,
      })
      const body = res.data
      setDebug('response: ' + JSON.stringify(body))
      if (body?.data?.length > 0) {
        const urls = body.data.map((d: any) => d.url).filter(Boolean)
        setImages(urls)
      } else if (body?.error) {
        setError(body.error)
        Toast.error(body.error)
      } else {
        setDebug((prev) => prev + ' | no data array in response')
        Toast.warning('生成成功但未返回图片')
      }
      if (body?.quota_remaining >= 0) {
        setQuota((prev) => ({ ...prev, remaining: body.quota_remaining }))
      } else {
        loadQuota()
      }
    } catch (err: any) {
      const msg = err?.response?.data?.error || err?.message || '生成失败'
      setError(msg)
      setDebug((prev) => prev + '\nerror: ' + JSON.stringify({ msg, status: err?.response?.status, data: err?.response?.data }))
      Toast.error(msg)
    } finally {
      setGenerating(false)
    }
  }

  const handleCopyUrl = (url: string) => {
    navigator.clipboard.writeText(url).then(
      () => Toast.success('链接已复制'),
      () => Toast.error('复制失败')
    )
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.ctrlKey && e.key === 'Enter') {
      handleGenerate()
    }
  }

  if (!loggedIn) {
    return (
      <div style={{ maxWidth: 600, margin: '80px auto', textAlign: 'center' }}>
        <div style={{ fontSize: 64, marginBottom: 24 }}>🎨</div>
        <Title heading={2} style={{ marginBottom: 12 }}>AI 画图</Title>
        <Text type="tertiary" style={{ display: 'block', marginBottom: 32, fontSize: 16 }}>
          免费 AI 图片生成，每天 {10} 次，登录即用
        </Text>
        {notices.length > 0 && showNotices && (
          <Card
            title="📢 公告"
            style={{ marginBottom: 24, textAlign: 'left', borderLeft: '3px solid var(--semi-color-primary)' }}
            headerExtraContent={
              <Button size="small" theme="borderless" onClick={() => setShowNotices(false)}>收起</Button>
            }
          >
            {notices.map((n, i) => (
              <div key={n.id} style={{ marginBottom: i < notices.length - 1 ? 16 : 0, paddingBottom: i < notices.length - 1 ? 16 : 0, borderBottom: i < notices.length - 1 ? '1px solid var(--semi-color-border)' : 'none' }}>
                <Text strong style={{ display: 'block', marginBottom: 4 }}>{n.title}</Text>
                <Text type="tertiary" style={{ lineHeight: 1.8, whiteSpace: 'pre-wrap' }}>{n.content}</Text>
              </div>
            ))}
          </Card>
        )}
        <AdSlot code={adCode} />
        <div style={{ display: 'flex', gap: 12, justifyContent: 'center' }}>
          <Button type="primary" theme="solid" size="large" icon={<IconUser />} onClick={() => navigate('/login')}>
            登录
          </Button>
          <Button theme="solid" size="large" icon={<IconPlus />} onClick={() => navigate('/register')}>
            注册
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 800, margin: '0 auto' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title heading={4}>AI 画图</Title>
        {quota.isAdmin ? (
          <Tag color="blue" size="large">管理员 · 无限次数</Tag>
        ) : (
          <Tag color={quota.remaining > 0 ? 'green' : 'red'} size="large">
            今日剩余: {quota.remaining} / {quota.total}
          </Tag>
        )}
      </div>

      {notices.length > 0 && showNotices && (
        <Card
          title="📢 公告"
          style={{ marginBottom: 16, borderLeft: '3px solid var(--semi-color-primary)' }}
          headerExtraContent={
            <Button size="small" theme="borderless" onClick={() => setShowNotices(false)}>收起</Button>
          }
        >
          {notices.map((n, i) => (
            <div key={n.id} style={{ marginBottom: i < notices.length - 1 ? 16 : 0, paddingBottom: i < notices.length - 1 ? 16 : 0, borderBottom: i < notices.length - 1 ? '1px solid var(--semi-color-border)' : 'none' }}>
              <Text strong style={{ display: 'block', marginBottom: 4 }}>{n.title}</Text>
              <Text type="tertiary" style={{ lineHeight: 1.8, whiteSpace: 'pre-wrap' }}>{n.content}</Text>
            </div>
          ))}
        </Card>
      )}

      <AdSlot code={adCode} />

      <Card style={{ marginBottom: 16 }}>
        <div style={{ marginBottom: 16 }}>
          <Text strong style={{ display: 'block', marginBottom: 8 }}>模型</Text>
          <Select
            value={selectedModel}
            onChange={(v) => setSelectedModel(v as string)}
            style={{ width: '100%' }}
            placeholder="选择画图模型"
          >
            {models.map((m) => (
              <Select.Option key={m} value={m}>{m}</Select.Option>
            ))}
          </Select>
        </div>

        <div style={{ marginBottom: 16 }}>
          <Text strong style={{ display: 'block', marginBottom: 8 }}>提示词</Text>
          <TextArea
            value={prompt}
            onChange={(v) => setPrompt(v)}
            placeholder="描述你想要生成的图片..."
            maxCount={4000}
            rows={3}
            onKeyDown={handleKeyDown}
          />
        </div>

        <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
          <div style={{ flex: 1 }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>尺寸</Text>
            <Select
              value={sizeMode === 'preset' ? size : '__custom__'}
              onChange={(v) => {
                if (v === '__custom__') {
                  setSizeMode('custom')
                } else {
                  setSizeMode('preset')
                  setSize(v as string)
                }
              }}
              style={{ width: '100%' }}
            >
              {SIZES.map((s) => (<Select.Option key={s} value={s}>{s}</Select.Option>))}
              <Select.Option value="__custom__">自定义...</Select.Option>
            </Select>
            {sizeMode === 'custom' && (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8 }}>
                <Input value={customW} onChange={(v) => setCustomW(v)} placeholder="宽" style={{ flex: 1, textAlign: 'center' }} />
                <Text style={{ fontWeight: 'bold', flexShrink: 0 }}>x</Text>
                <Input value={customH} onChange={(v) => setCustomH(v)} placeholder="高" style={{ flex: 1, textAlign: 'center' }} />
              </div>
            )}
          </div>
          <div style={{ flex: 1 }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>质量</Text>
            <Select value={quality} onChange={(v) => setQuality(v as string)} style={{ width: '100%' }}>
              {QUALITIES.map((q) => (<Select.Option key={q} value={q}>{q}</Select.Option>))}
            </Select>
          </div>
        </div>

        <Button
          type="primary"
          theme="solid"
          block
          loading={generating}
          onClick={handleGenerate}
          icon={<IconImage />}
          size="large"
        >
          生成图片 (Ctrl+Enter)
        </Button>
      </Card>

      {generating && (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
          <Text type="tertiary" style={{ display: 'block', marginTop: 16 }}>AI 正在生成图片，通常需要 10-60 秒...</Text>
        </div>
      )}

      {error && (
        <Card style={{ marginBottom: 16, borderColor: 'var(--semi-color-danger)' }}>
          <Text type="danger" strong>错误：{error}</Text>
        </Card>
      )}

      {debug && quota.isAdmin && (
        <Card title="调试信息" style={{ marginBottom: 16, fontSize: 11, opacity: 0.7 }}>
          <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', margin: 0 }}>{debug}</pre>
        </Card>
      )}

      {images.length > 0 && (
        <Card title="生成结果">
          <div style={{ display: 'grid', gap: 16 }}>
            {images.map((url, i) => (
              <div key={i}>
                <img
                  src={url}
                  alt={`Generated ${i + 1}`}
                  style={{ width: '100%', borderRadius: 8, border: '1px solid var(--semi-color-border)' }}
                  onLoad={() => setDebug((prev) => prev + '\nimage ' + i + ' loaded')}
                  onError={() => setDebug((prev) => prev + '\nimage ' + i + ' load FAILED')}
                />
                <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                  <Button size="small" icon={<IconCopy />} onClick={() => handleCopyUrl(url)}>
                    复制链接
                  </Button>
                  <Button size="small" icon={<IconDownload />} onClick={() => window.open(url, '_blank')}>
                    新窗口打开
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      <AdSlot code={adCode} />
    </div>
  )
}
