// Имитация чата с помощником и заявок для полного режима без backend (npm run dev:mock) и тестов.
// Backend эти эндпоинты уже реализует (docs/api-chat-requests.md), поэтому вместе с настоящим
// backend перехватываются только pendingHandlers внизу файла – то, чего в backend еще нет.
import { HttpResponse, delay, http } from 'msw'
import { API_URL } from '../api/client'
import type { TChatMessage, TKnowledgeBase, TRequestItem } from '../schemas/assistant'
import { setMockTaskStatus } from './pathData'

type Payload = Record<string, unknown>
interface PendingAction {
  kind: 'access' | 'leave' | 'other'
  title: string
  fields: { label: string; value: string }[]
  taskId: string | null
}

const SERVICES: Record<string, string> = {
  gitlab: 'GitLab',
  test_stand: 'Тестовый стенд',
  tracker: 'Трекер задач',
  wiki: 'Вики команды',
}
const ROLES: Record<string, string> = { developer: 'Developer', read: 'чтение', write: 'запись' }

// В тестах ждем минимально, чтобы статус «думает» успел показаться, но тесты шли быстро.
const wait = (ms: number) => delay(import.meta.env.MODE === 'test' ? 300 : ms)

let seq = 0
const id = (p: string) => `${p}-${++seq}`

function initialRequests(): TRequestItem[] {
  return [
    {
      id: 'req-1038',
      number: '1038',
      kind: 'access',
      title: 'Доступ к тестовому стенду',
      status: 'waiting',
      createdAt: '2026-11-09T11:05:00+03:00',
      via: 'buddy',
      fields: [
        { label: 'Сервис', value: 'Тестовый стенд' },
        { label: 'Роль', value: 'чтение' },
        { label: 'Куда ушла', value: 'Jira Service Desk, команда доступов' },
      ],
      history: [
        { title: 'Создана AI-buddy', at: '9 нояб, 11:05', state: 'done' },
        { title: 'Передана в Jira Service Desk', at: '9 нояб, 11:05', state: 'done' },
        { title: 'Ждет команду доступов', at: 'обычно до 1 рабочего дня', state: 'now' },
      ],
      rejectReason: null,
    },
    {
      id: 'req-1031',
      number: '1031',
      kind: 'leave',
      title: 'Отгул 14 ноября',
      status: 'approved',
      createdAt: '2026-11-06T16:40:00+03:00',
      via: 'buddy',
      fields: [
        { label: 'Дата', value: '14 ноября, пятница' },
        { label: 'Тип', value: 'отгул' },
        { label: 'Согласовал', value: 'Артемий, руководитель' },
      ],
      history: [
        { title: 'Создана AI-buddy', at: '6 нояб, 16:40', state: 'done' },
        { title: 'Одобрена Артемием', at: '7 нояб, 10:12', state: 'done' },
      ],
      rejectReason: null,
    },
    {
      id: 'req-1027',
      number: '1027',
      kind: 'access',
      title: 'Доступ к логам продакшена',
      status: 'rejected',
      createdAt: '2026-11-04T15:30:00+03:00',
      via: 'buddy',
      fields: [
        { label: 'Сервис', value: 'Логи продакшена' },
        { label: 'Роль', value: 'чтение' },
        { label: 'Решение', value: 'Артемий, руководитель' },
      ],
      history: [
        { title: 'Создана AI-buddy', at: '4 нояб, 15:30', state: 'done' },
        { title: 'Отклонена', at: '4 нояб, 17:02', state: 'rejected' },
      ],
      rejectReason: 'Стажерам этот доступ не выдается. Логи можно смотреть вместе с наставником.',
    },
    {
      id: 'req-1025',
      number: '1025',
      kind: 'access',
      title: 'Доступ к трекеру задач',
      status: 'done',
      createdAt: '2026-11-03T09:00:00+03:00',
      via: 'auto',
      fields: [
        { label: 'Сервис', value: 'Трекер задач' },
        { label: 'Роль', value: 'чтение и запись' },
      ],
      history: [
        { title: 'Создана при выходе на стажировку', at: '3 нояб, 09:00', state: 'done' },
        { title: 'Доступ выдан', at: '3 нояб, 09:02', state: 'done' },
      ],
      rejectReason: null,
    },
  ]
}

let requests = initialRequests()
let actions = new Map<string, PendingAction>()
let nextNumber = 1042

// Переписка, как ее хранит backend: GET /api/chat/messages возвращает ее после перезагрузки.
let chatLog: TChatMessage[] = []

