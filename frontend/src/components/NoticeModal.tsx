import React, { useEffect, useState } from 'react'
import { Modal, Typography, Button, Space } from '@douyinfe/semi-ui'
import { getNotices } from '../api'

const { Title, Text } = Typography

export interface NoticeItem {
  id: number
  title: string
  content: string
  priority: number
  status: number
  created_at: string
}

const DISMISS_KEY = 'notice_dismissed_date'

function getTodayKey(): string {
  return new Date().toISOString().slice(0, 10)
}

function isDismissedToday(): boolean {
  try {
    return localStorage.getItem(DISMISS_KEY) === getTodayKey()
  } catch {
    return false
  }
}

function dismissToday() {
  try {
    localStorage.setItem(DISMISS_KEY, getTodayKey())
  } catch { /* ignore */ }
}

export default function NoticeModal() {
  const [notices, setNotices] = useState<NoticeItem[]>([])
  const [visible, setVisible] = useState(false)
  const [currentIndex, setCurrentIndex] = useState(0)

  useEffect(() => {
    if (isDismissedToday()) return

    getNotices()
      .then((res) => {
        const list: NoticeItem[] = res.data.data ?? []
        if (list.length > 0) {
          setNotices(list)
          setCurrentIndex(0)

          // Auto-show after a short delay so the page loads first
          const timer = setTimeout(() => setVisible(true), 500)
          return () => clearTimeout(timer)
        }
      })
      .catch(() => {})
  }, [])

  // Auto-link bare URLs and add target="_blank" to all links
  const processContent = (html: string) =>
    html
      .replace(/(https?:\/\/[^\s<>"']+)/g, '<a href="$1">$1</a>')
      .replace(/<a\s/gi, '<a target="_blank" rel="noopener noreferrer" ')

  const current = notices[currentIndex]

  const handlePrev = () => {
    setCurrentIndex((i) => Math.max(0, i - 1))
  }

  const handleNext = () => {
    setCurrentIndex((i) => Math.min(notices.length - 1, i + 1))
  }

  const handleClose = () => {
    setVisible(false)
  }

  const handleDismissToday = () => {
    dismissToday()
    setVisible(false)
  }

  return (
    <Modal
      title={current?.title || '公告'}
      visible={visible}
      onCancel={handleClose}
      footer={
        <Space>
          <Button onClick={handleDismissToday}>今天不再显示</Button>
          {notices.length > 1 && (
            <>
              <Button disabled={currentIndex === 0} onClick={handlePrev}>
                上一条
              </Button>
              <Button disabled={currentIndex === notices.length - 1} onClick={handleNext}>
                下一条
              </Button>
            </>
          )}
          <Button type="primary" onClick={handleClose}>
            关闭
          </Button>
        </Space>
      }
      width={560}
      style={{ maxHeight: '70vh' }}
    >
      {current && (
        <div>
          {notices.length > 1 && (
            <Text type="tertiary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
              {currentIndex + 1} / {notices.length}
            </Text>
          )}
          <div
            style={{
              lineHeight: 1.8,
              fontSize: 14,
              maxHeight: 400,
              overflowY: 'auto',
            }}
            dangerouslySetInnerHTML={{ __html: processContent(current.content) }}
          />
        </div>
      )}
    </Modal>
  )
}
