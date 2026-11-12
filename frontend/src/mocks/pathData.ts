// Синтетический «Мой путь» для режима без backend и для тестов.
// Повторяет seed backend (004_seed_onboarding.sql) и формат GET /api/path.
import type { TPathView } from '../schemas/path'

type Status = 'todo' | 'in_progress' | 'waiting' | 'done'
type ActionT = {
  kind: 'access_request' | 'leave_request' | 'ask_buddy' | 'open_article' | 'join_chats'
  payload: Record<string, unknown>
} | null
interface TaskSeed {
  id: string
  title: string
  note: string
  status: Status
  due: string | null
  action: ActionT
}
interface ResSeed {
  id: string
  kind: 'person' | 'access' | 'tool' | 'article' | 'link'
  name: string
  note: string
  tag: string
  tagKind: 'ok' | 'wait' | 'none' | 'mentor'
  taskId?: string
}
interface StageSeed {
  id: string
  code: string
  title: string
  color: string
  period: string
  tip: string
  suggestions: string[]
  tasks: TaskSeed[]
  resources: ResSeed[]
}

const gitlab: ActionT = {
  kind: 'access_request',
  payload: { service: 'gitlab', group: 'backend', role: 'developer' },
}

// Подробности задач для раскрытия по клику (поле details в GET /api/path).
const DETAILS: Record<
  string,
  { about: string; helper: string; doneWhen: string; goal: string | null }
