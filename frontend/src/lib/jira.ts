import type { TRequestItem } from '../schemas/assistant'

// Заявки на доступы и в отделы живут во внешнем сервисе заявок (в демо – имитация Jira Service Desk).
// Отгулы и цели остаются у руководителя.
export const JIRA = 'Jira Service Desk'

const PREFIX = { access: 'ACCESS', other: 'SD' } as const

type Ref = Pick<TRequestItem, 'kind' | 'number' | 'external'>

/** Ключ заявки в Jira: из backend (external.key). null от backend – заявки в Jira нет.
 * Если поля нет совсем (режим без backend), ключ собирается из вида и номера. */
export function jiraKey(r: Ref): string | null {
  if (r.external) return r.external.key
  if (r.external === null) return null
  return r.kind === 'leave' ? null : `${PREFIX[r.kind]}-${r.number}`
}

/** Как показать номер заявки: ключ Jira или внутренний номер. */
export const requestRef = (r: Ref) => jiraKey(r) ?? `№${r.number}`

/** Сообщение наставнику перед отправкой заявки: «свериться с наставником». */
export function mentorCheckText(title: string, fields: { label: string; value: string }[]) {
  const rows = fields.map((f) => `${f.label.toLowerCase()}: ${f.value}`).join('; ')
  return `Хочу подать заявку «${title}»${rows ? ` (${rows})` : ''}. Все верно?`
}
