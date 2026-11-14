import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { API_URL } from '../src/api/client'
import { server } from '../src/mocks/server'
import { mockUsers } from '../src/mocks/data'
import { useAuthStore } from '../src/store/authStore'
import { loginAs, renderApp } from './test-utils'

const greeting = (name: string) =>
  screen.findByRole('heading', { name: `Привет, ${name}!` }, { timeout: 3000 })

describe('маршруты', () => {
  it('без токена защищенная страница перенаправляет на /login', async () => {
    const router = renderApp('/')
    expect(await screen.findByText('Вход в AI-buddy')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
  })

  it('стажер на / видит «Мой путь»', async () => {
    loginAs('intern_maria')
    renderApp('/')
    expect(await greeting('Мария')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Мой путь' })).toHaveAttribute('aria-current', 'page')
  })

  it('вошедшего пользователя /login отправляет на главную', async () => {
    loginAs('intern_maria')
    const router = renderApp('/login')
    expect(await greeting('Мария')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/')
  })

  it('вход через форму ведет на главную', async () => {
    const router = renderApp('/login')
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Логин'), 'intern_pavel')
    await user.type(screen.getByLabelText('Пароль'), 'demo123')
    await user.click(screen.getByRole('button', { name: 'Войти' }))
    expect(await greeting('Павел')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/')
  })

  it('руководитель видит цели стажеров и согласования, а не «Мой путь»', async () => {
    renderApp('/login')
    await userEvent.click(await screen.findByRole('button', { name: 'Войти как Артемий' }))
    expect(await screen.findByRole('heading', { name: 'Цели стажеров' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Согласования' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Мои заявки' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Сообщения' })).not.toBeInTheDocument()
  })

  it('«Выйти» в меню профиля возвращает на /login', async () => {
    loginAs('intern_maria')
    const router = renderApp('/')
    await greeting('Мария')
    // на медленном компьютере страница может перерисоваться и закрыть меню – открываем, пока не появится пункт
    const logout = await waitFor(
      async () => {
        if (!screen.queryByRole('menuitem', { name: 'Выйти' }))
          await userEvent.click(screen.getByRole('button', { name: 'Меню профиля' }))
        return screen.getByRole('menuitem', { name: 'Выйти' })
      },
      { timeout: 10000, interval: 300 },
    )
    await userEvent.click(logout)
    expect(await screen.findByText('Вход в AI-buddy')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('если /me вернул 401 (токен протух) – выход и редирект на /login', async () => {
    useAuthStore.setState({ token: 'expired', user: mockUsers[0] })
    const router = renderApp('/')
    expect(await screen.findByText('Вход в AI-buddy', {}, { timeout: 3000 })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
  })

  it('данные пользователя обновляются ответом /me', async () => {
    server.use(
      http.get(`${API_URL}/api/auth/me`, () =>
        HttpResponse.json({ ...mockUsers[0], name: 'Мария Н.' }),
      ),
    )
    loginAs('intern_maria')
    renderApp('/profile')
    expect(await screen.findByText('Мария Н.')).toBeInTheDocument()
  })

  it('заявки недоступны наставнику: редирект на главную', async () => {
    loginAs('mentor_igor')
    const router = renderApp('/requests')
    expect(await screen.findByRole('heading', { name: 'Мои стажеры' })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/')
  })
})
