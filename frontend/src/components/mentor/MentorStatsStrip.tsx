import { SimpleGrid, Stack, Text } from '@mantine/core'
import { Link } from '@tanstack/react-router'
import { useMentorStats } from '../../hooks/useMentor'

// Сколько работы AI-buddy снял с наставника и сколько времени ушло на наставничество.
export function MentorStatsStrip() {
  const stats = useMentorStats()
  if (!stats.data) return null
  const s = stats.data
  const hours = Math.round(s.savedMinutes / 6) / 10
  const pct = Math.min(100, Math.round((s.mentorHoursSpent / s.mentorHoursPlan) * 100))
  return (
    <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md" className="ab-rise">
      <Stack gap={4} p="md" style={{ background: '#C9FFBF' }}>
        <Text size="xs">AI-buddy ответил без тебя за {s.periodDays} дней</Text>
        <Text fz={30} fw={500} lh={1.1}>
          {s.answeredByBuddy} вопр.
        </Text>
        <Text size="xs">≈ {hours} ч сэкономлено на повторных объяснениях</Text>
      </Stack>
      <Stack
        gap={4}
        p="md"
        style={{ background: '#DFCDF3', textDecoration: 'none', color: 'inherit' }}
        renderRoot={(p) => <Link {...p} to="/inbox" />}
        aria-label={`Передано тебе: ${s.handedOff}. Открыть сообщения`}
      >
        <Text size="xs">Передано тебе</Text>
        <Text fz={30} fw={500} lh={1.1}>
          {s.handedOff} вопр.
        </Text>
        <Text size="xs">AI-buddy не нашел ответ в материалах → открыть</Text>
      </Stack>
      <Stack gap={6} p="md" style={{ background: '#FFFFFF' }}>
        <Text size="xs" c="ink.5">
          Твое время на наставничество
        </Text>
        <Text fz={30} fw={500} lh={1.1}>
          {s.mentorHoursSpent} из ~{s.mentorHoursPlan} ч
        </Text>
        <div style={{ height: 6, background: '#F4F4F4' }}>
          <div style={{ height: 6, width: `${pct}%`, background: '#965EEB' }} />
        </div>
        <Text size="xs" c="ink.5">
          ориентир на 6-месячную программу: 40–48 ч
        </Text>
      </Stack>
    </SimpleGrid>
  )
}