// Прошлые диалоги (docs/api-chat-dialogs.md): «Новый диалог» переносит переписку сюда.
interface PastDialog {
  id: string
  title: string
  preview: string
  kind: 'request' | 'handoff' | 'answer'
  updatedAt: string
  messages: TChatMessage[]
}
const daysAgo = (n: number) => new Date(Date.now() - n * 86_400_000).toISOString()
function seedDialogs(): PastDialog[] {
  const m = (id: string, role: 'user' | 'assistant', text: string): TChatMessage => ({
    id,
    role,
    kind: 'text',
    text,
    source: null,
    confirm: null,
    result: null,
    mood: role === 'assistant' ? 'calm' : 'calm',
  })
  return [
    {
      id: 'dlg-review',
      title: 'Как проходит код-ревью',
      preview: 'Передано наставнику',
      kind: 'handoff',
      updatedAt: daysAgo(1),
      messages: [
        m('dr-1', 'user', 'Как проходит код-ревью?'),
        m(
          'dr-2',
          'assistant',
          'В справочнике нет ответа для твоей команды – передал вопрос Игорю.',
        ),
      ],
    },
    {
      id: 'dlg-leave',
      title: 'Отгул на 14 ноября',
      preview: 'Заявка одобрена',
      kind: 'request',
      updatedAt: daysAgo(3),
      messages: [
        m('dl-1', 'user', 'Хочу взять отгул 14 ноября'),
        m('dl-2', 'assistant', 'Заявка на отгул создана, Артемий ее одобрил.'),
      ],
    },
  ]
}
let dialogs = seedDialogs()

export function resetPendingMocks() {
  chatLog = []
  dialogs = seedDialogs()
  requests = initialRequests()
  actions = new Map()
  nextNumber = 1042
  seq = 0
}

function msg(m: Partial<TChatMessage> & Pick<TChatMessage, 'text'>): TChatMessage {
  return {
    id: id('m'),
    role: 'assistant',
    kind: 'text',
    source: null,
    confirm: null,
    result: null,
    mood: 'calm',
    ...m,
  }
}

function accessConfirm(payload: Payload, taskId: string | null): TChatMessage[] {
  const service = SERVICES[String(payload.service)] ?? String(payload.service ?? 'сервис')
  const fields = [{ label: 'Сервис', value: service }]
  if (payload.group) fields.push({ label: 'Группа', value: String(payload.group) })
  fields.push({
    label: 'Роль',
    value: ROLES[String(payload.role)] ?? String(payload.role ?? 'по умолчанию'),
  })
  fields.push({ label: 'Согласует', value: 'Артемий, руководитель' })
  const actionId = id('act')
  actions.set(actionId, { kind: 'access', title: `Доступ к ${service}`, fields, taskId })
  return [
    msg({
      text: 'Доступ выдает руководитель команды через сервис доступов. Обычно заявку согласуют за один рабочий день.',
      source: { title: 'Памятка стажера → Доступы', articleId: 'a-access' },
    }),
    msg({
      kind: 'confirm',
      mood: 'mentor',
      text: 'Создать заявку на доступ?',
      confirm: { actionId, title: 'Создать заявку на доступ?', fields, state: 'pending' },
    }),
  ]
}

function leaveConfirm(): TChatMessage[] {
  const fields = [
    { label: 'Дата', value: '14 ноября, пятница' },
    { label: 'Тип', value: 'отгул' },
    { label: 'Согласует', value: 'Артемий, руководитель' },
  ]
  const actionId = id('act')
  actions.set(actionId, { kind: 'leave', title: 'Отгул 14 ноября', fields, taskId: null })
  return [
    msg({
      text: 'Отгул оформляется заявкой руководителю минимум за два рабочих дня. Наставника я предупрежу сам.',
      source: { title: 'Отпуска и отгулы → Как взять отгул', articleId: 'a-leave' },
    }),
    msg({
      kind: 'confirm',
      mood: 'mentor',
      text: 'Создать заявку на отгул?',
      confirm: { actionId, title: 'Создать заявку на отгул?', fields, state: 'pending' },
    }),
  ]
}

// Задача чек-листа «Добавиться в чаты и каналы»: действие через имитацию корпоративного мессенджера.
function joinChatsConfirm(taskId: string | null): TChatMessage[] {
  const fields = [
    { label: 'Чаты', value: 'Команда Backend, Стажеры Авито' },
    { label: 'Каналы', value: 'Дежурства Backend, Релизы' },
    { label: 'Закрытые', value: 'подтвердит наставник' },
  ]
  const actionId = id('act')
  actions.set(actionId, { kind: 'other', title: 'Чаты и каналы команды', fields, taskId })
  return [
    msg({
      kind: 'confirm',
      mood: 'mentor',
      text: 'Добавить тебя в чаты и каналы команды?',
      confirm: { actionId, title: 'Добавить в чаты и каналы?', fields, state: 'pending' },
    }),
  ]
}

