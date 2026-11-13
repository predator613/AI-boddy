// Моки ленты изменений наставника и согласований руководителя (docs/api-lead-feed.md).
// Backend их уже реализует (лента, согласования, чек-лист, конспект), поэтому leadHandlers работают
// только в полном режиме без backend и в тестах. В pendingHandlers остались leadPendingHandlers –
// то, чего в backend еще нет (метрики пользы).
import { HttpResponse, delay, http } from 'msw'
import { API_URL } from '../api/client'
import type { TApproval, TChecklistItem, TFeedItem } from '../schemas/staff'
import { mockToken, mockUsers } from './data'

const wait = () => delay(import.meta.env.MODE === 'test' ? 10 : 200)
const ago = (min: number) => new Date(Date.now() - min * 60_000).toISOString()
const err = (status: number, code: string, message: string) =>
  HttpResponse.json({ error: { code, message } }, { status })

function roleOf(request: Request) {
  const token = request.headers.get('Authorization')?.replace(/^Bearer\s+/i, '')
  return mockUsers.find((u) => mockToken(u.login) === token)?.role ?? null
}

const seedFeed = (): TFeedItem[] => [
  {
    id: 'f1',
    actor: { name: 'Олег', kind: 'lead' },
    text: 'Олег согласовал цели Анны',
    at: ago(25),
    read: false,
  },
  {
    id: 'f2',
    actor: { name: 'Анна', kind: 'intern' },
    text: 'Анна приняла приглашение на 1:1 в четверг',
    at: ago(50),
    read: false,
  },
  {
    id: 'f3',
    actor: { name: 'Павел', kind: 'intern' },
    text: 'Павел прошел встречу-знакомство с командой',
    at: ago(60 * 20),
    read: false,
  },
  {
    id: 'f4',
    actor: { name: 'Кирилл', kind: 'other' },
    text: 'Кирилл (DevOps) взял в работу тестовый стенд для Анны',
    at: ago(60 * 22),
    read: true,
  },
  {
    id: 'f5',
    actor: { name: 'AI-buddy', kind: 'buddy' },
    text: 'AI-buddy ответил Павлу по памятке стажера',
    at: ago(60 * 26),
    read: true,
  },
]

const seedApprovals = (): TApproval[] => [
  {
    id: 'ap-1',
    number: '1042',
    internId: 'u-anna',
    internName: 'Анна',
    title: 'Доступ к GitLab',
    fields: [
      { label: 'Проект', value: 'backend/auth-service' },
      { label: 'Уровень прав', value: 'Developer: код, ветки, merge request' },
      { label: 'Зачем', value: 'первая задача, цель «Пройти путь задачи до релиза»' },
      { label: 'Оформил', value: 'AI-buddy, стажер подтвердила' },
      { label: 'Почему ты', value: 'ты владелец проекта, заявка ACCESS-1042 в Jira' },
    ],
    createdAt: ago(90),
    status: 'waiting',
    rejectReason: null,
  },
  {
    id: 'ap-2',
    number: '1039',
    internId: 'u-anna',
    internName: 'Анна',
    title: 'Доступ к тестовому стенду',
    fields: [
      { label: 'Стенд', value: 'test-auth, только чтение' },
      { label: 'Зачем', value: 'проверять изменения до релиза' },
      { label: 'Оформил', value: 'AI-buddy, стажер подтвердила' },
      { label: 'Почему ты', value: 'ты владелец стенда, заявка ACCESS-1039 в Jira' },
    ],
    createdAt: ago(60 * 24 * 5),
    status: 'waiting',
    rejectReason: null,
  },
]

// Чек-лист наставника: пункты из чек-листа стажировки Авито для наставника.
const TEMPLATE: Omit<TChecklistItem, 'done'>[] = [
  {
    id: 'c-chats',
    title: 'Добавить в чаты и каналы',
    hint: 'в том числе закрытые',
    fromWeek: 1,
    period: 'неделя 1',
  },
  {
    id: 'c-access',
    title: 'Проверить, что доступы запрошены',
    hint: 'трекер, вики, GitLab, стенд',
    fromWeek: 1,
    period: 'неделя 1',
  },
  {
    id: 'c-11',
    title: 'Назначить регулярные 1:1',
    hint: 'про команду, инструменты и задачи',
    fromWeek: 1,
    period: 'неделя 1',
  },
  {
    id: 'c-goals',
    title: 'Обсудить цели и критерии успеха',
    hint: 'вместе с руководителем, от OKR команды',
    fromWeek: 1,
    period: 'неделя 1',
  },
  {
    id: 'c-guide',
    title: 'Запросить методичку наставника',
    hint: 'у куратора стажировки',
    fromWeek: 1,
    period: 'неделя 1',
  },
  {
    id: 'c-tracker',
    title: 'Проверить цели в трекере прогресса',
    hint: 'цели, задачи и ожидания внесены',
    fromWeek: 2,
    period: 'неделя 2',
  },
  {
    id: 'c-brief',
    title: 'Сходить на бриф перед экватором',
    hint: 'приглашение придет в календарь',
    fromWeek: 12,
    period: 'перед экватором',
  },
  {
    id: 'c-deck',
    title: 'Проверить презентацию стажера',
    hint: 'задачи и цели отражены верно',
    fromWeek: 12,
    period: 'перед экватором',
  },
  {
    id: 'c-words',
    title: 'Подготовить напутственное слово',
    hint: 'не обратная связь, а благодарность и пожелания',
    fromWeek: 13,
    period: 'экватор',
  },
  {
    id: 'c-feedback',
    title: '1:1 с обратной связью по целям',
    hint: 'промежуточные итоги стажировки',
    fromWeek: 13,
    period: 'после экватора',
  },
]

