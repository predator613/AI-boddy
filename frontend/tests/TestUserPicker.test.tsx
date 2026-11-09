import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { TestUserPicker } from '../src/components/TestUserPicker'
import { API_URL } from '../src/api/client'
import { server } from '../src/mocks/server'
import { mockToken } from '../src/mocks/data'
import { useAuthStore } from '../src/store/authStore'
import { renderWithProviders } from './test-utils'

describe('TestUserPicker', () => {
  it('показывает карточки тестовых пользователей с ролями', async () => {
    renderWithProviders(<TestUserPicker />)
    expect(await screen.findByText('Анна')).toBeInTheDocument()
    expect(screen.getByText('Павел')).toBeInTheDocument()
    expect(screen.getByText('Руководитель')).toBeInTheDocument()
  })

  it('клик по карточке выполняет вход', async () => {
    const onSuccess = vi.fn()
    renderWithProviders(<TestUserPicker onSuccess={onSuccess} />)
    await userEvent.click(await screen.findByRole('button', { name: 'Войти как Игорь' }))
    await vi.waitFor(() => expect(onSuccess).toHaveBeenCalled())
    expect(useAuthStore.getState().token).toBe(mockToken('mentor_igor'))
  })

  it('ошибка входа показывается под карточками', async () => {
    server.use(
      http.post(`${API_URL}/api/auth/login`, () =>
        HttpResponse.json(
          { error: { code: 'invalid_credentials', message: 'x' } },
          { status: 401 },
        ),
      ),
    )
    renderWithProviders(<TestUserPicker />)
    await userEvent.click(await screen.findByRole('button', { name: 'Войти как Анна' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Неверный логин или пароль')
  })

  it('не демо-режим (404) – блок скрыт', async () => {
    server.use(
      http.get(`${API_URL}/api/auth/test-users`, () =>
        HttpResponse.json({ error: { code: 'not_found', message: 'x' } }, { status: 404 }),
      ),
    )
    renderWithProviders(<TestUserPicker />)
    await vi.waitFor(() => expect(screen.queryByLabelText(/Загрузка/)).not.toBeInTheDocument())
    expect(screen.queryByText('Быстрый вход под тестовым пользователем')).not.toBeInTheDocument()
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })

  it('backend недоступен – блок скрыт', async () => {
    server.use(http.get(`${API_URL}/api/auth/test-users`, () => HttpResponse.error()))
    renderWithProviders(<TestUserPicker />)
    await vi.waitFor(() => expect(screen.queryByLabelText(/Загрузка/)).not.toBeInTheDocument())
    expect(screen.queryByText('Быстрый вход под тестовым пользователем')).not.toBeInTheDocument()
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })
})
