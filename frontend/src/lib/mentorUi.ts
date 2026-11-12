// Подписи и вспомогательные функции кабинета наставника.

export const ACTION_LABELS: Record<string, string> = {
  access_request: 'оформит заявку на доступ',
  leave_request: 'оформит отгул',
  ask_buddy: 'ответит на вопрос',
  open_article: 'откроет статью',
  join_chats: 'добавит в чаты команды',
}

export const SWATCHES = [
  { name: 'мятный', hex: '#C9FFBF' },
  { name: 'голубой', hex: '#BADAFB' },
  { name: 'сиреневый', hex: '#DFCDF3' },
  { name: 'розовый', hex: '#F7C1C5' },
]

/** «14:05» для сегодняшних сообщений, «9 нояб., 14:05» для остальных. */
export function timeLabel(iso: string, now = new Date()) {
  const d = new Date(iso)
  const time = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  if (now.toDateString() === d.toDateString()) return time
  return `${d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })}, ${time}`
}

/** Понедельник недели, в которую попадает дата (00:00 по местному времени). */
export function mondayOf(d: Date) {
  const x = new Date(d)
  x.setHours(0, 0, 0, 0)
  x.setDate(x.getDate() - ((x.getDay() + 6) % 7))
  return x
}

export function addDays(d: Date, n: number) {
  const x = new Date(d)
  x.setDate(x.getDate() + n)
  return x
}

/** «1 задача», «3 задачи», «5 задач». */
export function tasksWord(n: number) {
  const a = n % 10
  const b = n % 100
  if (a === 1 && b !== 11) return `${n} задача`
  if (a >= 2 && a <= 4 && (b < 12 || b > 14)) return `${n} задачи`
  return `${n} задач`
}

const WEEK_MS = 7 * 86_400_000

/**
 * Карточки «новый материал» в переписке: показываем три последние за 7 дней,
 * остальные сворачиваются (они все равно есть в справочнике).
 */
export function foldMaterials<
  T extends { id: string; kind: string; at: string; update: { type: string } | null },
>(messages: T[], now = Date.now()): { visible: T[]; hidden: number } {
  const materials = messages.filter((m) => m.kind === 'update' && m.update?.type === 'material')
  const keep = new Set(
    materials
      .filter((m) => now - new Date(m.at).getTime() <= WEEK_MS)
      .slice(-3)
      .map((m) => m.id),
  )
  const visible = messages.filter((m) => !materials.includes(m) || keep.has(m.id))
  return { visible, hidden: messages.length - visible.length }
}