> = {
  't-101': {
    about:
      'Команда рассказывает, кто чем занимается и как устроена неделя: стендапы, планирование, демо.',
    helper: 'Игорь, наставник',
    doneWhen: 'ты побывала на встрече',
    goal: null,
  },
  't-102': {
    about:
      'Первая встреча 1:1: договоритесь, как и когда задавать вопросы и как часто встречаться.',
    helper: 'Игорь, наставник',
    doneWhen: 'встреча 1:1 прошла',
    goal: null,
  },
  't-103': {
    about: 'Короткая памятка про процессы команды, инструменты и правила первых двух недель.',
    helper: 'AI-buddy ответит на вопросы по памятке',
    doneWhen: 'памятка отмечена прочитанной',
    goal: 'Разобраться в сервисе авторизации',
  },
  't-104': {
    about:
      'AI-buddy добавит тебя в чат команды Backend, чат стажеров и канал дежурств. Закрытые чаты подтверждает наставник.',
    helper: 'AI-buddy, наставник',
    doneWhen: 'ты видишь сообщения команды',
    goal: null,
  },
  't-105': {
    about:
      'Записывай после каждой встречи с наставником: что узнала о компании, договоренности, обратную связь. Заметки лежат в календаре у встречи и видны только тебе.',
    helper: 'Игорь, наставник',
    doneWhen: 'есть заметки хотя бы к одной встрече 1:1',
    goal: null,
  },
  't-501': {
    about:
      'Обязательные материалы: лендинг «Защита данных», памятка о раскрытии информации и политика по работе с коммерческой информацией. AI-buddy перескажет главное и задаст пару вопросов.',
    helper: 'AI-buddy',
    doneWhen: 'ты ответила на контрольные вопросы',
    goal: null,
  },
  't-502': {
    about:
      'Как оформить стажировку как учебную практику и что можно писать в учебных работах про Авито.',
    helper: 'куратор стажировки',
    doneWhen: 'ты знаешь порядок и сроки',
    goal: null,
  },
  't-601': {
    about: 'Мастер-классы помогают собрать презентацию и подготовить речь. Записи тоже подойдут.',
    helper: 'куратор стажировки',
    doneWhen: 'ты посетила или посмотрела мастер-класс',
    goal: null,
  },
  't-602': {
    about:
      'Презентация по заданной структуре: задачи, цели и прогресс по ним, чему научилась. AI-buddy соберет черновик из твоего пути и целей.',
    helper: 'AI-buddy, куратор помогает с оформлением',
    doneWhen: 'черновик готов и отправлен наставнику',
    goal: 'Все цели стажировки',
  },
  't-603': {
    about:
      'Наставник проверит, что задачи и цели отражены верно. Можно попросить репетицию на 15 минут.',
    helper: 'Игорь, куратор стажировки',
    doneWhen: 'наставник и куратор согласовали презентацию',
    goal: null,
  },
  't-604': {
    about:
      'Расскажи о промежуточных результатах. После экватора наставник даст обратную связь на 1:1.',
    helper: 'Игорь, Олег',
    doneWhen: 'выступление прошло',
    goal: null,
  },
  't-701': {
    about: 'Финальная встреча 1:1: что получилось по целям, куда расти дальше.',
    helper: 'Игорь, наставник',
    doneWhen: 'обратная связь получена',
    goal: 'Все цели стажировки',
  },
  't-702': {
    about: 'Все детали – в приглашении в календаре.',
    helper: 'куратор стажировки',
    doneWhen: 'ты на выпускном',
    goal: null,
  },
  't-703': {
    about: 'Короткий опрос о том, как прошла стажировка и чем помог AI-buddy.',
    helper: 'куратор стажировки',
    doneWhen: 'опрос пройден',
    goal: null,
  },
  't-201': {
    about: 'Трекер нужен, чтобы видеть свои задачи и задачи команды.',
    helper: 'выдается автоматически',
    doneWhen: 'доступ выдан',
    goal: null,
  },
  't-202': {
    about: 'В вики лежат описания сервисов, договоренности и инструкции команды.',
    helper: 'выдается автоматически',
    doneWhen: 'доступ выдан',
    goal: null,
  },
  't-203': {
    about: 'Стенд нужен, чтобы проверять изменения до релиза. Заявка уже отправлена.',
    helper: 'Олег согласует, AI-buddy напоминает ему',
    doneWhen: 'заявка согласована и доступ работает',
    goal: null,
  },
  't-204': {
    about:
      'Без GitLab не получится взять первую задачу. Нужна группа backend с правами Developer: читать код, создавать ветки и merge request.',
    helper: 'AI-buddy заполнит заявку, согласует Олег',
    doneWhen: 'заявка согласована и ты видишь репозиторий',
    goal: 'Пройти путь задачи до релиза',
  },
  't-301': {
    about: 'Склонируй репозиторий, подними сервис локально в Docker и прогони тесты.',
    helper: 'Игорь, статья про окружение',
    doneWhen: 'сервис запускается локально и тесты проходят',
    goal: 'Пройти путь задачи до релиза',
  },
  't-302': {
    about:
      'Посмотри, как в команде пишут и проверяют merge request: сколько нужно аппрувов и что смотрят на ревью.',
    helper: 'Игорь, наставник',
    doneWhen: 'ты прочитала правила и посмотрела пример ревью',
    goal: null,
  },
  't-303': {
    about:
      'Разберись, какие проверки запускаются на merge request и как изменение попадает в прод.',
    helper: 'Игорь, наставник',
    doneWhen: 'ты можешь объяснить путь кода до релиза',
    goal: null,
  },
  't-304': {
    about: 'Узнай, где смотреть ошибки и нагрузку своего сервиса.',
    helper: 'Игорь, наставник',
    doneWhen: 'ты нашла дашборд своего сервиса',
    goal: null,
  },
  't-305': {
    about: 'Подпишись на чат команды и уведомления о дежурствах, чтобы не пропускать важное.',
    helper: 'AI-buddy подскажет каналы',
    doneWhen: 'подписки настроены',
    goal: null,
  },
  't-401': {
    about: 'Маленькая задача, чтобы пройти весь цикл: ветка, merge request, ревью, релиз.',
    helper: 'Игорь, наставник',
    doneWhen: 'изменение в main',
    goal: 'Пройти путь задачи до релиза',
  },
  't-402': {
    about: 'Покрой тестами модуль сервиса вместе с наставником.',
    helper: 'Игорь, наставник',
    doneWhen: 'тесты в main и проходят в CI',
    goal: 'Разобраться в сервисе авторизации',
  },
  't-403': {
    about: 'Первое ревью: посмотри merge request коллеги и оставь комментарии по правилам команды.',
    helper: 'Игорь, наставник',
    doneWhen: 'ревью оставлено',
    goal: null,
  },
}

