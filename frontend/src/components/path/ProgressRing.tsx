import { ringSegments } from '../../lib/pathUi'
import type { TPathView } from '../../schemas/path'

interface ProgressRingProps {
  path: Pick<TPathView, 'stages' | 'progress'>
  size: number
  compact?: boolean
}

// Кольцо прогресса: одно деление – одна задача (зеленые готовы, синие в работе, серые впереди).
export function ProgressRing({ path, size, compact = false }: ProgressRingProps) {
  const segments = ringSegments(path.stages)
  const { percent, done, total } = path.progress
  return (
    <div
      style={{
        width: size,
        height: size,
        flex: 'none',
        transition: 'width .45s cubic-bezier(.2,.8,.2,1), height .45s cubic-bezier(.2,.8,.2,1)',
      }}
    >
      <svg
        width="100%"
        height="100%"
        viewBox="0 0 100 100"
        role="img"
        aria-label={`Пройдено ${percent}% адаптации: ${done} из ${total} задач`}
      >
        {segments.map((s, i) => (
          <circle
            key={i}
            cx="50"
            cy="50"
            r="42"
            fill="none"
            stroke={s.color}
            strokeWidth={compact ? 11 : 9}
            strokeDasharray={s.dash}
            transform={`rotate(${s.rotate} 50 50)`}
            style={{ transition: 'stroke .3s ease' }}
          />
        ))}
        <text
          x="50"
          y={compact ? 57 : 54}
          textAnchor="middle"
          fontSize="19"
          fontWeight="500"
          fill="#141414"
          letterSpacing="-0.5"
          fontFamily="Onest, sans-serif"
        >
          {percent}%
        </text>
        {!compact && (
          <text
            x="50"
            y="62"
            textAnchor="middle"
            fontSize="5.6"
            fill="#5C5C5C"
            fontFamily="Onest, sans-serif"
          >
            {done} из {total} задач
          </text>
        )}
      </svg>
    </div>
  )
}
