import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { API_URL } from '../src/api/client'
import { server } from '../src/mocks/server'
import { useAuthStore } from '../src/store/authStore'
import { useSettingsStore } from '../src/store/settingsStore'
import { loginAs, renderApp } from './test-utils'

const T = { timeout: 3000 }

describe('Заявки', () => {
  beforeEach(() => loginAs('intern_anna'))

  it('список, фильтры и детали заявки', async () => {
    renderApp('/requests')
    const list = await screen.findByRole('list', { name: 'Список заявок' }, T)
    expect(within(list).getAllByRole('listitem')).toHaveLength(4)
    expect(screen.getByRole('complementary', { name: 'Детали заявки' })).toHaveTextContent(
      'Заявка №1038',
    )

    await userEvent.click(screen.getByRole('tab', { name: /Отклонены/ }))
    expect(
      within(screen.getByRole('list', { name: 'Список заявок' })).getAllByRole('listitem'),
    ).toHaveLength(1)
    await userEvent.click(screen.getByRole('button', { name: /Доступ к логам продакшена/ }))
    expect(await screen.findByText(/Стажерам этот доступ не выдается/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: /Одобрены/ }))
    expect(
      within(screen.getByRole('list', { name: 'Список заявок' })).getAllByRole('listitem'),
    ).toHaveLength(2)
  })

  it('пустой фильтр и ошибка загрузки', async () => {
    server.use(http.get(`${API_URL}/api/requests`, () => HttpResponse.json([])))
    renderApp('/requests')
    expect(await screen.findByText('Здесь пока пусто.', {}, T)).toBeInTheDocument()
  })

  it('ошибка загрузки заявок', async () => {
    server.use(http.get(`${API_URL}/api/requests`, () => HttpResponse.error()))
    renderApp('/requests')
    expect(await screen.findByRole('alert', {}, T)).toHaveTextContent('Сервер недоступен')
  })
})

