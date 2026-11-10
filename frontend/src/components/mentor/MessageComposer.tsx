import { ActionIcon, Button, Group, Stack, Textarea } from '@mantine/core'
import { IconArrowRight } from '@tabler/icons-react'
import type { FormEvent, KeyboardEvent, ReactNode } from 'react'

interface Props {
  value: string
  onChange: (v: string) => void
  onSend: () => void
  busy?: boolean
  label: string
  placeholder: string
  quick?: string[]
  footer?: ReactNode
  accent?: string
}

// Поле ввода переписки: Enter – отправить, Shift+Enter – новая строка, быстрые ответы.
export function MessageComposer({
  value,
  onChange,
  onSend,
  busy,
  label,
  placeholder,
  quick = [],
  footer,
  accent = '#04E061',
}: Props) {
  const submit = (e?: FormEvent) => {
    e?.preventDefault()
    if (value.trim() && !busy) onSend()
  }
  const onKey = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) submit(e)
  }
  return (
    <Stack gap={10}>
      {quick.length > 0 && (
        <Group gap={8}>
          {quick.map((q) => (
            <Button key={q} size="xs" radius="xl" variant="default" onClick={() => onChange(q)}>
              {q}
            </Button>
          ))}
        </Group>
      )}
      <form
        onSubmit={submit}
        style={{
          display: 'flex',
          gap: 8,
          alignItems: 'flex-end',
          background: '#FFFFFF',
          border: '1px solid #D6D6D6',
          borderRadius: 16,
          padding: '8px 8px 8px 16px',
        }}
      >
        <Textarea
          aria-label={label}
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.currentTarget.value)}
          onKeyDown={onKey}
          autosize
          minRows={1}
          maxRows={6}
          variant="unstyled"
          style={{ flex: 1 }}
        />
        <ActionIcon
          type="submit"
          size={40}
          radius="md"
          aria-label="Отправить"
          loading={busy}
          disabled={!value.trim()}
          style={{ background: value.trim() ? accent : '#D6D6D6', color: '#141414' }}
        >
          <IconArrowRight size={20} />
        </ActionIcon>
      </form>
      {footer}
    </Stack>
  )
}
