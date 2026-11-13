import { Anchor, Button, Group, Paper, Stack, Text, UnstyledButton } from '@mantine/core'
import { IconBook2, IconCheck } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { mentorCheckText } from '../../lib/jira'
import type { TChatMessage } from '../../schemas/assistant'
import { Mascot } from '../Mascot'
import { StatusTag } from '../StatusTag'

interface ChatMessageViewProps {
  message: TChatMessage
  onConfirm?: (actionId: string) => void
  onCancel?: (actionId: string) => void
  onEdit?: () => void
  /** «Свериться с наставником» перед отправкой заявки: открыть чат с наставником с готовым текстом. */
  onAskMentor?: (text: string) => void
  busy?: boolean
}

const STATUS_TEXT = {
  waiting: 'на согласовании',
  approved: 'одобрена',
  done: 'выполнена',
  rejected: 'отклонена',
} as const

export function ChatMessageView({
  message: m,
  onConfirm,
  onCancel,
  onEdit,
  onAskMentor,
  busy,
}: ChatMessageViewProps) {
  // Заявка уже выполнена или одобрена – обещать «напишу позже» незачем.
  const finished = m.result?.status === 'approved' || m.result?.status === 'done'
  if (m.role === 'user') {
    return (
      <div
        className="ab-pop"
        style={{
          alignSelf: 'flex-end',
          maxWidth: '75%',
          background: '#141414',
          color: '#FFFFFF',
          borderRadius: '16px 16px 4px 16px',
          padding: '12px 16px',
          fontSize: 16,
          lineHeight: 1.45,
          whiteSpace: 'pre-wrap',
        }}
      >
        {m.text}
      </div>
    )
  }
  return (
    <Group className="ab-pop" align="flex-start" gap={10} wrap="nowrap" style={{ maxWidth: '85%' }}>
      <Mascot size={32} mood={m.mood} label="AI-buddy" />
      {m.kind === 'confirm' && m.confirm ? (
        <Paper
          radius="lg"
          p="md"
          style={{
            outline: '2px solid #965EEB',
            outlineOffset: -2,
            minWidth: 280,
            borderTopLeftRadius: 4,
          }}
        >
          <Stack gap={12}>
            <div>
              <StatusTag
                bg="#DFCDF3"
                style={{
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  fontSize: 11,
                  fontWeight: 600,
                }}
              >
                Нужно подтверждение
              </StatusTag>
            </div>
            <Text fz={18} fw={500}>
              {m.confirm.title}
            </Text>
            <dl
              style={{
                display: 'grid',
                gridTemplateColumns: '120px 1fr',
                gap: '6px 12px',
                margin: 0,
                fontSize: 14,
              }}
            >
              {m.confirm.fields.map((f) => (
                <div key={f.label} style={{ display: 'contents' }}>
                  <dt style={{ color: '#5C5C5C' }}>{f.label}</dt>
                  <dd style={{ margin: 0 }}>{f.value}</dd>
                </div>
              ))}
            </dl>
            {m.confirm.state === 'pending' ? (
              <Group gap={8}>
                <Button color="ink" loading={busy} onClick={() => onConfirm?.(m.confirm!.actionId)}>
                  Подтвердить
                </Button>
                {onEdit && (
                  <Button variant="outline" color="ink" disabled={busy} onClick={onEdit}>
                    Изменить
                  </Button>
                )}
                <Button
                  variant="subtle"
                  color="ink"
                  disabled={busy}
                  onClick={() => onCancel?.(m.confirm!.actionId)}
                >
                  Отмена
                </Button>
                {onAskMentor && (
                  <UnstyledButton
                    onClick={() =>
                      onAskMentor(mentorCheckText(m.confirm!.title, m.confirm!.fields))
                    }
                    style={{ flexBasis: '100%', fontSize: 13, color: '#5C5C5C' }}
                  >
                    Не уверен? <u>Сверься с наставником</u>
                  </UnstyledButton>
                )}
              </Group>
            ) : (
              <Text size="sm" c="ink.5">
                {m.confirm.state === 'confirmed' ? 'Подтверждено' : 'Отменено'}
              </Text>
            )}
          </Stack>
        </Paper>
      ) : m.kind === 'result' && m.result ? (
        <div
          style={{
            background: '#C9FFBF',
            borderRadius: '4px 16px 16px 16px',
            padding: 16,
            minWidth: 280,
            display: 'flex',
            flexDirection: 'column',
            gap: 10,
          }}
        >
          <Group gap={10} wrap="nowrap">
            <span
              style={{
                width: 28,
                height: 28,
                borderRadius: 14,
                background: '#04E061',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flex: 'none',
              }}
            >
              <IconCheck size={15} stroke={3} aria-hidden="true" />
            </span>
            <Text fz={18} fw={500}>
              {finished ? 'Готово' : `Заявка №${m.result.number} создана`}
            </Text>
          </Group>
          <Text size="sm">
            {m.result.title}. Статус: <b>{STATUS_TEXT[m.result.status]}</b>.
            {finished ? '' : ' Напишу, когда будет решение.'}
          </Text>
          <Button
            renderRoot={(p) => <Link {...p} to="/requests" search={{ id: m.result!.requestId }} />}
            variant="outline"
            color="ink"
            size="sm"
            style={{ alignSelf: 'flex-start' }}
          >
            Открыть заявку
          </Button>
        </div>
      ) : (
        <div
          style={{
            background: m.kind === 'error' ? '#F7C1C5' : '#FFFFFF',
            borderRadius: '4px 16px 16px 16px',
            padding: '14px 16px',
            display: 'flex',
            flexDirection: 'column',
            gap: 10,
          }}
        >
          <Text fz={16} lh={1.45} style={{ whiteSpace: 'pre-wrap' }}>
            {m.text}
          </Text>
          {m.source && (
            <Anchor
              renderRoot={(p) => (
                <Link
                  {...p}
                  to="/kb"
                  search={m.source!.articleId ? { article: m.source!.articleId } : {}}
                />
              )}
              size="sm"
              c="ink.6"
              underline="never"
              style={{
                alignSelf: 'flex-start',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                border: '1px solid #D6D6D6',
                padding: '4px 10px',
              }}
            >
              <IconBook2 size={14} aria-hidden="true" />
              Источник: {m.source.title}
            </Anchor>
          )}
        </div>
      )}
    </Group>
  )
}

export function ThinkingBubble() {
  return (
    <Group gap={10} wrap="nowrap" align="center">
      <Mascot size={32} mood="think" label="AI-buddy" />
      <div
        style={{
          background: '#FFFFFF',
          borderRadius: '4px 16px 16px 16px',
          padding: '12px 16px',
          display: 'flex',
          alignItems: 'center',
          gap: 10,
        }}
      >
        <span style={{ display: 'flex', gap: 4 }} aria-hidden="true">
          <span
            className="ab-dot"
            style={{ width: 7, height: 7, borderRadius: 4, background: '#00AAFF' }}
          />
          <span
            className="ab-dot"
            style={{ width: 7, height: 7, borderRadius: 4, background: '#00AAFF' }}
          />
          <span
            className="ab-dot"
            style={{ width: 7, height: 7, borderRadius: 4, background: '#00AAFF' }}
          />
        </span>
        <Text size="sm" c="ink.5" role="status">
          AI-buddy думает…
        </Text>
      </div>
    </Group>
  )
}