const seedChecklist = (): Record<string, Set<string>> => ({
  'u-anna': new Set(['c-chats', 'c-access', 'c-11', 'c-goals']),
  'u-pavel': new Set(['c-chats']),
})
let checklist = seedChecklist()

const checklistView = (internId: string) => {
  const done = (checklist[internId] ??= new Set())
  return { internId, items: TEMPLATE.map((t) => ({ ...t, done: done.has(t.id) })) }
}

// Конспекты встреч стажера: у 1:1 в понедельник уже есть заметка.
const seedNotes = () =>
  new Map<string, { eventId: string; text: string; updatedAt: string }>([
    [
      'ce-intro',
      {
        eventId: 'ce-intro',
        text: 'Команда: 6 человек, стендап в 11:00. Вопросы по коду – Игорю, по доступам – Олегу.',
        updatedAt: ago(60 * 48),
      },
    ],
  ])
let notes = seedNotes()

let feed = seedFeed()
let approvals = seedApprovals()

export function resetLeadMocks() {
  notes = seedNotes()
  checklist = seedChecklist()
  feed = seedFeed()
  approvals = seedApprovals()
}

export const leadHandlers = [
  http.get(`${API_URL}/api/me/notes`, async () => {
    await wait()
    return HttpResponse.json([...notes.values()])
  }),

  http.put(`${API_URL}/api/me/notes/:eventId`, async ({ request, params }) => {
    await wait()
    // с настоящим backend токен не из моков: роль неизвестна, запрещаем только явного не-стажера
    const r = roleOf(request)
    if (r && r !== 'intern') return err(403, 'forbidden', 'forbidden')
    const body = (await request.json().catch(() => null)) as { text?: string } | null
    const n = {
      eventId: String(params.eventId),
      text: String(body?.text ?? '').trim(),
      updatedAt: new Date().toISOString(),
    }
    if (n.text) notes.set(n.eventId, n)
    else notes.delete(n.eventId)
    return HttpResponse.json(n)
  }),

  http.get(`${API_URL}/api/mentor/interns/:internId/checklist`, async ({ request, params }) => {
    await wait()
    if (roleOf(request) === 'intern') return err(403, 'forbidden', 'forbidden')
    return HttpResponse.json(checklistView(String(params.internId)))
  }),

  http.post(
    `${API_URL}/api/mentor/interns/:internId/checklist/:itemId`,
    async ({ request, params }) => {
      await wait()
      if (roleOf(request) === 'intern') return err(403, 'forbidden', 'forbidden')
      const id = String(params.itemId)
      if (!TEMPLATE.some((t) => t.id === id)) return err(404, 'not_found', 'not found')
      const body = (await request.json().catch(() => null)) as { done?: boolean } | null
      const set = (checklist[String(params.internId)] ??= new Set())
      if (body?.done === false) set.delete(id)
      else set.add(id)
      return HttpResponse.json(checklistView(String(params.internId)))
    },
  ),

  http.get(`${API_URL}/api/mentor/feed`, async ({ request }) => {
    await wait()
    if (roleOf(request) === 'intern') return err(403, 'forbidden', 'forbidden')
    return HttpResponse.json({ items: feed, unread: feed.filter((f) => !f.read).length })
  }),

  http.post(`${API_URL}/api/mentor/feed/read`, async () => {
    await wait()
    feed = feed.map((f) => ({ ...f, read: true }))
    return new HttpResponse(null, { status: 204 })
  }),

  http.get(`${API_URL}/api/approvals`, async ({ request }) => {
    await wait()
    const r = roleOf(request)
    if (r === 'intern') return err(403, 'forbidden', 'forbidden')
    return HttpResponse.json(approvals)
  }),

  http.post(`${API_URL}/api/approvals/:id`, async ({ request, params }) => {
    await wait()
    if (roleOf(request) === 'intern') return err(403, 'forbidden', 'forbidden')
    const body = (await request.json().catch(() => null)) as {
      decision?: string
      reason?: string
    } | null
    const a = approvals.find((x) => x.id === params.id)
    if (!a) return err(404, 'not_found', 'not found')
    if (a.status !== 'waiting') return err(409, 'conflict', 'Заявка уже рассмотрена')
    if (body?.decision === 'reject' && !body.reason?.trim())
      return err(400, 'validation_error', 'Нужна причина отказа')
    const next: TApproval = {
      ...a,
      status: body?.decision === 'reject' ? 'rejected' : 'approved',
      rejectReason: body?.decision === 'reject' ? body.reason!.trim() : null,
    }
    approvals = approvals.map((x) => (x.id === a.id ? next : x))
    return HttpResponse.json(next)
  }),
]

// Еще нет в backend: перехватывается и при работе с настоящим backend (docs/api-lead-feed.md, раздел 11).
export const leadPendingHandlers = [
  http.get(`${API_URL}/api/mentor/metrics`, async ({ request }) => {
    await wait()
    if (roleOf(request) === 'intern') return err(403, 'forbidden', 'forbidden')
    return HttpResponse.json({
      answeredByBuddy: 14,
      escalated: 3,
      hoursSaved: 2.8,
      minutesPerQuestion: 12,
      accessHours: 6,
      onTimePercent: 82,
    })
  }),
]
