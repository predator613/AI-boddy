import { UnstyledButton } from '@mantine/core'

interface Props {
  hidden: number
  open: boolean
  onToggle: () => void
}

/** Строка над перепиской: раскрывает старые карточки «новый материал». */
export function MaterialsFold({ hidden, open, onToggle }: Props) {
  if (hidden === 0) return null
  return (
    <UnstyledButton
      onClick={onToggle}
      aria-expanded={open}
      fz="xs"
      c="ink.5"
      style={{
        alignSelf: 'center',
        padding: '4px 10px',
        border: '1px dashed #D6D6D6',
        borderRadius: 12,
      }}
    >
      {open ? 'Скрыть прошлые материалы' : `Показать прошлые материалы · ${hidden}`}
    </UnstyledButton>
  )
}
