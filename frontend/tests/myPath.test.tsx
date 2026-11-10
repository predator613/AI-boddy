import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { API_URL } from '../src/api/client'
import { server } from '../src/mocks/server'
import { loginAs, renderApp } from './test-utils'

const T = { timeout: 3000 }

describe('Мой путь', () => {
  beforeEach(() => loginAs('intern_anna'))

  it('показывает кольцо прогресса, этапы, следующий шаг и наставника', async () => {
    renderApp('/')
    expect(
      await screen.findByRole('img', { name: /Пройдено 33% адаптации: 5 из 15 задач/ }, T),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Этап 02: Доступы, 2 из 4/ })).toBeInTheDocument()
    expect(screen.getByText('Следующий шаг')).toBeInTheDocument()
    expect(screen.getAllByText('Получить доступ к GitLab').length).toBeGreaterThan(0)
    expect(screen.getByText('Игорь · твой наставник')).toBeInTheDocument()
    expect(screen.getByText(/ты на этапе «Доступы»/)).toBeInTheDocument()
  })

  it('этап раскрывается: задачи, доступы справа, совет; «Все этапы» сворачивает', async () => {
    const router = renderApp('/')
    await userEvent.click(await screen.findByRole('button', { name: /Этап 02: Доступы/ }, T))
    const list = await screen.findByRole('list', { name: 'Задачи этапа Доступы' })
    expect(within(list).getAllByRole('listitem')).toHaveLength(4)
    expect(screen.getByText('Статус доступов')).toBeInTheDocument()
    expect(screen.getByText('Совет')).toBeInTheDocument()
    expect(router.state.location.search).toEqual({ stage: '02' })

    await userEvent.click(screen.getByRole('tab', { name: /Этап 01/ }))
    expect(await screen.findByText('Твоя команда')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Все этапы' }))
    expect(await screen.findByText('Этапы адаптации')).toBeInTheDocument()
  })

  it('«Уже сделано» отмечает задачу и двигает следующий шаг', async () => {
    renderApp('/?stage=02')
    await userEvent.click(await screen.findByRole('button', { name: 'Уже сделано' }, T))
    expect(await screen.findByRole('img', { name: /6 из 15 задач/ }, T)).toBeInTheDocument()
  })

  it('«Открыть этап» в карточке следующего шага раскрывает этап', async () => {
    renderApp('/')
    await userEvent.click(await screen.findByRole('button', { name: 'Открыть этап' }, T))
    expect(await screen.findByText('Статус доступов')).toBeInTheDocument()
  })

  it('«Сделать с AI-buddy» открывает чат с заявкой, после подтверждения она видна в заявках', async () => {
    const router = renderApp('/')
    await userEvent.click(await screen.findByRole('button', { name: 'Сделать с AI-buddy' }, T))
    expect(router.state.location.pathname).toBe('/chat')
    expect(await screen.findByText('Создать заявку на доступ?', {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Подтвердить' }))
    expect(await screen.findByText('Заявка №1042 создана', {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('link', { name: 'Открыть заявку' }))
    expect(await screen.findByRole('heading', { name: 'Заявки' })).toBeInTheDocument()
    expect((await screen.findAllByText('Доступ к GitLab', {}, T)).length).toBeGreaterThan(0)
  })

  it('вопрос из панели помощника уходит в чат', async () => {
    const router = renderApp('/')
    await userEvent.click(await screen.findByRole('button', { name: 'Кто выдает доступы?' }, T))
    expect(router.state.location.pathname).toBe('/chat')
    await userEvent.type(await screen.findByLabelText('Сообщение'), 'x')
  })

  it('свой вопрос из панели помощника', async () => {
    const router = renderApp('/')
    await userEvent.type(
      await screen.findByLabelText('Сообщение помощнику', {}, T),
      'Что такое код-ревью?{enter}',
    )
    expect(router.state.location.pathname).toBe('/chat')
    expect(await screen.findByText(/ревьюер оставляет комментарии/i, {}, T)).toBeInTheDocument()
  })

  it('ошибка загрузки – сообщение и «Повторить»', async () => {
    server.use(
      http.get(`${API_URL}/api/path`, () => new HttpResponse(null, { status: 500 }), {
        once: true,
      }),
    )
    renderApp('/')
    expect(await screen.findByRole('alert', {}, T)).toHaveTextContent('Не удалось загрузить')
    await userEvent.click(screen.getByRole('button', { name: 'Повторить' }))
    expect(await screen.findByText('Прогресс адаптации', {}, T)).toBeInTheDocument()
  })
})
