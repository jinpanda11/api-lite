import React, { useEffect, useState } from 'react'
import { Card, Row, Col, Typography, Spin, Tag, Table, Modal, Button, Banner } from '@douyinfe/semi-ui'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title as ChartTitle,
  Tooltip,
  Filler,
} from 'chart.js'
import { Line } from 'react-chartjs-2'
import { getDashboard, getNotices, checkIn, getCheckInStatus } from '../api'
import type { DashboardData, DailyCount, CheckInStatus } from '../types'
import type { NoticeItem } from '../components/NoticeModal'
import { useAppStore } from '../store'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ChartTitle, Tooltip, Filler)

const { Title, Text } = Typography

interface StatCardProps {
  title: string
  value: string | number
  sub?: string
}

function StatCard({ title, value, sub }: StatCardProps) {
  return (
    <Card style={{ height: '100%' }}>
      <Text type="tertiary" size="small">{title}</Text>
      <Title heading={2} style={{ margin: '8px 0 4px' }}>{value}</Title>
      {sub && <Text type="tertiary" size="small">{sub}</Text>}
    </Card>
  )
}

export default function Dashboard() {
  const user = useAppStore((s) => s.user)
  const theme = useAppStore((s) => s.theme)
  const [data, setData] = useState<DashboardData | null>(null)
  const [loading, setLoading] = useState(true)
  const [notices, setNotices] = useState<NoticeItem[]>([])
  const [noticeDetail, setNoticeDetail] = useState<NoticeItem | null>(null)
  const isAdmin = user?.role === 'admin'
  const [checkInData, setCheckInData] = useState({ checked_in_today: true, streak: 0, today_reward: 0.01 })
  const [checkInLoading, setCheckInLoading] = useState(false)

  useEffect(() => {
    getCheckInStatus()
      .then((res) => setCheckInData(res.data))
      .catch(() => {})
  }, [])

  const handleCheckIn = () => {
    setCheckInLoading(true)
    checkIn()
      .then((res) => {
        setCheckInData({ checked_in_today: true, streak: res.data.streak, today_reward: res.data.reward })
      })
      .finally(() => setCheckInLoading(false))
  }

  useEffect(() => {
    getDashboard()
      .then((res) => setData(res.data))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    getNotices()
      .then((res) => setNotices((res.data.data ?? []) as NoticeItem[]))
      .catch(() => {})
  }, [])

  const sanitizeHTML = (html: string): string => {
    const allowedTags = new Set([
      'b', 'i', 'u', 'strong', 'em', 'a', 'p', 'br', 'ul', 'ol', 'li',
      'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'span', 'div', 'blockquote', 'code', 'pre', 'hr',
    ])
    const allowedAttrs = new Set(['href', 'target', 'rel', 'class', 'id'])
    const div = document.createElement('div')
    div.innerHTML = html
    const clean = (node: Node): void => {
      if (node.nodeType === 3) return
      if (node.nodeType !== 1) { node.parentNode?.removeChild(node); return }
      const el = node as HTMLElement
      const tag = el.tagName.toLowerCase()
      if (!allowedTags.has(tag)) {
        while (el.firstChild) el.parentNode!.insertBefore(el.firstChild, el)
        el.parentNode!.removeChild(el)
        return
      }
      for (let i = el.attributes.length - 1; i >= 0; i--) {
        const name = el.attributes[i].name.toLowerCase()
        if (!allowedAttrs.has(name)) { el.removeAttribute(name) }
        else if (name === 'href') {
          const val = el.getAttribute('href') || ''
          if (/^javascript:/i.test(val) || /^data:/i.test(val)) el.removeAttribute('href')
        }
      }
      if (tag === 'a') { el.setAttribute('rel', 'noopener noreferrer'); el.setAttribute('target', '_blank') }
      Array.from(el.childNodes).forEach(clean)
    }
    Array.from(div.childNodes).forEach(clean)
    return div.innerHTML
  }

  const trend = data?.trend ?? []
  const lineColor = '#6366f1'

  // 7-day chart
  const chartData = {
    labels: trend.map((d: DailyCount) => d.date),
    datasets: [
      {
        label: '请求次数',
        data: trend.map((d: DailyCount) => d.count),
        borderColor: lineColor,
        backgroundColor: lineColor + '20',
        fill: true,
        tension: 0.4,
        pointRadius: 3,
      },
    ],
  }

  const chartOptions = {
    responsive: true,
    plugins: { legend: { display: false } },
    scales: {
      x: {
        grid: { color: theme === 'dark' ? '#2d2d2d' : '#f0f0f0' },
        ticks: { color: theme === 'dark' ? '#8a8a9a' : '#666' },
      },
      y: {
        grid: { color: theme === 'dark' ? '#2d2d2d' : '#f0f0f0' },
        ticks: { color: theme === 'dark' ? '#8a8a9a' : '#666' },
        beginAtZero: true,
      },
    },
  }

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 80 }}>
        <Spin size="large" />
      </div>
    )
  }

  return (
    <div>
      <Title heading={4} style={{ marginBottom: 24 }}>仪表板</Title>

      {notices.length > 0 && (
        <div onClick={() => setNoticeDetail(notices[0])} style={{ cursor: 'pointer', marginBottom: 16 }}>
          <Banner
            type="info"
            title={notices[0].title}
            description={notices[0].content.replace(/<[^>]*>/g, '').slice(0, 80) + (notices[0].content.length > 80 ? '...' : '')}
          />
        </div>
      )}

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="账户余额" value={`$${user?.balance?.toFixed(4) ?? '0.0000'}`} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="今日请求"
            value={data?.stats?.today_requests ?? 0}
            sub={`共 ${data?.stats?.total_requests ?? 0} 次`}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="今日消费"
            value={`$${(data?.stats?.today_cost ?? 0).toFixed(6)}`}
            sub={`总消费 $${(data?.stats?.total_cost ?? 0).toFixed(6)}`}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="活跃令牌" value={data?.token_count ?? 0} />
        </Col>
      </Row>

      {/* Admin system stats */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ height: '100%' }}>
            <Text type="tertiary" size="small">每日签到</Text>
            <div style={{ marginTop: 8 }}>
              {checkInData.checked_in_today ? (
                <Tag color="green">已签到 · 连续 {checkInData.streak} 天</Tag>
              ) : (
                <Button
                  type="primary"
                  size="small"
                  loading={checkInLoading}
                  onClick={handleCheckIn}
                >
                  签到 +${checkInData.today_reward.toFixed(2)}
                </Button>
              )}
            </div>
          </Card>
        </Col>
      </Row>

      {isAdmin && (
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} sm={12} lg={6}>
            <StatCard title="用户总数" value={data?.total_users ?? 0} />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard title="活跃渠道" value={data?.active_channels ?? 0} />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="系统今日请求"
              value={data?.sys_stats?.today_requests ?? 0}
              sub={`共 ${data?.sys_stats?.total_requests ?? 0} 次`}
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="系统总消费"
              value={`$${(data?.sys_stats?.total_cost ?? 0).toFixed(4)}`}
              sub={`今日 $${(data?.sys_stats?.today_cost ?? 0).toFixed(4)}`}
            />
          </Col>
        </Row>
      )}

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={isAdmin ? 16 : 24}>
          <Card>
            <Text strong style={{ display: 'block', marginBottom: 16 }}>近 7 天请求趋势</Text>
            <Line data={chartData} options={chartOptions as any} height={80} />
          </Card>
        </Col>
        {isAdmin && (data?.top_models?.length ?? 0) > 0 && (
          <Col xs={24} lg={8}>
            <Card style={{ height: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 16 }}>热门模型</Text>
              <Table
                dataSource={data?.top_models ?? []}
                rowKey="model"
                columns={[
                  { title: '模型', dataIndex: 'model', render: (v: string) => <Tag>{v}</Tag> },
                  { title: '次数', dataIndex: 'count', align: 'right' as const },
                ]}
                pagination={false}
                size="small"
              />
            </Card>
          </Col>
        )}
      </Row>

      <Modal
        title={noticeDetail?.title || '公告'}
        visible={noticeDetail !== null}
        onCancel={() => setNoticeDetail(null)}
        footer={<Button type="primary" onClick={() => setNoticeDetail(null)}>关闭</Button>}
        width={560}
      >
        {noticeDetail && (
          <div
            style={{ lineHeight: 1.8, fontSize: 14, maxHeight: 400, overflowY: 'auto' }}
            dangerouslySetInnerHTML={{ __html: sanitizeHTML(noticeDetail.content) }}
          />
        )}
      </Modal>
    </div>
  )
}