export function agentReply(
  text: string,
  action: { kind: string; payload: Payload } | null,
  taskId: string | null = null,
): TChatMessage[] {
  const t = text.toLowerCase()
  if (action?.kind === 'access_request') return accessConfirm(action.payload, taskId)
  if (action?.kind === 'leave_request' || /отгул/.test(t)) return leaveConfirm()
  if (action?.kind === 'join_chats' || /(чат|канал).*(команд|добав)|добав.*(чат|канал)/.test(t))
    return joinChatsConfirm(taskId ?? 't-104')
  if (action?.kind === 'ask_buddy' && typeof action.payload.question === 'string')
    return agentReply(action.payload.question, null)
  if (/защит|раскрыти|коммерческ/.test(t)) {
    return [
      msg({
        text: 'Коротко: не выносим данные пользователей и коллег за пределы рабочих сервисов, не пересылаем коммерческую информацию в личные чаты, о внутреннем говорим публично только после согласования. Проверь себя: можно ли скинуть скриншот дашборда в чат с друзьями? Ответь, и я отмечу задачу.',
        source: { title: 'Безопасность и учеба → Защита данных', articleId: 'a-security' },
      }),
    ]
  }
  if (/экватор|презентац/.test(t)) {
    return [
      msg({
        text: 'Черновик к экватору: 1) кто я и команда; 2) цели и прогресс по шагам; 3) закрытые задачи: доступы, окружение, первая задача; 4) чему научилась; 5) планы до конца стажировки. Отправить черновик Игорю на проверку?',
        source: { title: 'Экватор → Структура презентации', articleId: 'a-equator' },
      }),
    ]
  }
  if (/прод|логи/.test(t)) {
    return [
      msg({
        kind: 'error',
        mood: 'sad',
        text: 'Доступ к логам продакшена стажерам не выдается. Посмотреть логи можно вместе с наставником: позвать Игоря?',
      }),
    ]
  }
  if (/gitlab|гитлаб/.test(t))
    return accessConfirm({ service: 'gitlab', group: 'backend', role: 'developer' }, 't-204')
  if (/согласован|ждать/.test(t)) {
    return [
      msg({
        text: 'Обычно доступ согласуют за один рабочий день. Если дольше, я сам напомню Артемию.',
        source: { title: 'Доступы и сервисы → Сколько ждать согласования', articleId: 'a-wait' },
      }),
    ]
  }
  if (/наставник|игор/.test(t)) {
    return [
      msg({
        mood: 'mentor',
        text: 'Твой наставник – Игорь. Ближайшая встреча 1:1 в четверг в 15:00. Могу передать ему вопрос прямо сейчас.',
      }),
    ]
  }
  if (/ревью/.test(t)) {
    return [
      msg({
        text: 'Каждое изменение смотрит хотя бы один коллега. Ревьюер оставляет комментарии, ты исправляешь, после одобрения код вливается в основную ветку.',
        source: { title: 'Код, ревью и релизы → Как устроено код-ревью', articleId: 'a-review' },
      }),
    ]
  }
  if (/шаг|дальше|что делать/.test(t)) {
    return [
      msg({
        text: 'Следующий шаг – «Получить доступ к GitLab». Подготовить заявку?',
        source: { title: 'Мой путь → этап 02', articleId: null },
      }),
    ]
  }
  return [
    msg({
      mood: 'mentor',
      text: 'Точного ответа в справочнике нет. Передать вопрос Игорю? Он ответит сегодня.',
    }),
  ]
}

