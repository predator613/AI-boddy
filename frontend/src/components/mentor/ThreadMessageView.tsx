import { Anchor, Button, Group, Stack, Text } from '@mantine/core'
import { IconBook2, IconCalendarEvent, IconCheck, IconListCheck } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { timeLabel } from '../../lib/mentorUi'
import type { TThreadMessage } from '../../schemas/mentor'
import { Mascot } from '../Mascot'

const UPDATE = {
  plan: {
    label: 'План изменен',
    icon: IconListCheck,
    bg: '#C9FFBF',
    ack: 'Понятно',
    acked: 'Принято',
  },
  material: {
    label: 'Новый материал',
    icon: IconBook2,
    bg: '#BADAFB',
    ack: 'Отметить прочитанным',
    acked: 'Прочитано',
  },
  meeting: {
    label: 'Встреча',
    icon: IconCalendarEvent,
    bg: '#DFCDF3',
    ack: 'Принять',
    acked: 'Принято, наставник видит ответ',
  },
} as const

interface Props {
  message: TThreadMessage
  /** Кто смотрит: свои сообщения справа. */
  viewer: 'intern' | 'mentor'
  onAck?: (id: string) => void
  ackBusy?: boolean
}

// Сообщение переписки стажера и наставника: текст, передача от AI-buddy или карточка изменения.
export function ThreadMessageView({ message: m, viewer, onAck, ackBusy }: Props) {
  // Вопрос, который AI-buddy не смог закрыть сам и передал наставнику: стоит как сообщение стажера,
  // но с пометкой, откуда он пришел, и подсказкой наставнику, что делать.
  if (m.kind === 'handoff') {
    const mine = viewer === 'intern'
    return (
      <Stack
        className="ab-pop"
        gap={8}
        aria-label="Вопрос, переданный AI-buddy"
        style={{
          alignSelf: mine ? 'flex-end' : 'flex-start',
          maxWidth: '75%',
          background: '#FFFFFF',
          outline: '2px solid #965EEB',
          outlineOffset: -2,
          borderRadius: mine ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
          padding: '12px 16px',
        }}
      >
        <Group gap={6} wrap="nowrap">
          <Mascot size={20} mood="mentor" />
          <Text size="xs" c="ink.5">
            {mine ? 'Ты спросил через AI-buddy' : 'AI-buddy передал вопрос стажера'} ·{' '}
            {timeLabel(m.at)}
          </Text>
        </Group>
        <Text fz={16} lh={1.4}>
          {m.text}
        </Text>
        {!mine && (
          <Text size="xs" c="ink.5">
            Ответь ниже: ответ получит стажер. С галочкой «Добавить ответ в справочник» AI-buddy
            дальше будет отвечать на такой вопрос сам.
          </Text>
        )}
      </Stack>
    )
  }

  if (m.kind === 'update' && m.update) {
    const u = UPDATE[m.update.type]
    return (
      <Group
        className="ab-pop"
        align="flex-start"
        gap={10}
        wrap="nowrap"
        style={{ maxWidth: '85%' }}
      >
        <span
          style={{
            width: 32,
            height: 32,
            borderRadius: 16,
            background: u.bg,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flex: 'none',
          }}
        >
          <u.icon size={16} stroke={1.8} />
        </span>
        <Stack
          gap={8}
          p="md"
          style={{ background: m.update.type === 'meeting' ? '#DFCDF3' : '#FFFFFF', minWidth: 260 }}
        >
          <Text size="11px" fw={600} tt="uppercase" style={{ letterSpacing: '0.06em' }}>
            {u.label} · {m.authorName}
          </Text>
          <Text fw={500} fz={16}>
            {m.update.title}
          </Text>
          <Text size="sm" c="ink.5">
            {m.update.detail}
          </Text>
          {viewer === 'intern' &&
            (m.update.acked ? (
              <Group gap={6}>
                <IconCheck size={14} stroke={3} />
                <Text size="sm" fw={500}>
                  {u.acked}
                </Text>
              </Group>
            ) : (
              <Group gap={8}>
                <Button size="xs" color="ink" loading={ackBusy} onClick={() => onAck?.(m.id)}>
                  {u.ack}
                </Button>
                {m.update.link && (
                  <Anchor
                    renderRoot={(p) => <Link {...p} to={m.update!.link as '/kb'} />}
                    size="sm"
                    c="ink.9"
                    fw={500}
                  >
                    Открыть
                  </Anchor>
                )}
              </Group>
            ))}
          {viewer === 'mentor' && (
            <Text size="xs" c="ink.5">
              {m.update.acked ? 'Стажер отметил, что ознакомился' : 'Стажер еще не открыл'}
            </Text>
          )}
          <Text size="xs" c="ink.5">
            {timeLabel(m.at)}
          </Text>
        </Stack>
      </Group>
    )
  }

  const mine = (viewer === 'mentor') === (m.author === 'mentor')
  return (
    <Stack
      gap={4}
      className="ab-pop"
      style={{ alignSelf: mine ? 'flex-end' : 'flex-start', maxWidth: '78%' }}
    >
      <div
        style={{
          background: mine ? '#141414' : '#FFFFFF',
          color: mine ? '#FFFFFF' : '#141414',
          borderRadius: mine ? '16px 16px 4px 16px' : '4px 16px 16px 16px',
          borderLeft: !mine && m.author === 'mentor' ? '3px solid #965EEB' : undefined,
          padding: '12px 16px',
          fontSize: 16,
          lineHeight: 1.45,
          whiteSpace: 'pre-wrap',
        }}
      >
        {m.text}
      </div>
      <Text size="xs" c="ink.5" ta={mine ? 'right' : 'left'}>
        {mine ? 'ты' : m.authorName} · {timeLabel(m.at)}
      </Text>
    </Stack>
  )
}
