import type { CSSProperties, ReactNode } from 'react'

interface StatusTagProps {
  bg: string
  border?: string
  color?: string
  children: ReactNode
  size?: 'sm' | 'md'
  style?: CSSProperties
}

// Прямоугольная плашка статуса, как стикеры в кейсе: светлая заливка и темный текст.
export function StatusTag({
  bg,
  border = '0',
  color = '#141414',
  children,
  size = 'sm',
  style,
}: StatusTagProps) {
  return (
    <span
      style={{
        display: 'inline-block',
        background: bg,
        border,
        color,
        fontSize: size === 'sm' ? 12 : 13,
        fontWeight: 500,
        padding: size === 'sm' ? '3px 9px' : '4px 10px',
        whiteSpace: 'nowrap',
        lineHeight: 1.3,
        ...style,
      }}
    >
      {children}
    </span>
  )
}
