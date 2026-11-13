import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { API_URL } from '../src/api/client'
import { glassBackground, stageWindow, tickColors, weekBars } from '../src/lib/pathLook'
import { resetMentorMocks } from '../src/mocks/mentorData'
import { server } from '../src/mocks/server'
import { loginAs, renderApp } from './test-utils'

const T = { timeout: 3000 }

// Среда, 11 ноября 2026, 13:20: неделя 9–13 ноября, 1:1 с наставником в четверг.
beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date(2026, 10, 11, 13, 20))
  resetMentorMocks()
})
afterEach(() => vi.useRealTimers())

describe('Календарь стажера', () => {
  it('в меню стажера есть календарь, наставника туда не пускает', async () => {
    loginAs('mentor_igor')
    const router = renderApp('/calendar')
    await screen.findByRole('heading', { name: 'Мои стажеры' }, T)
    expect(router.state.location.pathname).toBe('/')
  })

  it('показывает неделю, встречи и отвечает «Приду»', async () => {
    loginAs('intern_anna')
    renderApp('/')
    await userEvent.click(await screen.findByRole('link', { name: 'Календарь' }, T))
    expect(await screen.findByRole('heading', { name: 'Мои встречи' }, T)).toBeInTheDocument()
    expect(
      await screen.findByRole('button', { name: /Встреча-знакомство с командой, пн 11:00/ }, T),
    ).toBeInTheDocument()
    // первая встреча, которая ждет ответа, выбрана сразу
    const details = await screen.findByRole('heading', { name: '1:1 с Игорем' }, T)
    expect(details).toBeInTheDocument()
    expect(screen.getByText('ждет ответа')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Приду' }))
    expect(await screen.findByText(/Отметила, что ты придешь/, {}, T)).toBeInTheDocument()
    expect(await screen.findByText('ты придешь', {}, T)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /Встреча-знакомство с командой/ }))
    expect(await screen.findByText('прошла')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Приду' })).not.toBeInTheDocument()
  })

  it('просьба перенести уходит наставнику в чат', async () => {
    loginAs('intern_anna')
    renderApp('/calendar')
    await screen.findByRole('heading', { name: '1:1 с Игорем' }, T)
    await userEvent.click(screen.getByRole('button', { name: 'Попросить перенести' }))
    await userEvent.click(screen.getByRole('button', { name: 'Не успею подготовиться' }))
    expect(await screen.findByText(/AI-buddy написал наставнику/, {}, T)).toBeInTheDocument()
    expect(await screen.findByText('запрошен перенос', {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('link', { name: 'Чат с наставником' }))
    expect(
      await screen.findByText(/Можно перенести «1:1 с тобой»\? Не успею подготовиться/, {}, T),
    ).toBeInTheDocument()
  })

  it('своя запись: проверка полей, пересечение, добавление и удаление', async () => {
    loginAs('intern_anna')
    renderApp('/calendar')
    await screen.findByRole('heading', { name: '1:1 с Игорем' }, T)
    await userEvent.click(screen.getByRole('button', { name: 'Добавить запись' }))
    const dialog = await screen.findByRole('dialog', {}, T)
    await userEvent.click(within(dialog).getByRole('button', { name: 'Добавить' }))
    expect(await within(dialog).findByText('Напиши, что планируешь')).toBeInTheDocument()

    await userEvent.type(within(dialog).getByLabelText('Что планируешь'), 'Учу Go')
    await userEvent.selectOptions(within(dialog).getByLabelText('Начало'), '17.5')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Добавить' }))
    expect(await within(dialog).findByText(/выходит за рабочий день/)).toBeInTheDocument()

    // ср 14:00 – знакомство с DevOps
    await userEvent.selectOptions(within(dialog).getByLabelText('Начало'), '14')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Добавить' }))
    expect(
      await within(dialog).findByText(/Пересекается с «Знакомство с DevOps»/, {}, T),
    ).toBeInTheDocument()

    await userEvent.selectOptions(within(dialog).getByLabelText('Начало'), '16')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Учеба' }))
    await userEvent.click(within(dialog).getByLabelText('Показать наставнику в его расписании'))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Добавить' }))
    expect(await screen.findByText('Запись добавлена в твой календарь', {}, T)).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'Учу Go' }, T)).toBeInTheDocument()
    expect(screen.getByText('только у тебя')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Удалить запись' }))
    expect(await screen.findByText('Запись удалена', {}, T)).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: /Учу Go/ })).not.toBeInTheDocument(),
    )
  })

  it('листает недели и возвращается к текущей', async () => {
    loginAs('intern_anna')
    renderApp('/calendar')
    await screen.findByRole('heading', { name: '1:1 с Игорем' }, T)
    expect(screen.getByRole('button', { name: 'Эта неделя' })).toBeDisabled()
    await userEvent.click(screen.getByRole('button', { name: 'Следующая неделя' }))
    expect(await screen.findByText(/На этой неделе встреч нет/, {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Прошлая неделя' }))
    await userEvent.click(screen.getByRole('button', { name: 'Прошлая неделя' }))
    await userEvent.click(screen.getByRole('button', { name: 'Эта неделя' }))
    expect(await screen.findByRole('heading', { name: '1:1 с Игорем' }, T)).toBeInTheDocument()
  })

  it('ошибка загрузки календаря', async () => {
    server.use(
      http.get(`${API_URL}/api/me/calendar`, () =>
        HttpResponse.json({ error: { code: 'internal_error', message: 'x' } }, { status: 500 }),
      ),
    )
    loginAs('intern_anna')
    renderApp('/calendar')
    expect(await screen.findByRole('button', { name: 'Повторить' }, T)).toBeInTheDocument()
  })
})

describe('чат с наставником закреплен как мессенджер', () => {
  it('фильтр «Только изменения» и поле ввода внизу', async () => {
    loginAs('intern_anna')
    renderApp('/mentor')
    const log = await screen.findByRole('log', { name: 'Переписка с наставником' }, T)
    expect(within(log).getByText(/для GitLab нужна роль/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('switch', { name: 'Только изменения от наставника' }))
    expect(within(log).queryByText(/для GitLab нужна роль/)).not.toBeInTheDocument()
    expect(screen.getByLabelText('Сообщение наставнику')).toBeInTheDocument()
  })
})

describe('вид нового «Моего пути»', () => {
  it('стеклянный фон, штрихи кольца, окно этапов и шкала недель', () => {
    expect(glassBackground('#c9ffbf')).toContain('#9DF5A8')
    expect(glassBackground('#123456')).toContain('#123456')
    const colors = tickColors(
      [{ tasks: [{ status: 'done' }, { status: 'todo' }, { status: 'waiting' }] }] as never,
      6,
    )
    expect(colors).toEqual(['#04E061', '#04E061', '#00AAFF', '#00AAFF', '#E4E4E4', '#E4E4E4'])
    expect(tickColors([], 3)).toEqual(['#E4E4E4', '#E4E4E4', '#E4E4E4'])
    expect(stageWindow(4, 1)).toEqual([0, 4])
    expect(stageWindow(6, 4)).toEqual([1, 5])
    expect(stageWindow(6, 5)).toEqual([2, 6])
    const bars = weekBars(2, 24, [4, 12])
    expect(bars[0]!.color).toBe('#04E061')
    expect(bars[1]).toMatchObject({ height: 14, color: '#00AAFF' })
    expect(bars[3]!.height).toBe(10)
    expect(bars[5]!.height).toBe(6)
  })
})
