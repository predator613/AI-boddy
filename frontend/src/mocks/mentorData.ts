// Имитация эндпоинтов кабинета наставника, которых еще нет в backend (docs/api-mentor-console.md):
// переписка стажер – наставник, материалы, расписание, автоматизации и статистика.
// Данные общие для обеих ролей: если стажер напишет наставнику, а потом в этом же окне
// войти наставником, сообщение будет у него во «Входящих».
import { HttpResponse, delay, http } from 'msw'
import { API_URL } from '../api/client'
import type {
  TAutomation,
  TMaterial,
  TMentorMeeting,
  TThread,
  TThreadMessage,
} from '../schemas/mentor'
import { mockToken, mockUsers } from './data'

const wait = () => delay(import.meta.env.MODE === 'test' ? 10 : 250)
const err = (status: number, code: string, message: string) =>
  HttpResponse.json({ error: { code, message } }, { status })

/** С настоящим backend токен не моковый – тогда роль не проверяем (только демо). */
function roleOf(request: Request) {
  const token = request.headers.get('Authorization')?.replace(/^Bearer\s+/i, '')
  return mockUsers.find((u) => mockToken(u.login) === token)?.role ?? null
}
const forbiddenFor = (request: Request, role: 'intern' | 'staff') => {
  const r = roleOf(request)
  if (r === null) return false
  return role === 'intern' ? r === 'intern' : r !== 'intern'
}

export function weekStart(d = new Date()) {
  const x = new Date(d)
  x.setHours(0, 0, 0, 0)
  x.setDate(x.getDate() - ((x.getDay() + 6) % 7))
  return x
}
const at = (day: number, hour: number, min = 0) => {
  const x = weekStart()
  x.setDate(x.getDate() + day)
  x.setHours(hour, min, 0, 0)
  return x.toISOString()
}
const ago = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString()

let seq = 100
const nid = (p: string) => `${p}-${++seq}`

const MENTOR = { id: 'u-igor', name: 'Игорь' }

interface ThreadState extends TThread {
  unread: number
}

function seedThreads(): Record<string, ThreadState> {
  const m = (
    id: string,
    kind: TThreadMessage['kind'],
    author: TThreadMessage['author'],
    authorName: string,
    text: string,
    when: string,
    update: TThreadMessage['update'] = null,
  ): TThreadMessage => ({ id, kind, author, authorName, text, at: when, update })
  return {
    'u-anna': {
      internId: 'u-anna',
      internName: 'Мария',
      mentor: MENTOR,
      stageCode: '02',
      stageTitle: 'Доступы',
      unread: 2,
      messages: [
        m('a1', 'handoff', 'buddy', 'AI-buddy', 'Как проходит код-ревью?', ago(1500)),
        m(
          'a2',
          'text',
          'intern',
          'Мария',
          'Как у вас проходит код-ревью? В справочнике не нашла',
          ago(1499),
        ),
        m(
          'a3',
          'text',
          'mentor',
          'Игорь',
          'Нужен один аппрув от любого бэкендера и зеленый CI. Статью я добавил в справочник',
          ago(1470),
        ),
        m('a4', 'update', 'mentor', 'Игорь', 'Новый материал', ago(1469), {
          type: 'material',
          title: 'Правила код-ревью команды',
          detail: 'этап 03 · AI-buddy теперь тоже отвечает по нему',
          link: '/kb',
          acked: false,
        }),
        m(
          'a5',
          'handoff',
          'buddy',
          'AI-buddy',
          'Какую роль просить в GitLab, Developer или Reporter?',
          ago(40),
        ),
        m(
          'a6',
          'text',
          'intern',
          'Мария',
          'Игорь, для GitLab нужна роль Developer или Maintainer?',
          ago(39),
        ),
        m(
          'a7',
          'text',
          'intern',
          'Мария',
          'И еще: тестовый стенд не согласуют уже 5 дней, это нормально?',
          ago(36),
        ),
      ],
    },
    'u-dima': {
      internId: 'u-dima',
      internName: 'Дмитрий',
      mentor: MENTOR,
      stageCode: '04',
      stageTitle: 'Первые задачи',
      unread: 0,
      messages: [
        m('d1', 'text', 'intern', 'Дмитрий', 'Сделал ревью, посмотришь?', ago(1300)),
        m('d2', 'text', 'mentor', 'Игорь', 'Да, завтра утром посмотрю', ago(1280)),
      ],
    },
    'u-liza': {
      internId: 'u-liza',
      internName: 'Лиза',
      mentor: MENTOR,
      stageCode: '01',
      stageTitle: 'Знакомство с командой',
      unread: 1,
      messages: [
        m('l1', 'handoff', 'buddy', 'AI-buddy', 'Когда будет первая встреча 1:1?', ago(200)),
        m('l2', 'text', 'intern', 'Лиза', 'Привет! Во сколько завтра встреча 1:1?', ago(199)),
      ],
    },
  }
}

