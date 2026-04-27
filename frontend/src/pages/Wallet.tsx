import React, { useEffect, useState, useCallback } from 'react'
import { Typography, Card, Button, Input, Table, Toast, Divider } from '@douyinfe/semi-ui'
import { getBalance, redeemCode, getTopupLogs, getBranding } from '../api'
import { useAppStore } from '../store'
import type { TopupLog } from '../types'

const { Title, Text } = Typography

export default function Wallet() {
  const { user, setUser } = useAppStore()
  const [balance, setBalance] = useState(user?.balance ?? 0)
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [logs, setLogs] = useState<TopupLog[]>([])
  const [purchaseUrl, setPurchaseUrl] = useState('')

  const loadData = useCallback(() => {
    getBalance().then((res) => setBalance(res.data.balance))
    getTopupLogs().then((res) => setLogs(res.data.data ?? []))
  }, [])

  useEffect(() => { loadData() }, [loadData])

  useEffect(() => {
    getBranding()
      .then((res) => setPurchaseUrl(res.data.redeem_purchase_url || ''))
      .catch(() => {})
  }, [])

  const handleRedeem = async () => {
    if (!code.trim()) {
      Toast.warning('请输入兑换码')
      return
    }
    setLoading(true)
    try {
      const res = await redeemCode(code.trim())
      Toast.success(`兑换成功，增加余额 $${res.data.amount}`)
      setBalance(res.data.balance)
      if (user) setUser({ ...user, balance: res.data.balance })
      setCode('')
      loadData()
    } catch {
      // handled by interceptor
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    { title: '时间', dataIndex: 'created_at', render: (v: string) => new Date(v).toLocaleString() },
    { title: '金额', dataIndex: 'amount', render: (v: number) => <Text style={{ color: '#22c55e' }}>+${v.toFixed(4)}</Text> },
    { title: '兑换码', dataIndex: 'code' },
    { title: '备注', dataIndex: 'remark' },
  ]

  return (
    <div>
      <Title heading={4} style={{ marginBottom: 16 }}>钱包</Title>

      <Card style={{ marginBottom: 16 }}>
        <Text type="tertiary">当前余额</Text>
        <Title heading={1} style={{ color: '#6366f1', margin: '8px 0' }}>
          ${balance.toFixed(4)}
        </Title>
        <Text type="tertiary" size="small">余额以美元计价，用于支付 API 使用费用</Text>
      </Card>

      <Card title="兑换充值码" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 12, maxWidth: 560 }}>
          <Input
            value={code}
            onChange={(v) => setCode(v)}
            placeholder="输入兑换码"
            onEnterPress={handleRedeem}
            style={{ flex: 1 }}
          />
          <Button type="primary" theme="solid" loading={loading} onClick={handleRedeem}>
            兑换
          </Button>
          {purchaseUrl && (
            <Button
              type="tertiary"
              theme="solid"
              onClick={() => window.open(purchaseUrl, '_blank')}
            >
              购买兑换码
            </Button>
          )}
        </div>
      </Card>

      <Card title="充值记录">
        <Table
          columns={columns}
          dataSource={logs}
          rowKey="id"
          pagination={false}
          empty={<Text type="tertiary">暂无充值记录</Text>}
        />
      </Card>
    </div>
  )
}
