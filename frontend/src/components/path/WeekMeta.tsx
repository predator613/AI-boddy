import { Text } from '@mantine/core'
import { weekBars } from '../../lib/pathLook'
import type { TPathView } from '../../schemas/path'

// Неделя стажировки в шапке: тихая подпись и шкала из недель вместо плашек-кнопок.
export function WeekMeta({ path }: { path: Pick<TPathView, 'week' | 'intern'> }) {
  const { current, total, milestones } = path.week
  const bars = weekBars(
    current,
    total,
    milestones.map((m) => m.week),
  )
  return (
    <div
      aria-label={`Неделя ${current} из ${total}, стажер ${path.intern.teamName}`}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-end',
        gap: 8,
        paddingBottom: 4,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
        <Text size="xs" c="ink.5">
          Стажер · {path.intern.teamName}
        </Text>
        <span style={{ width: 1, height: 12, background: '#D6D6D6', alignSelf: 'center' }} />
        <Text fz={15} fw={500}>
          Неделя {current}{' '}
          <Text span c="ink.5" fw={400} inherit>
            из {total}
          </Text>
        </Text>
      </div>
      <div aria-hidden="true" style={{ display: 'flex', gap: 3, alignItems: 'flex-end' }}>
        {bars.map((b) => (
          <span
            key={b.week}
            style={{ width: 6, height: b.height, borderRadius: 2, background: b.color }}
          />
        ))}
      </div>
    </div>
  )
}
