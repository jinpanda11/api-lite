import React, { useEffect, useState } from 'react'
import { Card, Row, Col, Typography, Spin, Tag, Table } from '@douyinfe/semi-ui'
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
import type { DashboardData, DailyCount } from '../types'
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
  const isAdmin = user?.role === 'admin'

  useEffect(() => {
    getDashboard()
      .then((res) => setData(res.data))
      .finally(() => setLoading(false))
  }, [])

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
    </div>
  )
}