export const KB: TKnowledgeBase = {
  sections: [
    {
      id: 's-start',
      title: 'Первые недели стажера',
      hand: 'с чего начать',
      color: '#C9FFBF',
      count: 3,
    },
    { id: 's-access', title: 'Доступы и сервисы', hand: 'доступы', color: '#BADAFB', count: 3 },
    {
      id: 's-process',
      title: 'Процессы команды',
      hand: 'как мы работаем',
      color: '#DFCDF3',
      count: 2,
    },
    { id: 's-code', title: 'Код, ревью и релизы', hand: 'инструменты', color: '#FFFFFF', count: 2 },
    { id: 's-leave', title: 'Отпуска и отгулы', hand: 'отдых', color: '#F7C1C5', count: 1 },
    { id: 's-people', title: 'Кто за что отвечает', hand: 'люди', color: '#FFFFFF', count: 1 },
  ],
  articles: [
    {
      id: 'a-access',
      sectionId: 's-access',
      title: 'Как запросить доступ к сервису',
      summary: 'Заявку согласует руководитель команды, AI-buddy может подготовить ее за тебя.',
      minutes: 3,
      stageCode: '02',
    },
    {
      id: 'a-wait',
      sectionId: 's-access',
      title: 'Сколько ждать согласования',
      summary: 'Обычно один рабочий день. Если дольше – напомни руководителю.',
      minutes: 1,
      stageCode: '02',
    },
    {
      id: 'a-services',
      sectionId: 's-access',
      title: 'Какие доступы нужны стажеру',
      summary: 'Трекер, вики, тестовый стенд и GitLab. Продакшен стажерам не открывают.',
      minutes: 2,
      stageCode: '02',
    },
    {
      id: 'a-memo',
      sectionId: 's-start',
      title: 'Памятка стажера: первые две недели',
      summary: 'Встречи, доступы, первые задачи и к кому идти с вопросами.',
      minutes: 5,
      stageCode: '01',
    },
    {
      id: 'a-1on1',
      sectionId: 's-start',
      title: 'Как проходят встречи 1:1',
      summary: 'Раз в неделю с наставником: что получилось, что мешает, план.',
      minutes: 2,
      stageCode: '01',
    },
    {
      id: 'a-first',
      sectionId: 's-start',
      title: 'Первая задача: от тикета до релиза',
      summary: 'Маленькая задача, чтобы пройти весь цикл разработки.',
      minutes: 4,
      stageCode: '04',
    },
    {
      id: 'a-tasks',
      sectionId: 's-process',
      title: 'Как оформлять задачи',
      summary: 'Заголовок, контекст, критерии готовности.',
      minutes: 3,
      stageCode: '04',
    },
    {
      id: 'a-duty',
      sectionId: 's-process',
      title: 'Дежурства и уведомления',
      summary: 'Кто дежурит и куда приходят алерты.',
      minutes: 2,
      stageCode: '03',
    },
    {
      id: 'a-review',
      sectionId: 's-code',
      title: 'Как устроено код-ревью в команде',
      summary: 'Кто ревьюит, сколько ждать, как отвечать на комментарии.',
      minutes: 4,
      stageCode: '03',
    },
    {
      id: 'a-ci',
      sectionId: 's-code',
      title: 'CI/CD: как код попадает в прод',
      summary: 'Пайплайн, тесты, выкладка с feature flags.',
      minutes: 5,
      stageCode: '03',
    },
    {
      id: 'a-leave',
      sectionId: 's-leave',
      title: 'Как взять отгул',
      summary: 'Заявка руководителю минимум за два рабочих дня.',
      minutes: 2,
      stageCode: null,
    },
    {
      id: 'a-who',
      sectionId: 's-people',
      title: 'Кто за что отвечает в команде',
      summary: 'Руководитель, наставник, дежурные и владельцы сервисов.',
      minutes: 2,
      stageCode: '01',
    },
  ],
  terms: [
    {
      term: 'Код-ревью',
      def: 'Проверка кода коллегой перед тем, как он попадет в общий репозиторий.',
    },
    { term: 'Релиз', def: 'Выкладка изменений пользователям.' },
    {
      term: 'Feature flag',
      def: 'Переключатель, который включает новую функцию без нового релиза.',
    },
    { term: '1:1', def: 'Регулярная встреча с наставником один на один.' },
  ],
}

const reply = (messages: TChatMessage[]) => {
  chatLog.push(...messages)
  return HttpResponse.json({ messages })
}
const markCard = (actionId: string, state: 'confirmed' | 'cancelled') => {
  chatLog = chatLog.map((m) =>
    m.confirm?.actionId === actionId ? { ...m, confirm: { ...m.confirm, state } } : m,
  )
}
const notFound = () =>
  HttpResponse.json({ error: { code: 'not_found', message: 'not found' } }, { status: 404 })

