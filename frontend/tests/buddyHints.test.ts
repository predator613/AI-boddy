import { describe, expect, it } from 'vitest'
import { chatHints, pathHints, stageHints, taskHint } from '../src/lib/buddyHints'
import type { TRequestItem } from '../src/schemas/assistant'
import type { TPathView } from '../src/schemas/path'

type Stage = TPathView['stages'][number]
type Task = Stage['tasks'][number]

const task = (over: Partial<Task>): Task =>
  ({
    id: 't',
    title: 'Задача',
    note: '',
    status: 'todo',
    isNext: false,
    dueDate: null,
    action: null,
    details: null,
    ...over,
  }) as Task

const stage = (tasks: Task[], suggestions: string[]): Stage =>
  ({
    id: 's',
    code: '02',
    title: 'Доступы',
    status: 'current',
    tasks,
    suggestions,
  }) as unknown as Stage

const gitlab = (status: Task['status']) =>
  task({
    id: 'g',
    title: 'Получить доступ к GitLab',
    status,
    action: { kind: 'access_request', payload: { service: 'gitlab' } },
  })

describe('подсказки AI-buddy по состоянию', () => {
  it('задача: просьба помочь, вопрос о статусе, готовый вопрос, ничего для сделанной', () => {
    expect(taskHint(gitlab('todo'))).toBe('Помоги: получить доступ к GitLab')
    expect(taskHint(gitlab('waiting'))).toBe('Что с задачей «Получить доступ к GitLab»?')
    expect(taskHint(gitlab('done'))).toBeNull()
    expect(
      taskHint(
        task({ action: { kind: 'ask_buddy', payload: { prompt: 'Как проходит код-ревью?' } } }),
      ),
    ).toBe('Как проходит код-ревью?')
    expect(taskHint(task({}))).toBeNull()
  })

  it('«Запросить GitLab» пропадает, когда доступ уже выдан', () => {
    const plan = ['Запросить GitLab', 'Сколько ждать согласования?']
    expect(stageHints(stage([gitlab('todo')], plan))).toEqual([
      'Помоги: получить доступ к GitLab',
      'Запросить GitLab',
      'Сколько ждать согласования?',
    ])
    expect(stageHints(stage([gitlab('done')], plan))).toEqual(['Сколько ждать согласования?'])
  })

  it('не больше четырех, следующий шаг первым', () => {
    const tasks = ['A', 'B', 'C', 'D', 'E'].map((t, i) =>
      task({
        id: t,
        title: `Сделать ${t}`,
        isNext: i === 3,
        action: { kind: 'leave_request', payload: {} },
      }),
    )
    const hints = stageHints(stage(tasks, ['Еще вопрос']))
    expect(hints).toHaveLength(4)
    expect(hints[0]).toBe('Помоги: сделать D')
  })

  it('без текущего этапа – подсказки с главной', () => {
    const path = {
      stages: [stage([], ['x'])].map((s) => ({ ...s, status: 'done' as const })),
      buddy: { text: '', suggestions: ['Что дальше?'] },
    } as unknown as TPathView
    expect(pathHints(path)).toEqual(['Что дальше?'])
  })

  it('чат: заявка на согласовании и следующий шаг, затем общие вопросы', () => {
    const path = {
      stages: [stage([gitlab('todo')], [])],
      nextStep: { taskId: 'g', title: 'Получить доступ к GitLab', status: 'todo' },
      buddy: { text: '', suggestions: [] },
    } as unknown as TPathView
    const waiting = [{ number: '1043', status: 'waiting' }] as TRequestItem[]
    expect(chatHints(path, waiting)).toEqual([
      'Что с заявкой №1043?',
      'Помоги: получить доступ к GitLab',
      'Какой мой следующий шаг?',
      'Как оформить отгул?',
    ])
    // без пути и заявок – общие вопросы, как раньше
    expect(chatHints(null, null)).toEqual([
      'Какой мой следующий шаг?',
      'Как оформить отгул?',
      'Кто мой наставник?',
      'Как устроено код-ревью?',
    ])
  })
})