function seedMaterials(): TMaterial[] {
  return [
    {
      id: 'mt-1',
      title: 'Памятка стажера: первые две недели',
      section: 'Первые недели стажера',
      kind: 'pdf',
      stageCode: '01',
      useInAi: true,
      authorName: 'Игорь',
      createdAt: '2026-11-01T10:00:00+03:00',
      sizeLabel: '12 стр',
    },
    {
      id: 'mt-2',
      title: 'Как запросить доступ к сервису',
      section: 'Доступы и сервисы',
      kind: 'md',
      stageCode: '02',
      useInAi: true,
      authorName: 'Артемий',
      createdAt: '2026-10-30T10:00:00+03:00',
      sizeLabel: '1 мин',
    },
    {
      id: 'mt-3',
      title: 'Правила код-ревью команды',
      section: 'Код, ревью, релизы',
      kind: 'docx',
      stageCode: '03',
      useInAi: true,
      authorName: 'Игорь',
      createdAt: '2026-10-28T10:00:00+03:00',
      sizeLabel: '4 стр',
    },
    {
      id: 'mt-4',
      title: 'Запись демо команды за октябрь',
      section: 'Процессы команды',
      kind: 'link',
      stageCode: null,
      useInAi: false,
      authorName: 'Артемий',
      createdAt: '2026-10-25T10:00:00+03:00',
      sizeLabel: 'видео',
    },
  ]
}

function seedMeetings(): TMentorMeeting[] {
  return [
    {
      id: 'mm-1',
      title: '1:1 с Марией',
      internId: 'u-anna',
      withName: 'Мария',
      kind: 'one_on_one',
      startsAt: at(3, 15),
      durationMin: 60,
    },
    {
      id: 'mm-2',
      title: 'Разбор ревью',
      internId: 'u-dima',
      withName: 'Дмитрий',
      kind: 'review',
      startsAt: at(0, 11),
      durationMin: 60,
    },
    {
      id: 'mm-3',
      title: '1:1 с Лизой',
      internId: 'u-liza',
      withName: 'Лиза',
      kind: 'one_on_one',
      startsAt: at(1, 12),
      durationMin: 60,
    },
    {
      id: 'mm-4',
      title: 'Демо команды',
      internId: null,
      withName: 'все стажеры',
      kind: 'team',
      startsAt: at(4, 16),
      durationMin: 90,
    },
  ]
}

function seedAutomations(): TAutomation[] {
  return [
    {
      id: 'au-1',
      kind: 'deadline_reminder',
      title: 'Напоминать о сроке задачи',
      note: 'за день до срока, стажеру в чат',
      enabled: true,
    },
    {
      id: 'au-2',
      kind: 'weekly_goal_checkin',
      title: 'Еженедельный чек-ин по целям',
      note: 'пятница 17:00 · AI-buddy спросит о прогрессе и обновит трекер',
      enabled: true,
    },
    {
      id: 'au-3',
      kind: 'overdue_alert',
      title: 'Сообщать мне о просрочке',
      note: 'если задача просрочена больше 2 дней',
      enabled: true,
    },
    {
      id: 'au-4',
      kind: 'approver_reminder',
      title: 'Напоминать согласующему о заявке',
      note: 'если доступ висит дольше 3 дней',
      enabled: false,
    },
  ]
}

let threads = seedThreads()
let materials = seedMaterials()
let meetings = seedMeetings()
let automations = seedAutomations()

export function resetMentorMocks() {
  threads = seedThreads()
  materials = seedMaterials()
  meetings = seedMeetings()
  automations = seedAutomations()
}

const publicThread = ({ unread: _unread, ...t }: ThreadState): TThread => t

function pushUpdate(
  internId: string | null,
  update: NonNullable<TThreadMessage['update']>,
  text: string,
) {
  const ids = internId ? [internId] : Object.keys(threads)
  for (const id of ids) {
    const t = threads[id]
    if (!t) continue
    t.messages = [
      ...t.messages,
      {
        id: nid('upd'),
        kind: 'update',
        author: 'mentor',
        authorName: MENTOR.name,
        text,
        at: new Date().toISOString(),
        update: { ...update },
      },
    ]
  }
}

