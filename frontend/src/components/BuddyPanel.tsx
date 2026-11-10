import { ActionIcon, Group, Paper, Stack, Text, TextInput, UnstyledButton } from '@mantine/core'
import { IconArrowRight, IconMaximize } from '@tabler/icons-react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type FormEvent } from 'react'
import { Mascot } from './Mascot'

interface BuddyPanelProps {
  text: string
  suggestions: string[]
}

// Панель помощника на главной: подсказка по текущему этапу и быстрый вопрос в чат.
export function BuddyPanel({ text, suggestions }: BuddyPanelProps) {
  const navigate = useNavigate()
  const [value, setValue] = useState('')
  const ask = (q: string) => navigate({ to: '/chat', search: { q } })
  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (value.trim()) ask(value.trim())
  }
  return (
    <Paper
      component="aside"
      aria-label="AI-buddy"
      radius="lg"
      p="lg"
      style={{
        flex: '1 1 320px',
        maxWidth: 400,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: 640,
        position: 'sticky',
        top: 32,
      }}
    >
      <Group gap={10} wrap="nowrap">
        <Mascot size={36} />
        <div style={{ flex: 1 }}>
          <Text fw={600}>AI-buddy</Text>
          <Text size="xs" c="ink.5">
            помощник стажера
          </Text>
        </div>
        <ActionIcon
          component={Link}
          to="/chat"
          variant="subtle"
          color="ink"
          size="lg"
          aria-label="Открыть чат на весь экран"
        >
          <IconMaximize size={20} />
        </ActionIcon>
      </Group>
      <Stack align="center" justify="center" style={{ flex: 1, position: 'relative' }}>
        <div style={{ position: 'relative' }}>
          <Mascot size={132} glow float />
          <span
            className="ab-hand"
            style={{
              position: 'absolute',
              right: -54,
              top: 4,
              fontSize: 22,
              transform: 'rotate(-8deg)',
            }}
          >
            спроси меня!
          </span>
        </div>
      </Stack>
      <div
        className="ab-pop"
        key={text}
        style={{
          background: '#F4F4F4',
          borderRadius: '16px 16px 16px 4px',
          padding: '12px 14px',
          fontSize: 14,
          lineHeight: 1.4,
        }}
      >
        {text}
      </div>
      <Group gap={8}>
        {suggestions.map((s) => (
          <UnstyledButton
            key={s}
            onClick={() => ask(s)}
            style={{
              height: 32,
              padding: '0 12px',
              border: '1px solid #D6D6D6',
              borderRadius: 16,
              fontSize: 13,
            }}
          >
            {s}
          </UnstyledButton>
        ))}
      </Group>
      <form onSubmit={submit}>
        <TextInput
          aria-label="Сообщение помощнику"
          placeholder="Спроси что-нибудь…"
          radius="xl"
          size="md"
          value={value}
          onChange={(e) => setValue(e.currentTarget.value)}
          rightSectionWidth={48}
          rightSection={
            <ActionIcon type="submit" radius="xl" size={36} aria-label="Отправить">
              <IconArrowRight size={20} />
            </ActionIcon>
          }
        />
      </form>
    </Paper>
  )
}