export const assistantHandlers = [
  http.post(`${API_URL}/api/chat/messages`, async ({ request }) => {
    const body = (await request.json().catch(() => null)) as {
      text?: string
      action?: { kind: string; payload: Payload } | null
      taskId?: string
    } | null
    if (!body?.text?.trim()) {
      return HttpResponse.json(
        { error: { code: 'validation_error', message: 'text is required' } },
        { status: 400 },
      )
    }
    chatLog.push(msg({ role: 'user', text: body.text.trim() }))
    await wait(900)
    const taskId =
      body.action?.kind === 'access_request' && body.action.payload.service === 'gitlab'
        ? 't-204'
        : null
    return reply(agentReply(body.text, body.action ?? null, taskId))
  }),

  http.post(`${API_URL}/api/chat/actions/:actionId/confirm`, async ({ params }) => {
    const a = actions.get(String(params.actionId))
    if (!a) return notFound()
    actions.delete(String(params.actionId))
    markCard(String(params.actionId), 'confirmed')
    await wait(600)
    const number = String(nextNumber++)
    const item: TRequestItem = {
      id: `req-${number}`,
      number,
      kind: a.kind,
      title: a.title,
      status: a.kind === 'other' ? 'done' : 'waiting',
      createdAt: new Date().toISOString(),
      via: 'buddy',
      fields: a.fields,
      history:
        a.kind === 'other'
          ? [{ title: 'Выполнено AI-buddy', at: 'только что', state: 'done' }]
          : [
              { title: 'Создана AI-buddy', at: 'только что', state: 'done' },
              { title: 'Ждет решения Артемия', at: 'обычно до 1 рабочего дня', state: 'now' },
            ],
      rejectReason: null,
    }
    requests = [item, ...requests]
    if (a.taskId) setMockTaskStatus(a.taskId, a.kind === 'other' ? 'done' : 'waiting')
    return reply([
      msg({
        kind: 'result',
        mood: 'happy',
        text: a.kind === 'other' ? 'Готово: ты в чатах команды' : `Заявка №${number} создана`,
        result: { requestId: item.id, number, title: a.title, status: item.status },
      }),
    ])
  }),

  http.post(`${API_URL}/api/chat/actions/:actionId/cancel`, ({ params }) => {
    if (!actions.delete(String(params.actionId))) return notFound()
    markCard(String(params.actionId), 'cancelled')
    return reply([msg({ text: 'Хорошо, не создаю. Если передумаешь – просто напиши.' })])
  }),

  http.get(`${API_URL}/api/requests`, () => HttpResponse.json(requests)),

  http.get(`${API_URL}/api/chat/messages`, () => HttpResponse.json({ messages: chatLog })),

  http.get(`${API_URL}/api/chat/dialogs`, () =>
    HttpResponse.json([
      ...(chatLog.length
        ? [
            {
              id: 'current',
              title: chatLog.find((x) => x.role === 'user')?.text.slice(0, 60) ?? 'Текущий диалог',
              preview: 'идет сейчас',
              kind: 'answer',
              updatedAt: new Date().toISOString(),
              current: true,
            },
          ]
        : []),
      ...dialogs.map(({ messages: _m, ...d }) => ({ ...d, current: false })),
    ]),
  ),

  http.get(`${API_URL}/api/chat/dialogs/:id/messages`, ({ params }) => {
    const d = dialogs.find((x) => x.id === params.id)
    return d
      ? HttpResponse.json({ messages: d.messages })
      : HttpResponse.json({ error: { code: 'not_found', message: 'not found' } }, { status: 404 })
  }),

  http.post(`${API_URL}/api/chat/reset`, () => {
    if (chatLog.length) {
      const first = chatLog.find((x) => x.role === 'user')?.text ?? 'Диалог'
      const req = chatLog.some((x) => x.result)
      dialogs = [
        {
          id: `dlg-${Date.now()}`,
          title: first.slice(0, 60),
          preview: req ? 'Заявка создана' : 'Ответ из справочника',
          kind: req ? 'request' : 'answer',
          updatedAt: new Date().toISOString(),
          messages: chatLog,
        },
        ...dialogs,
      ]
    }
    chatLog = []
    actions = new Map()
    return HttpResponse.json({ messages: [] })
  }),
  // GET /api/kb уже отдает backend: обработчик перенесен в handlers.ts (только полный режим без backend).
]

// Эндпоинты, которых еще нет в backend: MSW перехватывает их и при работе с настоящим backend
// (main.tsx, VITE_PENDING_MOCKS). Сейчас это кабинет наставника и чат стажера с наставником
// (docs/api-mentor-console.md). Когда backend реализует ручку – удалите ее из mentorHandlers.
// Календарь стажера backend уже отдает сам: его моки только в полном режиме (handlers.ts).
// Сейчас backend реализует все, что вызывает фронт (docs/api-backend-status.md), поэтому список пуст.
// Новую ручку, которой еще нет в backend, добавьте сюда – MSW будет перехватывать ее и с настоящим backend.
export const pendingHandlers: typeof assistantHandlers = []
