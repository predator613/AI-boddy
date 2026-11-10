import { useEffect } from 'react'
import { useToastStore } from '../store/toastStore'
import { Mascot } from './Mascot'

const TTL = 3400

export function Toast() {
  const text = useToastStore((s) => s.text)
  const hide = useToastStore((s) => s.hide)
  useEffect(() => {
    if (!text) return
    const t = setTimeout(hide, TTL)
    return () => clearTimeout(t)
  }, [text, hide])
  if (!text) return null
  return (
    <div
      role="status"
      className="ab-rise"
      style={{
        position: 'fixed',
        right: 24,
        bottom: 24,
        zIndex: 300,
        maxWidth: 420,
        background: '#141414',
        color: '#FFFFFF',
        padding: '12px 16px 12px 12px',
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        boxShadow: '0 12px 32px rgba(20, 20, 20, 0.25)',
      }}
    >
      <Mascot size={36} mood="happy" />
      <span style={{ fontSize: 14, lineHeight: 1.35 }}>{text}</span>
    </div>
  )
}