const KIND_TITLE: Record<TMentorMeeting['kind'], string> = {
  one_on_one: '1:1',
  review: 'разбор',
  team: 'команда',
}
const fmtWhen = (iso: string) =>
  new Date(iso).toLocaleString('ru-RU', {
    weekday: 'short',
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  })

export const mentorHandlers = [
  // ---------- статистика ----------
  http.get(`${API_URL}/api/mentor/stats`, ({ request }) => {
    if (forbiddenFor(request, 'intern')) return err(403, 'forbidden', 'forbidden')
    return HttpResponse.json({
      periodDays: 7,
      answeredByBuddy: 23,
      handedOff: Object.values(threads).filter((t) => t.unread > 0).length + 1,
      savedMinutes: 240,
      mentorHoursSpent: 9,
      mentorHoursPlan: 44,
    })
  }),

  // ---------- переписка: наставник ----------
  http.get(`${API_URL}/api/mentor/threads`, ({ request }) => {
    if (forbiddenFor(request, 'intern')) return err(403, 'forbidden', 'forbidden')
    return HttpResponse.json(
      Object.values(threads)
        .map((t) => {
          const last = [...t.messages].reverse().find((x) => x.kind !== 'handoff') ?? t.messages[0]
          return {
            internId: t.internId,
            internName: t.internName,
            stageCode: t.stageCode,
            stageTitle: t.stageTitle,
            lastText:
              last?.kind === 'update' ? `${last.text}: ${last.update?.title}` : (last?.text ?? ''),
            lastAt: last?.at ?? '',
            unread: t.unread,
          }
        })
        .sort((a, b) => b.lastAt.localeCompare(a.lastAt)),
    )
  }),

  http.get(`${API_URL}/api/mentor/threads/:internId`, ({ request, params }) => {
    if (forbiddenFor(request, 'intern')) return err(403, 'forbidden', 'forbidden')
    const t = threads[String(params.internId)]
    return t ? HttpResponse.json(publicThread(t)) : err(404, 'not_found', 'not found')
  }),

  http.post(`${API_URL}/api/mentor/threads/:internId/read`, ({ params }) => {
    const t = threads[String(params.internId)]
    if (!t) return err(404, 'not_found', 'not found')
    t.unread = 0
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API_URL}/api/mentor/threads/:internId/messages`, async ({ request, params }) => {
    if (forbiddenFor(request, 'intern')) return err(403, 'forbidden', 'forbidden')
    const t = threads[String(params.internId)]
    if (!t) return err(404, 'not_found', 'not found')
    const body = (await request.json().catch(() => null)) as {
      text?: string
      addToKb?: boolean
    } | null
    const text = body?.text?.trim()
    if (!text) return err(400, 'validation_error', 'text is required')
    await wait()
    t.messages = [
      ...t.messages,
      {
        id: nid('msg'),
        kind: 'text',
        author: 'mentor',
        authorName: MENTOR.name,
        text,
        at: new Date().toISOString(),
        update: null,
      },
    ]
    t.unread = 0
    if (body?.addToKb) {
      materials = [
        {
          id: nid('mt'),
          title: `Ответ наставника: ${text.slice(0, 60)}`,
          kind: 'answer',
          stageCode: t.stageCode,
          section: 'Ответы наставника',
          useInAi: true,
          authorName: MENTOR.name,
          createdAt: new Date().toISOString(),
          sizeLabel: 'из переписки',
        },
        ...materials,
      ]
    }
    return HttpResponse.json(publicThread(t))
  }),

  // ---------- переписка: стажер ----------
  http.get(`${API_URL}/api/me/mentor-thread`, ({ request }) => {
    if (forbiddenFor(request, 'staff')) return err(403, 'forbidden', 'only interns')
    return HttpResponse.json(publicThread(threads['u-anna']!))
  }),

  http.post(`${API_URL}/api/me/mentor-thread/messages`, async ({ request }) => {
    if (forbiddenFor(request, 'staff')) return err(403, 'forbidden', 'only interns')
    const body = (await request.json().catch(() => null)) as { text?: string } | null
    const text = body?.text?.trim()
    if (!text) return err(400, 'validation_error', 'text is required')
    await wait()
    const t = threads['u-anna']!
    t.messages = [
      ...t.messages,
      {
        id: nid('msg'),
        kind: 'text',
        author: 'intern',
        authorName: t.internName,
        text,
        at: new Date().toISOString(),
        update: null,
      },
    ]
    t.unread += 1
    return HttpResponse.json(publicThread(t))
  }),

  http.post(`${API_URL}/api/me/mentor-thread/updates/:messageId/ack`, ({ params }) => {
    const t = threads['u-anna']!
    const msg = t.messages.find((x) => x.id === params.messageId && x.update)
    if (!msg?.update) return err(404, 'not_found', 'not found')
    msg.update = { ...msg.update, acked: true }
    t.messages = [...t.messages]
    return HttpResponse.json(publicThread(t))
  }),

  // ---------- материалы ----------
  http.get(`${API_URL}/api/materials`, () => HttpResponse.json(materials)),

  http.post(`${API_URL}/api/materials`, async ({ request }) => {
    if (forbiddenFor(request, 'intern')) return err(403, 'forbidden', 'forbidden')
    const form = await request.formData().catch(() => null)
    const title = String(form?.get('title') ?? '').trim()
    const file = form?.get('file')
    const url = String(form?.get('url') ?? '').trim()
    if (!title) return err(400, 'validation_error', 'title is required')
    if (!(file instanceof File) && !url)
      return err(400, 'validation_error', 'file or url is required')
    await wait()
    const name = file instanceof File ? file.name.toLowerCase() : ''
    const kind: TMaterial['kind'] = !name
      ? 'link'
      : name.endsWith('.pdf')
        ? 'pdf'
        : name.endsWith('.docx')
          ? 'docx'
          : 'md'
    const item: TMaterial = {
      id: nid('mt'),
      title,
      kind,
      stageCode: String(form?.get('stageCode') ?? '') || null,
      section: String(form?.get('section') ?? '').trim() || null,
      useInAi: form?.get('useInAi') !== 'false',
      authorName: MENTOR.name,
      createdAt: new Date().toISOString(),
      sizeLabel:
        file instanceof File ? `${Math.max(1, Math.round(file.size / 1024))} КБ` : 'ссылка',
    }
    materials = [item, ...materials]
    if (form?.get('notify') !== 'false') {
      pushUpdate(
        null,
        {
          type: 'material',
          title,
          detail: item.stageCode ? `этап ${item.stageCode}` : 'для всех этапов',
          link: '/kb',
          acked: false,
        },
        'Новый материал',
      )
    }
    return HttpResponse.json(item, { status: 201 })
  }),

  http.delete(`${API_URL}/api/materials/:id`, ({ params }) => {
    const before = materials.length
    materials = materials.filter((m) => m.id !== params.id)
    return materials.length === before
      ? err(404, 'not_found', 'not found')
      : new HttpResponse(null, { status: 204 })
  }),

  // ---------- расписание ----------
  http.get(`${API_URL}/api/mentor/meetings`, ({ request }) => {
    if (forbiddenFor(request, 'intern')) return err(403, 'forbidden', 'forbidden')
    const u = new URL(request.url)
    const from = u.searchParams.get('from') ?? ''
    const to = u.searchParams.get('to') ?? '9999'
    return HttpResponse.json(
      meetings
        .filter((m) => m.startsAt >= from && m.startsAt < to)
        .sort((a, b) => a.startsAt.localeCompare(b.startsAt)),
    )
  }),

  http.post(`${API_URL}/api/mentor/meetings`, async ({ request }) => {
    if (forbiddenFor(request, 'intern')) return err(403, 'forbidden', 'forbidden')
    const b = (await request.json().catch(() => null)) as {
      internId?: string
      title?: string
      startsAt?: string
      durationMin?: number
      notify?: boolean
    } | null
    if (!b?.internId || !b.title?.trim() || !b.startsAt)
      return err(400, 'validation_error', 'internId, title and startsAt are required')
    await wait()
    const all = b.internId === 'all'
    const who = all ? 'все стажеры' : (threads[b.internId]?.internName ?? 'стажер')
    const item: TMentorMeeting = {
      id: nid('mm'),
      title: b.title.trim(),
      internId: all ? null : b.internId,
      withName: who,
      kind: all ? 'team' : 'one_on_one',
      startsAt: new Date(b.startsAt).toISOString(),
      durationMin: b.durationMin ?? 60,
    }
    meetings = [...meetings, item]
    if (b.notify !== false) {
      pushUpdate(
        item.internId,
        {
          type: 'meeting',
          title: item.title,
          detail: `${fmtWhen(item.startsAt)} · ${KIND_TITLE[item.kind]} · напомню за час`,
          link: null,
          acked: false,
        },
        'Встреча',
      )
    }
    return HttpResponse.json(item, { status: 201 })
  }),

  http.patch(`${API_URL}/api/mentor/meetings/:id`, async ({ request, params }) => {
    const m = meetings.find((x) => x.id === params.id)
    if (!m) return err(404, 'not_found', 'not found')
    const b = (await request.json().catch(() => null)) as { startsAt?: string } | null
    if (!b?.startsAt) return err(400, 'validation_error', 'startsAt is required')
    m.startsAt = new Date(b.startsAt).toISOString()
    meetings = [...meetings]
    pushUpdate(
      m.internId,
      {
        type: 'meeting',
        title: `${m.title} перенесена`,
        detail: fmtWhen(m.startsAt),
        link: null,
        acked: false,
      },
      'Встреча',
    )
    return HttpResponse.json(m)
  }),

  http.delete(`${API_URL}/api/mentor/meetings/:id`, ({ params }) => {
    const m = meetings.find((x) => x.id === params.id)
    if (!m) return err(404, 'not_found', 'not found')
    meetings = meetings.filter((x) => x.id !== m.id)
    pushUpdate(
      m.internId,
      {
        type: 'meeting',
        title: `${m.title} отменена`,
        detail: fmtWhen(m.startsAt),
        link: null,
        acked: false,
      },
      'Встреча',
    )
    return new HttpResponse(null, { status: 204 })
  }),

  // ---------- автоматизации ----------
  http.get(`${API_URL}/api/mentor/automations`, () => HttpResponse.json(automations)),

  http.patch(`${API_URL}/api/mentor/automations/:id`, async ({ request, params }) => {
    const a = automations.find((x) => x.id === params.id)
    if (!a) return err(404, 'not_found', 'not found')
    const b = (await request.json().catch(() => null)) as { enabled?: boolean } | null
    if (typeof b?.enabled !== 'boolean') return err(400, 'validation_error', 'enabled is required')
    automations = automations.map((x) => (x.id === a.id ? { ...x, enabled: b.enabled! } : x))
    return HttpResponse.json(automations.find((x) => x.id === a.id))
  }),
]

// ---------- календарь стажера (docs/api-intern-calendar.md) ----------
// Встречи берутся из расписания наставника (стажер Мария и общие), плюс прошедшие знакомства
// и собственные записи стажера.

type CalEvent = import('../schemas/calendar').TCalendarEvent
type Answer = 'accept' | 'move'

function seedOwnCalendar(): { own: CalEvent[]; answers: Record<string, Answer> } {
  return { own: [], answers: {} }
}
let cal = seedOwnCalendar()
export function resetCalendarMock() {
  cal = seedOwnCalendar()
}

// Даты считаются при каждом запросе, а не при загрузке модуля: тесты подменяют текущую дату.
const fixedEvents = (): CalEvent[] => [
  {
    id: 'ce-intro',
    title: 'Встреча-знакомство с командой',
    kind: 'team',
    startsAt: at(0, 11),
    durationMin: 60,
    withName: 'команда Backend',
    place: 'переговорка «Байкал»',
    note: 'Записала имена и роли команды – они в справочнике, раздел «Кто за что отвечает».',
    status: 'accepted',
    shared: false,
  },
  {
    id: 'ce-lead',
    title: 'Цели стажировки с Артемием',
    kind: 'lead',
    startsAt: at(1, 12),
    durationMin: 30,
    withName: 'Артемий, руководитель',
    place: 'видеозвонок',
    note: 'Согласовали 3 цели от OKR команды. Они уже в «Моем пути».',
    status: 'accepted',
    shared: false,
  },
  {
    id: 'ce-devops',
    title: 'Знакомство с DevOps',
    kind: 'intro',
    startsAt: at(2, 14),
    durationMin: 30,
    withName: 'Кирилл, DevOps',
    place: 'видеозвонок',
    note: 'Спроси про тестовый стенд: заявка ждет его согласования.',
    status: 'accepted',
    shared: false,
  },
]

const NOTES: Record<string, string> = {
  one_on_one: 'Темы AI-buddy соберет из вопросов, которые ты задавала за неделю.',
  review: 'Подготовь вопросы по замечаниям из ревью.',
  team: 'Посмотришь, что команда выпустила. Можно просто слушать.',
}

function calendarEvents(): CalEvent[] {
  const fromMentor: CalEvent[] = meetings
    .filter((m) => m.internId === 'u-anna' || m.internId === null)
    .map((m) => {
      const a = cal.answers[m.id]
      return {
        id: m.id,
        title: m.internId ? m.title.replace('с Марией', 'с Игорем') : m.title,
        kind: m.kind === 'team' ? 'team' : 'mentor',
        startsAt: m.startsAt,
        durationMin: m.durationMin,
        withName: m.internId ? 'Игорь, наставник' : 'вся команда и стажеры',
        place: m.kind === 'team' ? 'большая переговорка' : 'видеозвонок',
        note: NOTES[m.kind] ?? '',
        status: a === 'accept' ? 'accepted' : a === 'move' ? 'move_requested' : 'pending',
        shared: false,
      }
    })
  return [...fixedEvents(), ...fromMentor, ...cal.own]
}

export const calendarHandlers = [
  http.get(`${API_URL}/api/me/calendar`, ({ request }) => {
    if (forbiddenFor(request, 'staff')) return err(403, 'forbidden', 'only interns')
    const u = new URL(request.url)
    const from = u.searchParams.get('from') ?? ''
    const to = u.searchParams.get('to') ?? '9999'
    return HttpResponse.json({
      events: calendarEvents()
        .filter((e) => e.startsAt >= from && e.startsAt < to)
        .sort((a, b) => a.startsAt.localeCompare(b.startsAt)),
    })
  }),

  http.post(`${API_URL}/api/me/calendar/entries`, async ({ request }) => {
    const b = (await request.json().catch(() => null)) as {
      title?: string
      kind?: CalEvent['kind']
      startsAt?: string
      durationMin?: number
      shareWithMentor?: boolean
    } | null
    if (!b?.title?.trim() || !b.startsAt || !b.durationMin || !b.kind)
      return err(400, 'validation_error', 'title, kind, startsAt and durationMin are required')
    const start = new Date(b.startsAt).getTime()
    const end = start + b.durationMin * 60_000
    const clash = calendarEvents().find((e) => {
      const s = new Date(e.startsAt).getTime()
      return start < s + e.durationMin * 60_000 && s < end
    })
    if (clash) return err(409, 'conflict', `Пересекается с «${clash.title}»`)
    await wait()
    const item: CalEvent = {
      id: nid('own'),
      title: b.title.trim(),
      kind: b.kind,
      startsAt: new Date(b.startsAt).toISOString(),
      durationMin: b.durationMin,
      withName: 'только ты',
      place: '–',
      note: b.shareWithMentor
        ? 'Наставник видит, что ты занята в это время.'
        : 'Видна только тебе.',
      status: 'own',
      shared: Boolean(b.shareWithMentor),
    }
    cal.own = [...cal.own, item]
    return HttpResponse.json(item, { status: 201 })
  }),

  http.delete(`${API_URL}/api/me/calendar/entries/:id`, ({ params }) => {
    const before = cal.own.length
    cal.own = cal.own.filter((e) => e.id !== params.id)
    return cal.own.length === before
      ? err(404, 'not_found', 'not found')
      : new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API_URL}/api/me/calendar/events/:id/answer`, async ({ request, params }) => {
    const id = String(params.id)
    const m = meetings.find((x) => x.id === id)
    if (!m) return err(404, 'not_found', 'only meetings from the mentor can be answered')
    const b = (await request.json().catch(() => null)) as {
      answer?: Answer
      reason?: string
    } | null
    if (b?.answer !== 'accept' && b?.answer !== 'move')
      return err(400, 'validation_error', 'answer must be accept or move')
    await wait()
    cal.answers = { ...cal.answers, [id]: b.answer }
    if (b.answer === 'move') {
      const t = threads['u-anna']!
      t.messages = [
        ...t.messages,
        {
          id: nid('msg'),
          kind: 'text',
          author: 'intern',
          authorName: t.internName,
          text: `Можно перенести «${m.title.replace('с Марией', 'с тобой')}»? ${b.reason ?? ''}`.trim(),
          at: new Date().toISOString(),
          update: null,
        },
      ]
      t.unread += 1
    }
    return HttpResponse.json(calendarEvents().find((e) => e.id === id))
  }),
]
