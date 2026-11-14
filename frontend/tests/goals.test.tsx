import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { API_URL } from '../src/api/client'
import { resetGoalsMock } from '../src/mocks/goalsData'
import { server } from '../src/mocks/server'
import { agreedLabel, nextPct, withName, type TGoalsView } from '../src/schemas/goals'
import { loginAs, renderApp } from './test-utils'

const T = { timeout: 3000 }

afterEach(() => resetGoalsMock())

describe('Цели стажировки', () => {
  it('стажер видит цели с шагами; прогресс не кликается', async () => {
    loginAs('intern_maria')
    renderApp('/')
    expect(await screen.findByText('Мои цели на стажировку', {}, T)).toBeInTheDocument()
    expect(screen.getByText(/согласованы с Артемием .* · от OKR команды/)).toBeInTheDocument()
    const goal = screen.getByLabelText('Разобраться в сервисе авторизации: 25%')
    expect(
      within(goal).getByText(/критерий: провести демо команде на 8 неделе/),
    ).toBeInTheDocument()
    expect(within(goal).getByText(/1 из 4 шагов/)).toBeInTheDocument()
    // шаги скрыты, пока цель не раскрыта
    expect(within(goal).queryByRole('list', { name: /Шаги цели/ })).toBeNull()
    await userEvent.click(within(goal).getByRole('button', { name: /показать шаги/ }))
    const steps = within(goal).getByRole('list', { name: /Шаги цели/ })
    expect(within(steps).getAllByRole('listitem')).toHaveLength(4)
    expect(within(steps).getByText('· этап 01')).toBeInTheDocument()
    expect(within(steps).getAllByText('· наставник').length).toBeGreaterThan(0)
  })

  it('цель без шагов (старый ответ backend) показывает полоску процента', async () => {
    server.use(
      http.get(`${API_URL}/api/goals`, () =>
        HttpResponse.json({
          internId: 'u-anna',
          agreed: null,
          fromOkr: false,
          goals: [
            {
              id: 'g',
              position: 1,
              title: 'Цель без шагов',
              criterion: 'к 4 неделе',
              dueWeek: 4,
              progressPct: 40,
              okr: null,
              lastUpdate: null,
              updatedAt: '2026-11-10T10:00:00Z',
              createdBy: null,
            },
          ],
          okrs: [],
          canEdit: false,
          canAgree: false,
        }),
      ),
    )
    loginAs('intern_maria')
    renderApp('/')
    expect(await screen.findByLabelText('Цель без шагов: 40%', {}, T)).toBeInTheDocument()
  })

  it('у стажера без целей – подсказка про встречу с руководителем', async () => {
    loginAs('intern_pavel')
    renderApp('/')
    expect(
      await screen.findByText(/Цели появятся после встречи с руководителем/, {}, T),
    ).toBeInTheDocument()
    expect(screen.getByText('еще не согласованы с руководителем')).toBeInTheDocument()
  })

  it('руководитель видит цели стажера в его карточке', async () => {
    loginAs('lead_artemiy')
    renderApp('/interns/u-anna')
    expect(await screen.findByText('Цели стажировки', {}, T)).toBeInTheDocument()
    expect(screen.getByText('Самостоятельно закрывать задачи уровня junior')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Согласовать цели' })).not.toBeInTheDocument()
  })

  it('подписи и шаг прогресса', () => {
    expect(withName('Артемий')).toBe('Артемием')
    expect(withName('Нина')).toBe('Ниной')
    expect(withName('Игорь')).toBe('Игорем')
    expect(withName('Илья')).toBe('Ильей')
    expect(withName('Bob')).toBe('Bob')
    expect(nextPct(10)).toBe(35)
    expect(nextPct(90)).toBe(100)
    expect(nextPct(100)).toBe(0)
    const view: TGoalsView = {
      internId: 'x',
      agreed: { by: { id: '1', name: 'Артемий' }, at: '2026-11-04T10:00:00Z' },
      fromOkr: false,
      goals: [],
      okrs: [],
      canEdit: false,
      canAgree: false,
    }
    expect(agreedLabel(view)).toBe('согласованы с Артемием 4 нояб')
  })
})

describe('Справочник: статья с сервера', () => {
  it('в окне статьи показывается текст целиком', async () => {
    loginAs('intern_maria')
    renderApp('/kb?article=a-leave')
    const dialog = await screen.findByRole('dialog', {}, T)
    expect(await within(dialog).findByText(/Демо-текст статьи/, {}, T)).toBeInTheDocument()
  })

  it('поиск идет на сервер и показывает найденное', async () => {
    loginAs('intern_maria')
    renderApp('/kb')
    await screen.findByText('Популярное', {}, T)
    await userEvent.type(screen.getByLabelText('Поиск по справочнику'), 'отгул')
    expect(await screen.findByRole('button', { name: /Как взять отгул/ }, T)).toBeInTheDocument()
  })
})