describe('Справочник', () => {
  beforeEach(() => loginAs('intern_anna'))

  it('разделы, поиск и статья', async () => {
    renderApp('/kb')
    expect(await screen.findByText('Популярное', {}, T)).toBeInTheDocument()

    const sections = screen.getByRole('navigation', { name: 'Разделы справочника' })
    await userEvent.click(within(sections).getByRole('button', { name: /Отпуска и отгулы/ }))
    expect(await screen.findByText('Как взять отгул')).toBeInTheDocument()
    expect(within(sections).getByRole('button', { name: /Отпуска и отгулы/ })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Назад' }))
    expect(await screen.findByText('Популярное')).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Поиск по справочнику'), 'ревью')
    expect(await screen.findByText('Найдено: 1', {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /Как устроено код-ревью в команде/ }))
    expect(await screen.findByRole('dialog')).toHaveTextContent('мин чтения')
    await userEvent.click(within(screen.getByRole('dialog')).getAllByRole('button')[0])

    await userEvent.clear(screen.getByLabelText('Поиск по справочнику'))
    await userEvent.type(screen.getByLabelText('Поиск по справочнику'), 'абракадабра')
    expect(await screen.findByText(/Ничего не нашлось/, {}, T)).toBeInTheDocument()
  })

  it('после «Моего пути» показывает статьи текущего этапа', async () => {
    renderApp('/')
    await screen.findByText('Позади 33% пути адаптации!', {}, T)
    await userEvent.click(screen.getByRole('link', { name: 'Справочник' }))
    expect(await screen.findByText('Часто задаваемые вопросы', {}, T)).toBeInTheDocument()
    expect(screen.getByText('Этап 02')).toBeInTheDocument()
  })

  it('ссылка на статью открывает ее', async () => {
    renderApp('/kb?article=a-leave')
    expect(await screen.findByRole('dialog', {}, T)).toHaveTextContent('Как взять отгул')
  })
})

describe('Поддержка', () => {
  it('частые вопросы и форма', async () => {
    loginAs('intern_anna')
    renderApp('/support')
    expect(await screen.findByText('Кому написать', {}, T)).toBeInTheDocument()
    expect(await screen.findByText('Игорь', {}, T)).toBeInTheDocument()
    expect(screen.getByText('Олег')).toBeInTheDocument()
    expect(screen.getByText('Отдел оборудования')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Какие данные видит AI-buddy?' }))
    expect(await screen.findByText(/Чужие заявки и личные данные/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Отправить' }))
    expect(await screen.findByText('Опиши подробнее, хотя бы пару слов')).toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Подробности'), 'Помощник перепутал сроки')
    await userEvent.click(screen.getByRole('button', { name: 'Отправить' }))
    expect(await screen.findByText(/Сообщение отправлено/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Написать еще' }))
    expect(screen.getByLabelText('Подробности')).toBeInTheDocument()
  })

  it('стажер без наставника видит, что наставник пока не назначен', async () => {
    loginAs('intern_pavel')
    renderApp('/support')
    expect(await screen.findByText('пока не назначен', {}, T)).toBeInTheDocument()
    expect(screen.queryByText('Игорь')).not.toBeInTheDocument()
  })

  it('наставнику показываются только общие службы', async () => {
    loginAs('mentor_igor')
    renderApp('/support')
    expect(await screen.findByText('HR-поддержка', {}, T)).toBeInTheDocument()
    expect(screen.queryByText('руководитель команды')).not.toBeInTheDocument()
  })
})

describe('Профиль', () => {
  it('настройки сохраняются, выход из аккаунта', async () => {
    loginAs('intern_anna')
    renderApp('/profile')
    expect(await screen.findByText('intern_anna', {}, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('switch', { name: 'Подсказки AI-buddy' }))
    expect(useSettingsStore.getState().notifications.tips).toBe(true)
    await userEvent.click(screen.getByText('Подробно'))
    expect(useSettingsStore.getState().answerStyle).toBe('detailed')
    await userEvent.click(screen.getByRole('button', { name: 'Выйти из аккаунта' }))
    expect(await screen.findByText('Вход в AI-buddy')).toBeInTheDocument()
    expect(useAuthStore.getState().token).toBeNull()
  })
})

describe('Наставник', () => {
  beforeEach(() => loginAs('mentor_igor'))

  it('список стажеров и путь стажера со сменой статуса', async () => {
    renderApp('/')
    await userEvent.click(await screen.findByRole('link', { name: /Анна: 33%/ }, T))
    await userEvent.click(await screen.findByRole('button', { name: /Этап 02/ }, T))
    expect(screen.getByText(/как наставник/)).toBeInTheDocument()
    await userEvent.selectOptions(
      screen.getByLabelText('Статус задачи «Получить доступ к GitLab»'),
      'done',
    )
    expect(await screen.findByRole('img', { name: /6 из 15 задач/ }, T)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('link', { name: /Все стажеры/ }))
    expect(await screen.findByRole('heading', { name: 'Мои стажеры' })).toBeInTheDocument()
  })

  it('нет стажеров – пустое состояние; чужой стажер – ошибка', async () => {
    server.use(http.get(`${API_URL}/api/interns`, () => HttpResponse.json([])))
    renderApp('/')
    expect(await screen.findByText(/Пока нет стажеров/, {}, T)).toBeInTheDocument()
  })

  it('чужой стажер – 404', async () => {
    renderApp('/interns/someone')
    expect(await screen.findByRole('alert', {}, T)).toHaveTextContent('Ничего не нашлось')
  })

  it('ошибка списка стажеров', async () => {
    server.use(
      http.get(`${API_URL}/api/interns`, () =>
        HttpResponse.json({ error: { code: 'forbidden', message: 'x' } }, { status: 403 }),
      ),
    )
    renderApp('/')
    expect(await screen.findByRole('alert', {}, T)).toHaveTextContent('недоступен для твоей роли')
  })
})
