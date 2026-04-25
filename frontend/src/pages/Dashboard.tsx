import React, { useEffect, useState } from 'react'
import { Card, Row, Col, Typography, Spin } from '@douyinfe/semi-ui'
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
import { getDashboard } from '../api'
import type { DashboardStats, DailyCount } from '../types'
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
      <Text type="tertiary" size="small">
        {title}
      </Text>
      <Title heading={2} style={{ margin: '8px 0 4px' }}>
        {value}
      </Title>
      {sub && <Text type="tertiary" size="small">{sub}</Text>}
    </Card>
  )
}

export default function Dashboard() {
  const user = useAppStore((s) => s.user)
  const theme = useAppStore((s) => s.theme)
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [trend, setTrend] = useState<DailyCount[]>([])
  const [tokenCount, setTokenCount] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getDashboard()
      .then((res) => {
        setStats(res.data.stats)
        setTrend(res.data.trend)
        setTokenCount(res.data.token_count)
      })
      .finally(() => setLoading(false))
  }, [])

  const lineColor = '#6366f1'
  const chartData = {
    labels: trend.map((d) => d.date),
    datasets: [
      {
        label: '请求次数',
        data: trend.map((d) => d.count),
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
      <Title heading={4} style={{ marginBottom: 24 }}>
        仪表板
      </Title>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="账户余额" value={`$${user?.balance?.toFixed(4) ?? '0.0000'}`} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="今日请求"
            value={stats?.today_requests ?? 0}
            sub={`共 ${stats?.total_requests ?? 0} 次`}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="今日消费"
            value={`$${(stats?.today_cost ?? 0).toFixed(6)}`}
            sub={`总消费 $${(stats?.total_cost ?? 0).toFixed(6)}`}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="活跃令牌" value={tokenCount} />
        </Col>
      </Row>

      <Card style={{ marginTop: 16 }}>
        <Text strong style={{ display: 'block', marginBottom: 16 }}>
          近 7 天请求趋势
        </Text>
        <Line data={chartData} options={chartOptions as any} height={80} />
      </Card>
    </div>
  )
}
