// Синтетический «Мой путь» для режима без backend и для тестов.
// Повторяет seed backend (004_seed_onboarding.sql) и формат GET /api/path.
import type { TPathView } from '../schemas/path'

type Status = 'todo' | 'in_progress' | 'waiting' | 'done'
type ActionT = {
  kind: 'access_request' | 'leave_request' | 'ask_buddy' | 'open_article'
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
  tip: string
  suggestions: string[]
  tasks: TaskSeed[]
  resources: ResSeed[]
}

const gitlab: ActionT = {
  kind: 'access_request',
  payload: { service: 'gitlab', group: 'backend', role: 'developer' },
}

function seed(): StageSeed[] {
  return [
    {
      id: 'st-01',
      code: '01',
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
  ]
}

let stages = seed()

export function resetPathMock() {
  stages = seed()
}

export function setMockTaskStatus(taskId: string, status: Status): boolean {
  for (const s of stages) {
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
): TPathView {
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
  ]
}
