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

  // Sanitize HTML: only allow safe tags/attributes, strip everything else
  const sanitizeHTML = (html: string): string => {
    const allowedTags = new Set([
      'b', 'i', 'u', 'strong', 'em', 'a', 'p', 'br', 'ul', 'ol', 'li',
      'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'span', 'div', 'blockquote', 'code', 'pre', 'hr',
    ])
    const allowedAttrs = new Set(['href', 'target', 'rel', 'class', 'id'])
    const uriAttrs = new Set(['href'])

    const div = document.createElement('div')
    div.innerHTML = html

    const clean = (node: Node): void => {
      if (node.nodeType === 3) return // text node, keep
      if (node.nodeType !== 1) {
        node.parentNode?.removeChild(node)
        return
      }
      const el = node as HTMLElement
      const tag = el.tagName.toLowerCase()
      if (!allowedTags.has(tag)) {
        // Replace with its children
        while (el.firstChild) {
          el.parentNode!.insertBefore(el.firstChild, el)
        }
        el.parentNode!.removeChild(el)
        return
      }
      // Remove disallowed attributes
      for (let i = el.attributes.length - 1; i >= 0; i--) {
        const name = el.attributes[i].name.toLowerCase()
        if (!allowedAttrs.has(name)) {
          el.removeAttribute(name)
        } else if (uriAttrs.has(name)) {
          const val = el.getAttribute(name) || ''
          if (/^javascript:/i.test(val) || /^data:/i.test(val)) {
            el.removeAttribute(name)
          }
        }
      }
      // Ensure links have rel="noopener noreferrer" and target="_blank"
      if (tag === 'a') {
        el.setAttribute('rel', 'noopener noreferrer')
        el.setAttribute('target', '_blank')
      }
      // Recurse (use a copy since children may be removed during iteration)
      const children = Array.from(el.childNodes)
      children.forEach(clean)
    }

    const children = Array.from(div.childNodes)
    children.forEach(clean)
    return div.innerHTML
  }

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
            dangerouslySetInnerHTML={{ __html: sanitizeHTML(current.content) }}
          />
        </div>
      )}
    </Modal>
  )
}
