import { useAuthStore } from '../src/store/authStore'
import { mockUsers } from '../src/mocks/data'

const anna = mockUsers[0]

describe('authStore', () => {
  it('setAuth сохраняет токен и пользователя, в том числе в localStorage', () => {
    useAuthStore.getState().setAuth('t-1', anna)
    expect(useAuthStore.getState().token).toBe('t-1')
    expect(useAuthStore.getState().user?.login).toBe('intern_anna')
    expect(localStorage.getItem('ai-buddy-auth')).toContain('t-1')
  })

  it('setUser обновляет только пользователя', () => {
    useAuthStore.getState().setAuth('t-1', anna)
    useAuthStore.getState().setUser({ ...anna, name: 'Анна Н.' })
    expect(useAuthStore.getState().token).toBe('t-1')
    expect(useAuthStore.getState().user?.name).toBe('Анна Н.')
  })

  it('logout очищает токен и пользователя', () => {
    useAuthStore.getState().setAuth('t-1', anna)
    useAuthStore.getState().logout()
    expect(useAuthStore.getState().token).toBeNull()
    expect(useAuthStore.getState().user).toBeNull()
  })
})
