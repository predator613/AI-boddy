import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { API_URL } from '../src/api/client'
import { server } from '../src/mocks/server'
import { mockToken, mockUsers } from '../src/mocks/data'
import { useAuthStore } from '../src/store/authStore'
import { renderApp } from './test-utils'

const anna = mockUsers[0]
const loginAsAnna = () => useAuthStore.setState({ token: mockToken(anna.login), user: anna })

describe('маршруты', () => {
  it('без токена защищенная страница перенаправляет на /login', async () => {
    const router = renderApp('/')
    expect(await screen.findByText('Вход в AI-buddy')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
  })

  it('с токеном на / показывается приветствие', async () => {
    loginAsAnna()
    renderApp('/')
    expect(await screen.findByText('Привет, Анна!')).toBeInTheDocument()
    expect(screen.getByText('Backend')).toBeInTheDocument()
  })

  it('вошедшего пользователя /login отправляет на главную', async () => {
    loginAsAnna()
    const router = renderApp('/login')
    expect(await screen.findByText('Привет, Анна!')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/')
  })

  it('вход через форму ведет на главную', async () => {
    const router = renderApp('/login')
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Логин'), 'intern_pavel')
    await user.type(screen.getByLabelText('Пароль'), 'demo123')
    await user.click(screen.getByRole('button', { name: 'Войти' }))
    expect(await screen.findByText('Привет, Павел!')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/')
  })

  it('вход через карточку тестового пользователя ведет на главную', async () => {
    renderApp('/login')
    await userEvent.click(await screen.findByRole('button', { name: 'Войти как Олег' }))
    expect(await screen.findByText('Привет, Олег!')).toBeInTheDocument()
  })

  it('кнопка «Выйти» возвращает на /login', async () => {
    loginAsAnna()
    const router = renderApp('/')
    await userEvent.click(await screen.findByRole('button', { name: 'Выйти' }))
    expect(await screen.findByText('Вход в AI-buddy')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('если /me вернул 401 (токен протух) – выход и редирект на /login', async () => {
    useAuthStore.setState({ token: 'expired', user: anna })
    const router = renderApp('/')
    expect(await screen.findByText('Вход в AI-buddy')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
  })

  it('данные пользователя обновляются ответом /me', async () => {
    server.use(
      http.get(`${API_URL}/api/auth/me`, () => HttpResponse.json({ ...anna, name: 'Анна Н.' })),
    )
    loginAsAnna()
    renderApp('/')
    expect(await screen.findByText('Привет, Анна Н.!')).toBeInTheDocument()
  })
})
