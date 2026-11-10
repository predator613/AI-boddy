import { Button, Group, Paper, Stack, Text } from '@mantine/core'
import { Link } from '@tanstack/react-router'
import { meetingParts } from '../../lib/pathUi'
import type { TPathView } from '../../schemas/path'

interface MeetingCardProps {
  mentor: NonNullable<TPathView['intern']['mentor']> | null
  meeting: TPathView['nextMeeting']
  now?: Date
}

// Наставник и ближайшая встреча: помощник разгружает наставника, но не заменяет его.
export function MeetingCard({ mentor, meeting, now }: MeetingCardProps) {
  const parts = meeting ? meetingParts(meeting.startsAt, now) : null
  return (
    <Paper radius="lg" p="lg">
      <Stack gap={14}>
        <Group gap={14} wrap="nowrap">
          <span
            style={{
              width: 48,
              height: 48,
              borderRadius: 24,
              background: '#DFCDF3',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 18,
              fontWeight: 600,
              flex: 'none',
            }}
          >
            {mentor ? mentor.name.charAt(0) : '?'}
          </span>
          <div>
            <Text fw={500}>
              {mentor ? `${mentor.name} · твой наставник` : 'Наставник пока не назначен'}
            </Text>
            <Text size="sm" c="ink.5">
              Сложные вопросы AI-buddy передаст ему сам
            </Text>
          </div>
        </Group>
        {meeting && parts && (
          <Group gap={12} wrap="nowrap" bg="#F4F4F4" px={14} py={12}>
            <div style={{ textAlign: 'center', lineHeight: 1 }}>
              <Text size="11px" tt="uppercase" c="ink.5">
                {parts.weekday}
              </Text>
              <Text fz={24} fw={500}>
                {parts.date}
              </Text>
            </div>
            <div>
              <Text size="sm">
                {meeting.title}, {parts.time}
              </Text>
              <Text size="xs" c="ink.5">
                {parts.inText}
              </Text>
            </div>
          </Group>
        )}
        {mentor && (
          <Button
            component={Link}
            to="/support"
            variant="outline"
            color="ink"
            style={{ alignSelf: 'flex-start' }}
          >
            Написать наставнику
          </Button>
        )}
      </Stack>
    </Paper>
  )
}