function seed(): StageSeed[] {
  return [
    {
      id: 'st-01',
      code: '01',
      period: 'неделя 1',
      title: 'Знакомство с командой',
      color: '#C9FFBF',
      tip: 'Если не знаешь, к кому идти с вопросом, спроси AI-buddy: он подскажет нужного человека.',
      suggestions: [
        'Кто за что отвечает в команде?',
        'Как проходят встречи 1:1?',
        'Как оформить отгул?',
      ],
      tasks: [
        {
          id: 't-101',
          title: 'Встреча-знакомство с командой',
          note: 'Проходит в первый понедельник',
          status: 'done',
          due: '2026-11-01',
          action: null,
        },
        {
          id: 't-102',
          title: 'Познакомиться с наставником',
          note: 'Первая встреча 1:1',
          status: 'done',
          due: '2026-11-02',
          action: null,
        },
        {
          id: 't-103',
          title: 'Прочитать памятку стажера',
          note: 'Как устроена команда и процессы',
          status: 'done',
          due: '2026-11-03',
          action: null,
        },
        {
          id: 't-104',
          title: 'Добавиться в чаты и каналы команды',
          note: 'AI-buddy добавит в чат команды, стажеров и дежурств',
          status: 'done',
          due: '2026-11-03',
          action: { kind: 'join_chats', payload: { team: 'backend' } },
        },
        {
          id: 't-105',
          title: 'Завести конспект встреч с наставником',
          note: 'Заметки к 1:1 – в календаре, у каждой встречи',
          status: 'done',
          due: '2026-11-04',
          action: null,
        },
      ],
      resources: [
        {
          id: 'r-11',
          kind: 'person',
          name: 'Игорь',
          note: 'наставник',
          tag: '1:1 по четвергам',
          tagKind: 'mentor',
        },
        {
          id: 'r-12',
          kind: 'person',
          name: 'Олег',
          note: 'руководитель команды',
          tag: 'Backend',
          tagKind: 'none',
        },
        {
          id: 'r-13',
          kind: 'link',
          name: 'Команда Backend',
          note: '6 человек',
          tag: 'чат команды',
          tagKind: 'none',
        },
      ],
    },
    {
      id: 'st-02',
      code: '02',
      period: 'недели 1–2',
      title: 'Доступы',
      color: '#BADAFB',
      tip: 'Доступы выдает руководитель команды. Обычно заявка согласуется за один рабочий день.',
      suggestions: ['Запросить GitLab', 'Сколько ждать согласования?', 'Кто выдает доступы?'],
      tasks: [
        {
          id: 't-201',
          title: 'Доступ к трекеру задач',
          note: 'Выдается автоматически',
          status: 'done',
          due: '2026-11-01',
          action: null,
        },
        {
          id: 't-202',
          title: 'Доступ к вики команды',
          note: 'Выдается автоматически',
          status: 'done',
          due: '2026-11-01',
          action: null,
        },
        {
          id: 't-203',
          title: 'Доступ к тестовому стенду',
          note: 'Ждет подтверждения Олега',
          status: 'waiting',
          due: '2026-11-05',
          action: { kind: 'access_request', payload: { service: 'test_stand', role: 'read' } },
        },
        {
          id: 't-204',
          title: 'Получить доступ к GitLab',
          note: 'Нужен для первой задачи. AI-buddy подготовит заявку, останется подтвердить',
          status: 'in_progress',
          due: '2026-11-10',
          action: gitlab,
        },
      ],
      resources: [
        {
          id: 'r-21',
          kind: 'access',
          name: 'Трекер задач',
          note: 'чтение и запись',
          tag: '',
          tagKind: 'none',
          taskId: 't-201',
        },
        {
          id: 'r-22',
          kind: 'access',
          name: 'Вики команды',
          note: 'чтение',
          tag: '',
          tagKind: 'none',
          taskId: 't-202',
        },
        {
          id: 'r-23',
          kind: 'access',
          name: 'Тестовый стенд',
          note: 'роль: чтение',
          tag: '',
          tagKind: 'none',
          taskId: 't-203',
        },
        {
          id: 'r-24',
          kind: 'access',
          name: 'GitLab',
          note: 'группа backend',
          tag: '',
          tagKind: 'none',
          taskId: 't-204',
        },
      ],
    },
    {
      id: 'st-03',
      code: '03',
      period: 'недели 2–4',
      title: 'Рабочие инструменты',
      color: '#DFCDF3',
      tip: 'Откроется после этапа «Доступы»: без GitLab окружение не настроить.',
      suggestions: ['Как устроен CI/CD команды?', 'Что такое код-ревью?'],
      tasks: [
        {
          id: 't-301',
          title: 'Настроить рабочее окружение',
          note: 'Репозиторий, Docker, локальный запуск',
          status: 'todo',
          due: '2026-11-14',
          action: null,
        },
        {
          id: 't-302',
          title: 'Изучить процесс код-ревью',
          note: 'Памятка и пример ревью',
          status: 'todo',
          due: '2026-11-16',
          action: { kind: 'ask_buddy', payload: { question: 'Как устроено код-ревью в команде?' } },
        },
        {
          id: 't-303',
          title: 'Разобраться с CI/CD команды',
          note: 'Как код попадает в прод',
          status: 'todo',
          due: '2026-11-18',
          action: { kind: 'ask_buddy', payload: { question: 'Как устроен CI/CD команды?' } },
        },
        {
          id: 't-304',
          title: 'Посмотреть дашборды метрик',
          note: 'Где смотреть ошибки и нагрузку',
          status: 'todo',
          due: '2026-11-19',
          action: null,
        },
        {
          id: 't-305',
          title: 'Настроить уведомления',
          note: 'Чат команды и дежурства',
          status: 'todo',
          due: '2026-11-20',
          action: null,
        },
      ],
      resources: [
        {
          id: 'r-31',
          kind: 'tool',
          name: 'Docker',
          note: 'локальный запуск',
          tag: 'нужен',
          tagKind: 'none',
        },
        {
          id: 'r-32',
          kind: 'access',
          name: 'GitLab',
          note: 'код и ревью',
          tag: '',
          tagKind: 'none',
          taskId: 't-204',
        },
        {
          id: 'r-33',
          kind: 'tool',
          name: 'Дашборды метрик',
          note: 'мониторинг',
          tag: 'нужен',
          tagKind: 'none',
        },
      ],
    },
    {
      id: 'st-04',
      code: '04',
      period: 'недели 4–8',
      title: 'Первые задачи',
      color: '#F7C1C5',
      tip: 'Первая задача нужна не для результата, а чтобы пройти путь от задачи до релиза.',
      suggestions: ['Как проходит код-ревью?', 'Как оформлять задачи?'],
      tasks: [
        {
          id: 't-401',
          title: 'Исправить текст ошибки входа',
          note: 'Маленькая задача, чтобы пройти весь цикл',
          status: 'todo',
          due: '2026-11-25',
          action: null,
        },
        {
          id: 't-402',
          title: 'Написать тесты к модулю',
          note: 'Вместе с наставником',
          status: 'todo',
          due: '2026-11-30',
          action: null,
        },
        {
          id: 't-403',
          title: 'Сделать ревью кода коллеги',
          note: 'Первое ревью',
          status: 'todo',
          due: '2026-12-05',
          action: null,
        },
      ],
      resources: [
        {
          id: 'r-41',
          kind: 'article',
          name: 'Памятка стажера',
          note: 'раздел «Первые задачи»',
          tag: 'читать',
          tagKind: 'none',
        },
        {
          id: 'r-42',
          kind: 'article',
          name: 'Правила код-ревью',
          note: 'чек-лист',
          tag: 'читать',
          tagKind: 'none',
        },
      ],
    },
    {
      id: 'st-05',
      code: '05',
      period: 'месяцы 1–2',
      title: 'Безопасность и учеба',
      color: '#C9FFBF',
      tip: 'Это обязательные материалы. AI-buddy коротко перескажет главное и задаст пару вопросов для проверки.',
      suggestions: ['Перескажи правила защиты данных', 'Как закрыть практику по Авито?'],
      tasks: [
        {
          id: 't-501',
          title: 'Пройти материалы по защите данных',
          note: 'Лендинг «Защита данных», памятка о раскрытии информации, политика по КИ',
          status: 'todo',
          due: '2026-12-07',
          action: {
            kind: 'ask_buddy',
            payload: { prompt: 'Перескажи коротко правила защиты данных и проверь меня' },
          },
        },
        {
          id: 't-502',
          title: 'Прочитать, как закрыть практику по Авито',
          note: 'Если стажировка засчитывается как практика в вузе',
          status: 'todo',
          due: '2027-01-11',
          action: null,
        },
      ],
      resources: [
        {
          id: 'r-51',
          kind: 'article',
          name: 'Защита данных',
          note: 'лендинг и памятки',
          tag: 'обязательно',
          tagKind: 'wait',
        },
      ],
    },
    {
      id: 'st-06',
      code: '06',
      period: 'после 3 месяцев',
      title: 'Экватор',
      color: '#BADAFB',
      tip: 'Экватор – промежуточные итоги. AI-buddy соберет черновик презентации из закрытых задач и целей.',
      suggestions: ['Собери черновик презентации к экватору', 'Какая структура у презентации?'],
      tasks: [
        {
          id: 't-601',
          title: 'Сходить на мастер-классы к экватору',
          note: 'Приглашения придут в календарь',
          status: 'todo',
          due: '2027-02-01',
          action: null,
        },
        {
          id: 't-602',
          title: 'Подготовить презентацию с результатами',
          note: 'AI-buddy соберет черновик по твоим задачам и целям',
          status: 'todo',
          due: '2027-02-08',
          action: {
            kind: 'ask_buddy',
            payload: { prompt: 'Собери черновик презентации к экватору по моим задачам и целям' },
          },
        },
        {
          id: 't-603',
          title: 'Согласовать презентацию с куратором и наставником',
          note: 'При желании – репетиция на 15 минут',
          status: 'todo',
          due: '2027-02-12',
          action: null,
        },
        {
          id: 't-604',
          title: 'Выступить на экваторе',
          note: 'Промежуточные результаты по целям',
          status: 'todo',
          due: '2027-02-15',
          action: null,
        },
      ],
      resources: [
        {
          id: 'r-61',
          kind: 'person',
          name: 'Куратор стажировки',
          note: 'помогает с презентацией',
          tag: 'HR',
          tagKind: 'none',
        },
      ],
    },
    {
      id: 'st-07',
      code: '07',
      period: 'конец стажировки',
      title: 'Итоги',
      color: '#DFCDF3',
      tip: 'Итоги подводят по целям стажировки: что получилось и куда расти дальше.',
      suggestions: ['Как проходит выпускной?', 'Что будет после стажировки?'],
      tasks: [
        {
          id: 't-701',
          title: 'Обсудить итоги с наставником на 1:1',
          note: 'Обратная связь по целям',
          status: 'todo',
          due: '2027-04-19',
          action: null,
        },
        {
          id: 't-702',
          title: 'Прийти на выпускной',
          note: 'Приглашение придет в календарь',
          status: 'todo',
          due: '2027-04-23',
          action: null,
        },
        {
          id: 't-703',
          title: 'Пройти опрос о стажировке',
          note: 'Пара минут, помогает сделать программу лучше',
          status: 'todo',
          due: '2027-04-26',
          action: null,
        },
      ],
      resources: [],
    },
  ]
}

