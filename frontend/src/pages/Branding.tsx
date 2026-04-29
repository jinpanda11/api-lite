import { useEffect, useState } from 'react'
import { Typography, Form, Button, Toast, Card } from '@douyinfe/semi-ui'
import { getSettings, updateSettings } from '../api'

const { Title, Text } = Typography

export default function Branding() {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setLoading(true)
    getSettings()
      .then((res) => {
        const s = res.data || {}
        form.setValues({
          site_name: s.site_name || '',
          site_logo: s.site_logo || '',
          site_title: s.site_title || '',
          site_favicon: s.site_favicon || '',
          site_logo_size: s.site_logo_size || '',
          site_name_size: s.site_name_size || '',
        })
      })
      .catch(() => Toast.error('加载设置失败'))
      .finally(() => setLoading(false))
  }, [form])

  const handleSave = () => {
    form.validate().then((values) => {
      setSaving(true)
      updateSettings(values as Record<string, string>)
        .then(() => Toast.success('已保存，刷新页面生效'))
        .catch(() => Toast.error('保存失败'))
        .finally(() => setSaving(false))
    }).catch(() => Toast.error('请检查字段格式'))
  }

  return (
    <div style={{ maxWidth: 600 }}>
      <Title heading={4} style={{ marginBottom: 8 }}>站点品牌</Title>
      <Text type="tertiary" style={{ marginBottom: 24, display: 'block' }}>
        自定义站点名称、图标和浏览器标签页显示
      </Text>

      <Card loading={loading}>
        <Form form={form} labelPosition="left" labelWidth={120}>
          <Form.Input
            field="site_name"
            label="站点名称"
            placeholder="显示在侧边栏"
            extraText="留空使用默认名称 New API Lite"
          />
          <Form.Input
            field="site_logo"
            label="Logo"
            placeholder="emoji 图标或图片 URL"
            extraText="侧边栏名称旁的图标，支持 emoji（如 ⚡）或图片链接"
          />
          <Form.Input
            field="site_title"
            label="浏览器标题"
            placeholder="浏览器标签页上的标题"
            extraText="留空使用默认标题 New API Lite"
          />
          <Form.Input
            field="site_favicon"
            label="标签页图标"
            placeholder="emoji 图标"
            extraText="浏览器标签页上的小图标，支持 emoji（如 ⚡）"
          />
          <Form.Input
            field="site_logo_size"
            label="Logo 大小"
            placeholder="默认 20（像素）"
            extraText="Logo 字体/图片大小（像素），留空使用默认值"
          />
          <Form.Input
            field="site_name_size"
            label="站点名称大小"
            placeholder="默认 16（像素）"
            extraText="站点名称字体大小（像素），留空使用默认值"
          />

          <Button
            theme="solid"
            type="primary"
            loading={saving}
            onClick={handleSave}
            style={{ marginTop: 16 }}
          >
            保存
          </Button>
        </Form>
      </Card>
    </div>
  )
}
