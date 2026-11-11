import { tickColors } from '../../lib/pathLook'
import type { TPathView } from '../../schemas/path'

// Кольцо из 60 штрихов: на каждую задачу поровну, готовые зеленые, в работе синие, остальные серые.
export function ProgressTicks({ path }: { path: Pick<TPathView, 'stages' | 'progress'> }) {
  const colors = tickColors(path.stages)
  const { percent, done, total, inProgress } = path.progress
  return (
    <svg
      width="100%"
      height="100%"
      viewBox="0 0 100 100"
      role="img"
      aria-label={`Пройдено ${percent}%: ${done} из ${total} задач, ${inProgress} в работе`}
    >
      {colors.map((c, i) => (
        <line
          key={i}
          x1="50"
          y1="3"
          x2="50"
          y2="13"
          stroke={c}
          strokeWidth="2.4"
          strokeLinecap="round"
          transform={`rotate(${i * (360 / colors.length)} 50 50)`}
          style={{ transition: 'stroke .3s ease' }}
        />
      ))}
      <text
        x="50"
        y="53"
        textAnchor="middle"
        fontFamily="Onest, sans-serif"
        fontSize="17"
        fontWeight="500"
        fill="#141414"
        letterSpacing="-0.5"
      >
        {percent}%
      </text>
      <text
        x="50"
        y="62"
        textAnchor="middle"
        fontFamily="Onest, sans-serif"
        fontSize="4.6"
        fill="#5C5C5C"
      >
        {done} из {total} задач
      </text>
    </svg>
  )
}