let stages = seed()

/** Второй стажер – Павел, первая неделя: готова только встреча с командой. */
function seedPavel(): StageSeed[] {
  return seed().map((st) => ({
    ...st,
    tasks: st.tasks.map((t) => ({ ...t, status: t.id === 't-101' ? 'done' : 'todo' })),
  }))
}
let pavel = seedPavel()

export const PAVEL_ID = 'u-pavel'

export function resetPathMock() {
  stages = seed()
  pavel = seedPavel()
}

export function setMockTaskStatus(taskId: string, status: Status, internId = 'u-anna'): boolean {
  for (const s of internId === PAVEL_ID ? pavel : stages) {
    const t = s.tasks.find((x) => x.id === taskId)
    if (t) {
      t.status = status
      return true
    }
  }
  return false
}

function accessTag(status: Status | undefined) {
  if (status === 'done') return { tag: 'активен', tagKind: 'ok' as const }
  if (status === 'waiting' || status === 'in_progress')
    return {
      tag: status === 'waiting' ? 'на согласовании' : 'не запрошен',
      tagKind: status === 'waiting' ? ('wait' as const) : ('none' as const),
    }
  return { tag: 'не запрошен', tagKind: 'none' as const }
}

export function buildMockPath(
  intern: { name: string; teamName: string } = { name: 'Анна', teamName: 'Backend' },
  today = '2026-11-10',
  src: StageSeed[] = stages,
): TPathView {
  const stages = src
  const taskStatus = new Map(stages.flatMap((s) => s.tasks.map((t) => [t.id, t.status] as const)))
  const all = stages.flatMap((s) => s.tasks)
  const next =
    all.find((t) => t.status === 'in_progress') ?? all.find((t) => t.status === 'todo') ?? null
  let currentFound = false
  const views = stages.map((s) => {
    const done = s.tasks.filter((t) => t.status === 'done').length
    let status: 'done' | 'current' | 'todo' = 'todo'
    if (done === s.tasks.length) status = 'done'
    else if (!currentFound) {
      status = 'current'
      currentFound = true
    }
    return {
      id: s.id,
      code: s.code,
      title: s.title,
      color: s.color,
      period: s.period,
      tip: s.tip,
      status,
      doneCount: done,
      totalCount: s.tasks.length,
      suggestions: s.suggestions,
      tasks: s.tasks.map((t) => ({
        id: t.id,
        title: t.title,
        note: t.note,
        status: t.status,
        isNext: next?.id === t.id,
        dueDate: t.due,
        action: t.action,
        details: DETAILS[t.id] ?? null,
      })),
      resources: s.resources.map((r) => {
        const tag = r.taskId
          ? accessTag(taskStatus.get(r.taskId))
          : { tag: r.tag, tagKind: r.tagKind }
        return { id: r.id, kind: r.kind, name: r.name, note: r.note, ...tag }
      }),
    }
  })
  const done = all.filter((t) => t.status === 'done').length
  const inProgress = all.filter((t) => t.status === 'in_progress' || t.status === 'waiting').length
  const nextStage = next ? views.find((s) => s.tasks.some((t) => t.id === next.id))! : null
  const dueInDays = next?.due
    ? Math.round((Date.parse(next.due) - Date.parse(today)) / 86_400_000)
    : null
  const current = views.find((s) => s.status === 'current')
  return {
    intern: {
      id: 'u-anna',
      name: intern.name,
      teamName: intern.teamName,
      mentor: { id: 'u-igor', name: 'Игорь' },
    },
    week: {
      current: 2,
      total: 24,
      day: 10,
      startDate: '2026-11-01',
      milestones: [
        { week: 1, title: 'Старт' },
        { week: 4, title: 'Первые задачи' },
        { week: 12, title: 'Отзыв' },
        { week: 24, title: 'Итоги' },
      ],
    },
    progress: {
      done,
      inProgress,
      todo: all.length - done - inProgress,
      total: all.length,
      percent: Math.round((done / all.length) * 100),
    },
    stages: views,
    nextStep:
      next && nextStage
        ? {
            taskId: next.id,
            stageId: nextStage.id,
            stageCode: nextStage.code,
            title: next.title,
            status: next.status,
            dueDate: next.due,
            dueInDays,
            action: next.action,
          }
        : null,
    nextMeeting: {
      id: 'm-1',
      title: 'Встреча 1:1',
      startsAt: '2026-11-12T15:00:00+03:00',
      hostId: 'u-igor',
      hostName: 'Игорь',
    },
    buddy: current
      ? {
          text: `${intern.name}, ты на этапе «${current.title}». Осталось ${current.totalCount - current.doneCount} из ${current.totalCount}.${next ? ` Следующий шаг – «${next.title}». Могу помочь сделать его прямо сейчас.` : ''}`,
          suggestions: current.suggestions,
        }
      : { text: 'Все этапы пройдены! Поздравляю.', suggestions: ['Что дальше?'] },
  }
}

export const mockInterns = () => {
  const p = buildMockPath()
  const cur = p.stages.find((s) => s.status === 'current')
  return [
    {
      id: 'u-anna',
      login: 'intern_anna',
      name: 'Анна',
      teamName: 'Backend',
      mentor: { id: 'u-igor', name: 'Игорь' },
      week: 2,
      progress: p.progress,
      currentStageCode: cur?.code ?? '',
      currentStage: cur?.title ?? '',
    },
    (() => {
      const pp = buildMockPavelPath()
      const c = pp.stages.find((x) => x.status === 'current')
      return {
        id: PAVEL_ID,
        login: 'intern_pavel',
        name: 'Павел',
        teamName: 'Backend',
        mentor: { id: 'u-igor', name: 'Игорь' },
        week: 1,
        progress: pp.progress,
        currentStageCode: c?.code ?? '',
        currentStage: c?.title ?? '',
      }
    })(),
  ]
}

export const buildMockPavelPath = () =>
  buildMockPath({ name: 'Павел', teamName: 'Backend' }, '2026-11-10', pavel)
