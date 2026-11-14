import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { LoginForm } from '../src/components/LoginForm'
import { API_URL } from '../src/api/client'
import { server } from '../src/mocks/server'
import { mockToken } from '../src/mocks/data'
import { useAuthStore } from '../src/store/authStore'
import { renderWithProviders } from './test-utils'

async function fillAndSubmit(login: string, password: string) {
  const user = userEvent.setup()
  if (login) await user.type(screen.getByLabelText('Логин'), login)
  if (password) await user.type(screen.getByLabelText('Пароль'), password)
  await user.click(screen.getByRole('button', { name: 'Войти' }))
}

describe('LoginForm', () => {
  it('пустые поля – ошибки валидации, запрос не отправляется', async () => {
    renderWithProviders(<LoginForm />)
    await fillAndSubmit('', '')
    expect(await screen.findByText('Введите логин')).toBeInTheDocument()
    expect(screen.getByText('Введите пароль')).toBeInTheDocument()
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('неверный пароль (401) – «Неверный логин или пароль»', async () => {
    renderWithProviders(<LoginForm />)
    await fillAndSubmit('intern_maria', 'wrong')
    expect(await screen.findByRole('alert')).toHaveTextContent('Неверный логин или пароль')
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('много неудачных попыток (429) – просим подождать', async () => {
    server.use(
      http.post(`${API_URL}/api/auth/login`, () =>
        HttpResponse.json(
          { error: { code: 'too_many_attempts', message: 'too many login attempts' } },
          { status: 429 },
        ),
      ),
    )
    renderWithProviders(<LoginForm />)
    await fillAndSubmit('intern_maria', 'demo123')
    expect(await screen.findByRole('alert')).toHaveTextContent('Подождите 5 минут')
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('успешный вход – токен в store и вызов onSuccess', async () => {
    const onSuccess = vi.fn()
    renderWithProviders(<LoginForm onSuccess={onSuccess} />)
    await fillAndSubmit('  intern_maria  ', 'demo123')
    await vi.waitFor(() => expect(onSuccess).toHaveBeenCalled())
    expect(useAuthStore.getState().token).toBe(mockToken('intern_maria'))
    expect(useAuthStore.getState().user?.name).toBe('Мария')
  })

  it('сервер недоступен – понятный текст ошибки', async () => {
    server.use(http.post(`${API_URL}/api/auth/login`, () => HttpResponse.error()))
    renderWithProviders(<LoginForm />)
    await fillAndSubmit('intern_maria', 'demo123')
    expect(await screen.findByRole('alert')).toHaveTextContent('Сервер недоступен')
  })

  it('прочие ошибки – общий текст', async () => {
    server.use(
      http.post(`${API_URL}/api/auth/login`, () => new HttpResponse(null, { status: 500 })),
    )
    renderWithProviders(<LoginForm />)
    await fillAndSubmit('intern_maria', 'demo123')
    expect(await screen.findByRole('alert')).toHaveTextContent('Не удалось войти')
  })
})
