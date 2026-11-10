import type { TestUser, User } from '../schemas/auth'

// Синтетические пользователи, совпадают с seed в backend.
export const MOCK_PASSWORD = 'demo123'

export const mockUsers: User[] = [
  {
    id: '11111111-1111-4111-8111-111111111111',
    login: 'intern_anna',
    name: 'Анна',
    role: 'intern',
    teamId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
    teamName: 'Backend',
    mentorId: '22222222-2222-4222-8222-222222222222',
  },
  {
    id: '22222222-2222-4222-8222-222222222222',
    login: 'mentor_igor',
    name: 'Игорь',
    role: 'mentor',
    teamId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
    teamName: 'Backend',
    mentorId: null,
  },
  {
    id: '33333333-3333-4333-8333-333333333333',
    login: 'lead_oleg',
    name: 'Олег',
    role: 'lead',
    teamId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
    teamName: 'Backend',
    mentorId: null,
  },
  {
    id: '44444444-4444-4444-8444-444444444444',
    login: 'intern_pavel',
    name: 'Павел',
    role: 'intern',
    teamId: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',
    teamName: 'Frontend',
    mentorId: null,
  },
]

export const mockTestUsers: TestUser[] = mockUsers.map(({ login, name, role, teamName }) => ({
  login,
  name,
  role,
  teamName,
}))

export const mockToken = (login: string) => `mock-token-${login}`
