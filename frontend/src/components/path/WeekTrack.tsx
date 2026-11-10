import { Group, Stack, Text } from '@mantine/core'
import type { TPathView } from '../../schemas/path'
import { Mascot } from '../Mascot'

// Путь стажировки: одно деление – одна неделя, над текущей неделей маскот.
export function WeekTrack({ week }: { week: TPathView['week'] }) {
  const milestoneWeeks = new Set(week.milestones.map((m) => m.week))
  const weeks = Array.from({ length: week.total }, (_, i) => i + 1)
  return (
    <Stack gap={8}>
      <Group justify="space-between">
        <Text size="xs" c="ink.5">
          Путь стажировки
        </Text>
        <Text size="xs" c="ink.5">
          неделя {week.current} из {week.total}
        </Text>
      </Group>
      <div
        style={{ display: 'flex', alignItems: 'flex-end', gap: 3, height: 52 }}
        aria-hidden="true"
      >
        {weeks.map((w) => {
          const current = w === week.current
          const milestone = milestoneWeeks.has(w)
          const color =
            w < week.current ? '#04E061' : current ? '#00AAFF' : milestone ? '#D6D6D6' : '#EDEDED'
          return (
            <div
              key={w}
              style={{
                flex: 1,
                height: 52,
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'flex-end',
                alignItems: 'center',
                position: 'relative',
              }}
            >
              {current && (
                <div style={{ position: 'absolute', top: 0 }}>
                  <Mascot size={22} float />
                </div>
              )}
              <div
                data-week={w}
                style={{
                  width: '100%',
                  height: current ? 16 : milestone ? 12 : 8,
                  background: color,
                }}
              />
            </div>
          )
        })}
      </div>
      <Group justify="space-between" gap="xs">
        {week.milestones.map((m) => (
          <Text key={m.week} size="11px" c="ink.5">
            {m.title}
            {m.week > 1 ? `, нед. ${m.week}` : ''}
          </Text>
        ))}
      </Group>
    </Stack>
  )
}
