export default function ChatEmbed() {
  return (
    <iframe
      src="/chat"
      title="在线聊天"
      style={{
        width: '100%',
        height: 'calc(100vh - 104px)',
        border: 'none',
        borderRadius: 8,
        display: 'block',
      }}
    />
  )
}
