import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { HttpResponse, http } from 'msw'
import type { ReactNode } from 'react'
import { API_URL } from '../src/api/client'
import * as api from '../src/api/path'
import * as hooks from '../src/hooks/usePath'
import { server } from '../src/mocks/server'
import { createTestQueryClient, loginAs } from './test-utils'

const stage = {
  id: 's1',
  teamId: 'team',
  position: 1,
  title: 'Этап',
  color: '#C9FFBF',
  tip: '',
  suggestions: [],
}
const task = {
  id: 't1',
  stageId: 's1',
  position: 1,
  title: 'Задача',
  note: '',
  dueDay: null,
  action: null,
}
const resource = {
  id: 'r1',
  stageId: 's1',
  position: 1,
  kind: 'tool',
  name: 'Docker',
  note: '',
  tag: '',
  tagKind: 'none',
  taskId: null,
}

beforeEach(() => {
  loginAs('mentor_igor')
  server.use(
    http.get(`${API_URL}/api/console/templates`, () =>
      HttpResponse.json([{ ...stage, tasks: [task], resources: [resource] }]),
    ),
    http.get(`${API_URL}/api/console/action-kinds`, () =>
      HttpResponse.json([{ kind: 'access_request', description: 'заявка на доступ' }]),
    ),
    http.post(`${API_URL}/api/console/stages`, () => HttpResponse.json(stage, { status: 201 })),
    http.put(`${API_URL}/api/console/stages/:id`, () => HttpResponse.json(stage)),
    http.delete(`${API_URL}/api/console/stages/:id`, () => new HttpResponse(null, { status: 204 })),
    http.post(`${API_URL}/api/console/stages/:id/tasks`, () =>
      HttpResponse.json(task, { status: 201 }),
    ),
    http.put(`${API_URL}/api/console/tasks/:id`, () => HttpResponse.json(task)),
    http.delete(`${API_URL}/api/console/tasks/:id`, () => new HttpResponse(null, { status: 204 })),
    http.post(`${API_URL}/api/console/stages/:id/resources`, () =>
      HttpResponse.json(resource, { status: 201 }),
    ),
    http.delete(
      `${API_URL}/api/console/resources/:id`,
      () => new HttpResponse(null, { status: 204 }),
    ),
    http.post(`${API_URL}/api/interns/:id/meetings`, () =>
      HttpResponse.json(
        {
          id: 'm2',
          title: 'Встреча 1:1',
          startsAt: '2026-11-13T11:00:00+03:00',
          hostId: 'u-igor',
          hostName: 'Игорь',
        },
        { status: 201 },
      ),
    ),
  )
})

describe('api/path', () => {
  it('стажеры и путь стажера', async () => {
    expect(await api.getInterns()).toHaveLength(2)
    expect((await api.getInternPath('u-anna')).intern.name).toBe('Мария')
    const path = await api.setInternTaskStatus('u-anna', 't-204', { status: 'done' })
    expect(path.progress.done).toBe(8)
    expect(
      (await api.createMeeting('u-anna', { startsAt: '2026-11-13T11:00:00+03:00' })).title,
    ).toBe('Встреча 1:1')
  })

  it('консоль наставника: шаблоны', async () => {
    expect(await api.getTemplates()).toHaveLength(1)
    expect(await api.getActionKinds()).toHaveLength(1)
    expect((await api.createStage({ title: 'Этап' })).id).toBe('s1')
    expect((await api.updateStage('s1', { title: 'Этап', position: 1 })).id).toBe('s1')
    expect(await api.deleteStage('s1')).toBeUndefined()
    expect((await api.createTask('s1', { title: 'Задача' })).id).toBe('t1')
    expect((await api.updateTask('t1', { title: 'Задача', position: 1 })).id).toBe('t1')
    expect(await api.deleteTask('t1')).toBeUndefined()
    expect((await api.createResource('s1', { kind: 'tool', name: 'Docker' })).id).toBe('r1')
    expect(await api.deleteResource('r1')).toBeUndefined()
  })
})

describe('hooks/usePath', () => {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={createTestQueryClient()}>{children}</QueryClientProvider>
  )

  it('запросы', async () => {
    const t = renderHook(() => hooks.useTemplates(), { wrapper })
    const k = renderHook(() => hooks.useActionKinds(), { wrapper })
    const p = renderHook(() => hooks.useInternPath('u-anna'), { wrapper })
    await waitFor(() => expect(t.result.current.data).toHaveLength(1))
    await waitFor(() => expect(k.result.current.data).toHaveLength(1))
    await waitFor(() => expect(p.result.current.data?.intern.name).toBe('Мария'))
  })

  it('мутации шаблонов и встреч', async () => {
    const { result } = renderHook(
      () => ({
        createStage: hooks.useCreateStage(),
        updateStage: hooks.useUpdateStage(),
        deleteStage: hooks.useDeleteStage(),
        createTask: hooks.useCreateTask(),
        updateTask: hooks.useUpdateTask(),
        deleteTask: hooks.useDeleteTask(),
        createResource: hooks.useCreateResource(),
        deleteResource: hooks.useDeleteResource(),
        meeting: hooks.useCreateMeeting('u-anna'),
      }),
      { wrapper },
    )
    await act(async () => {
      await result.current.createStage.mutateAsync({ title: 'Этап' })
      await result.current.updateStage.mutateAsync({ id: 's1', title: 'Этап', position: 1 })
      await result.current.deleteStage.mutateAsync('s1')
      await result.current.createTask.mutateAsync({ stageId: 's1', title: 'Задача' })
      await result.current.updateTask.mutateAsync({ id: 't1', title: 'Задача', position: 1 })
      await result.current.deleteTask.mutateAsync('t1')
      await result.current.createResource.mutateAsync({
        stageId: 's1',
        kind: 'tool',
        name: 'Docker',
      })
      await result.current.deleteResource.mutateAsync('r1')
      await result.current.meeting.mutateAsync({ startsAt: '2026-11-13T11:00:00+03:00' })
    })
    await waitFor(() => expect(result.current.meeting.isSuccess).toBe(true))
  })
})
