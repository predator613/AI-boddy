// Консоль наставника (шаблон плана команды) для полного режима без backend и для тестов.
// Повторяет формат /api/console/* из docs/api-my-path.md. С настоящим backend не используется.
import { HttpResponse, http } from 'msw'
import { API_URL } from '../api/client'
import type { TStageTemplate, TTaskTemplate } from '../schemas/path'

const TEAM = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
let seq = 0
const nid = (p: string) => `${p}-${++seq}`

type ActionT = TTaskTemplate['action']
const task = (
  stageId: string,
  position: number,
  title: string,
  dueDay: number,
  action: ActionT = null,
  note = '',
): TTaskTemplate => ({
  id: nid('tt'),
  stageId,
  position,
  title,
  note,
  dueDay,
  action,
})

function seed(): TStageTemplate[] {
  seq = 0
  const st = (
    id: string,
    position: number,
    title: string,
    color: string,
    tasks: TTaskTemplate[],
  ): TStageTemplate => ({
    id,
    teamId: TEAM,
    position,
    title,
    color,
    tip: '',
    suggestions: [],
    tasks,
    resources: [],
  })
  return [
    st('st-1', 1, 'Знакомство с командой', '#C9FFBF', [
      task('st-1', 1, 'Встреча-знакомство с командой', 1),
      task('st-1', 2, 'Познакомиться с наставником', 2),
      task('st-1', 3, 'Прочитать памятку стажера', 3, { kind: 'open_article', payload: {} }),
      task('st-1', 4, 'Обсудить цели и критерии успеха с руководителем', 4),
    ]),
    st('st-2', 2, 'Доступы', '#BADAFB', [
      task('st-2', 1, 'Доступ к трекеру задач', 1),
      task('st-2', 2, 'Доступ к тестовому стенду', 5, {
        kind: 'access_request',
        payload: { service: 'test_stand' },
      }),
      task('st-2', 3, 'Запросить трекер прогресса и внести цели', 8, {
        kind: 'ask_buddy',
        payload: {},
      }),
      task('st-2', 4, 'Получить доступ к GitLab', 10, {
        kind: 'access_request',
        payload: { service: 'gitlab' },
      }),
    ]),
    st('st-3', 3, 'Рабочие инструменты', '#DFCDF3', [
      task('st-3', 1, 'Настроить рабочее окружение', 14),
      task('st-3', 2, 'Изучить процесс код-ревью', 16, { kind: 'ask_buddy', payload: {} }),
    ]),
    st('st-4', 4, 'Первые задачи', '#F7C1C5', [
      task('st-4', 1, 'Исправить текст ошибки входа', 25),
      task('st-4', 2, 'Написать тесты к модулю', 30),
    ]),
  ]
}

let stages = seed()
export function resetConsoleMock() {
  stages = seed()
}

const err = (status: number, code: string, message: string) =>
  HttpResponse.json({ error: { code, message } }, { status })
type Body = Record<string, unknown>
const readBody = async (r: Request) => ((await r.json().catch(() => null)) ?? {}) as Body
const stageOf = (taskId: string) => stages.find((s) => s.tasks?.some((t) => t.id === taskId))

function taskFrom(
  b: Body,
  base: Pick<TTaskTemplate, 'id' | 'stageId' | 'position'>,
): TTaskTemplate | null {
  const title = String(b.title ?? '').trim()
  if (!title) return null
  return {
    ...base,
    title,
    note: String(b.note ?? ''),
    dueDay: typeof b.dueDay === 'number' ? b.dueDay : null,
    action: (b.action as ActionT) ?? null,
  }
}

export const consoleHandlers = [
  http.get(`${API_URL}/api/console/templates`, () => HttpResponse.json(stages)),
  http.get(`${API_URL}/api/console/action-kinds`, () =>
    HttpResponse.json([
      { kind: 'access_request', description: 'заявка на доступ' },
      { kind: 'leave_request', description: 'заявка на отгул' },
      { kind: 'ask_buddy', description: 'открыть чат с готовым вопросом' },
      { kind: 'open_article', description: 'открыть статью справочника' },
    ]),
  ),
  http.post(`${API_URL}/api/console/stages`, async ({ request }) => {
    const b = await readBody(request)
    const title = String(b.title ?? '').trim()
    if (!title) return err(400, 'validation_error', 'title is required (max 80)')
    const s: TStageTemplate = {
      id: nid('st'),
      teamId: TEAM,
      position: stages.length + 1,
      title,
      color: String(b.color ?? '#C9FFBF'),
      tip: String(b.tip ?? ''),
      suggestions: [],
      tasks: [],
      resources: [],
    }
    stages = [...stages, s]
    return HttpResponse.json(s, { status: 201 })
  }),
  http.put(`${API_URL}/api/console/stages/:id`, async ({ request, params }) => {
    const s = stages.find((x) => x.id === params.id)
    if (!s) return err(404, 'not_found', 'not found')
    const b = await readBody(request)
    const next = {
      ...s,
      title: String(b.title ?? s.title),
      color: String(b.color ?? s.color),
      tip: String(b.tip ?? s.tip),
    }
    const others = stages.filter((x) => x.id !== s.id)
    const pos = Math.min(Math.max(Number(b.position ?? s.position), 1), stages.length)
    others.splice(pos - 1, 0, next)
    stages = others.map((x, i) => ({ ...x, position: i + 1 }))
    return HttpResponse.json(stages.find((x) => x.id === s.id))
  }),
  http.delete(`${API_URL}/api/console/stages/:id`, ({ params }) => {
    if (!stages.some((x) => x.id === params.id)) return err(404, 'not_found', 'not found')
    stages = stages.filter((x) => x.id !== params.id).map((x, i) => ({ ...x, position: i + 1 }))
    return new HttpResponse(null, { status: 204 })
  }),
  http.post(`${API_URL}/api/console/stages/:id/tasks`, async ({ request, params }) => {
    const s = stages.find((x) => x.id === params.id)
    if (!s) return err(404, 'not_found', 'not found')
    const t = taskFrom(await readBody(request), {
      id: nid('tt'),
      stageId: s.id,
      position: (s.tasks?.length ?? 0) + 1,
    })
    if (!t) return err(400, 'validation_error', 'title is required (max 120)')
    stages = stages.map((x) => (x.id === s.id ? { ...x, tasks: [...(x.tasks ?? []), t] } : x))
    return HttpResponse.json(t, { status: 201 })
  }),
  http.put(`${API_URL}/api/console/tasks/:id`, async ({ request, params }) => {
    const s = stageOf(String(params.id))
    const cur = s?.tasks?.find((t) => t.id === params.id)
    if (!s || !cur) return err(404, 'not_found', 'not found')
    const t = taskFrom(await readBody(request), cur)
    if (!t) return err(400, 'validation_error', 'title is required (max 120)')
    stages = stages.map((x) =>
      x.id === s.id ? { ...x, tasks: x.tasks?.map((y) => (y.id === t.id ? t : y)) } : x,
    )
    return HttpResponse.json(t)
  }),
  http.delete(`${API_URL}/api/console/tasks/:id`, ({ params }) => {
    const s = stageOf(String(params.id))
    if (!s) return err(404, 'not_found', 'not found')
    stages = stages.map((x) =>
      x.id === s.id ? { ...x, tasks: x.tasks?.filter((y) => y.id !== params.id) } : x,
    )
    return new HttpResponse(null, { status: 204 })
  }),
]
