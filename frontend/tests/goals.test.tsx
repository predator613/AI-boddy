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
  it('стажер видит согласованные цели и отмечает прогресс кликом', async () => {
    loginAs('intern_anna')
    renderApp('/')
    expect(await screen.findByText('Мои цели на стажировку', {}, T)).toBeInTheDocument()
    expect(screen.getByText(/согласованы с Олегом .* · от OKR команды/)).toBeInTheDocument()
    const goal = screen.getByRole('button', { name: /Разобраться в сервисе авторизации: 10%/ })
    expect(
      within(goal).getByText('критерий: провести демо команде на 8 неделе'),
    ).toBeInTheDocument()
    await userEvent.click(goal)
    expect(
      await screen.findByRole('button', { name: /Разобраться в сервисе авторизации: 35%/ }, T),
    ).toBeInTheDocument()
  })

  it('ошибка сохранения откатывает прогресс', async () => {
    server.use(
      http.post(`${API_URL}/api/goals/:goalId/progress`, () =>
        HttpResponse.json({ error: { code: 'internal_error', message: 'x' } }, { status: 500 }),
      ),
    )
    loginAs('intern_anna')
    renderApp('/')
    await userEvent.click(
      await screen.findByRole('button', { name: /Пройти путь задачи до релиза: 0%/ }, T),
    )
    expect(
      await screen.findByRole('button', { name: /Пройти путь задачи до релиза: 0%/ }, T),
    ).toBeInTheDocument()
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
    loginAs('lead_oleg')
    renderApp('/interns/u-anna')
    expect(await screen.findByText('Цели стажировки', {}, T)).toBeInTheDocument()
    expect(screen.getByText('Самостоятельно закрывать задачи уровня junior')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Согласовать цели' })).not.toBeInTheDocument()
  })

  it('подписи и шаг прогресса', () => {
    expect(withName('Олег')).toBe('Олегом')
    expect(withName('Нина')).toBe('Ниной')
    expect(withName('Игорь')).toBe('Игорем')
    expect(withName('Илья')).toBe('Ильей')
    expect(withName('Bob')).toBe('Bob')
    expect(nextPct(10)).toBe(35)
    expect(nextPct(90)).toBe(100)
    expect(nextPct(100)).toBe(0)
    const view: TGoalsView = {
      internId: 'x',
      agreed: { by: { id: '1', name: 'Олег' }, at: '2026-11-04T10:00:00Z' },
      fromOkr: false,
      goals: [],
      okrs: [],
      canEdit: false,
      canAgree: false,
    }
    expect(agreedLabel(view)).toBe('согласованы с Олегом 4 нояб')
  })
})

describe('Справочник: статья с сервера', () => {
  it('в окне статьи показывается текст целиком', async () => {
    loginAs('intern_anna')
    renderApp('/kb?article=a-leave')
    const dialog = await screen.findByRole('dialog', {}, T)
    expect(await within(dialog).findByText(/Демо-текст статьи/, {}, T)).toBeInTheDocument()
  })

  it('поиск идет на сервер и показывает найденное', async () => {
    loginAs('intern_anna')
    renderApp('/kb')
    await screen.findByText('Популярное', {}, T)
    await userEvent.type(screen.getByLabelText('Поиск по справочнику'), 'отгул')
    expect(await screen.findByRole('button', { name: /Как взять отгул/ }, T)).toBeInTheDocument()
  })
})
