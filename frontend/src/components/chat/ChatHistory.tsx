import { Button, Text, UnstyledButton } from '@mantine/core'
import { IconPlus } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { getDialogs } from '../../api/chatDialogs'
import type { TChatDialog } from '../../schemas/chatDialogs'
import classes from '../../pages/ChatPage.module.css'

export const dialogsKey = ['chat', 'dialogs'] as const

const KIND_COLOR: Record<TChatDialog['kind'], string> = {
  request: '#5C5C5C',
  handoff: '#965EEB',
  answer: '#5C5C5C',
}

function groupOf(iso: string, now: Date) {
  const d = new Date(iso)
  if (d.toDateString() === now.toDateString()) return 'Сегодня'
  const days = (now.getTime() - d.getTime()) / 86_400_000
  return days < 7 ? 'На этой неделе' : 'Раньше'
}

interface ChatHistoryProps {
  openId: string | null
  onOpen: (id: string | null) => void
  onNew: () => void
  newDisabled: boolean
  newBusy: boolean
}

// Колонка истории диалогов слева от чата (как в макете). Если backend еще не отдает
// список диалогов, показывается только текущий.
export function ChatHistory({ openId, onOpen, onNew, newDisabled, newBusy }: ChatHistoryProps) {
  const dialogs = useQuery({ queryKey: dialogsKey, queryFn: getDialogs, retry: false })
  const list = dialogs.data ?? []
  const past = list.filter((d) => !d.current)
  const now = new Date()
  const groups: { title: string; items: TChatDialog[] }[] = []
  for (const d of past) {
    const g = groupOf(d.updatedAt, now)
    const last = groups[groups.length - 1]
    if (last?.title === g) last.items.push(d)
    else groups.push({ title: g, items: [d] })
  }

  return (
    <aside className={classes.history} aria-label="Диалоги">
      <Button
        color="ink"
        leftSection={<IconPlus size={16} />}
        onClick={onNew}
        loading={newBusy}
        disabled={newDisabled}
        fullWidth
      >
        Новый диалог
      </Button>
      <nav className={classes.historyList}>
        <Text className={classes.historyGroup}>Сейчас</Text>
        <UnstyledButton
          className={classes.historyItem}
          data-active={openId === null || undefined}
          aria-current={openId === null ? 'true' : undefined}
          onClick={() => onOpen(null)}
        >
          Текущий диалог
          <Text span size="xs" c="ink.5" display="block">
            продолжить переписку
          </Text>
        </UnstyledButton>
        {groups.map((g) => (
          <div key={g.title}>
            <Text className={classes.historyGroup}>{g.title}</Text>
            {g.items.map((d) => (
              <UnstyledButton
                key={d.id}
                className={classes.historyItem}
                data-active={openId === d.id || undefined}
                aria-current={openId === d.id ? 'true' : undefined}
                onClick={() => onOpen(d.id)}
              >
                <span className={classes.historyTitle}>{d.title}</span>
                <Text span size="xs" display="block" style={{ color: KIND_COLOR[d.kind] }}>
                  {d.preview}
                </Text>
              </UnstyledButton>
            ))}
          </div>
        ))}
        {dialogs.isSuccess && past.length === 0 && (
          <Text size="xs" c="ink.5" px={10} pt={8}>
            Прошлые диалоги появятся здесь после «Нового диалога».
          </Text>
        )}
      </nav>
    </aside>
  )
}
