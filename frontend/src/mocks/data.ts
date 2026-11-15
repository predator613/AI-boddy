import type { TestUser, User } from '../schemas/auth'

// Синтетические пользователи, совпадают с seed в backend.
export const MOCK_PASSWORD = 'demo123'

export const mockUsers: User[] = [
  {
    id: '11111111-1111-4111-8111-111111111111',
    login: 'intern_maria',
    name: 'Мария',
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
    login: 'lead_artemiy',
    name: 'Артемий',
    role: 'lead',
    teamId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
    teamName: 'Backend',
    mentorId: null,
  },
  {
    id: '44444444-4444-4444-8444-444444444444',
    login: 'intern_dima',
    name: 'Дмитрий',
    role: 'intern',
    teamId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
    teamName: 'Backend',
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

// Отделы компании – как в миграции 013 backend.
export const MOCK_DEPARTMENTS = [
  {
    code: 'equipment',
    title: 'Отдел оборудования',
    slaHours: 48,
    hours: 'пн–пт 10:00–19:00',
    about: 'Ноутбук, монитор, гарнитура, клавиатура и мышь, пропуск в офис.',
    items: 'Ноутбук, Монитор',
  },
  {
    code: 'it_support',
    title: 'ИТ-поддержка',
    slaHours: 8,
    hours: 'круглосуточно',
    about: 'VPN, корпоративная почта, пароли, двухфакторная аутентификация.',
    items: 'VPN, Сброс пароля',
  },
  {
    code: 'hr',
    title: 'HR-поддержка',
    slaHours: 24,
    hours: 'пн–пт 10:00–19:00',
    about: 'Отпуск, справка с места работы, документы для вуза.',
    items: 'Отпуск, Справка с места работы',
  },
  {
    code: 'accounting',
    title: 'Бухгалтерия',
    slaHours: 72,
    hours: 'пн–пт 10:00–18:00',
    about: 'Справка 2-НДФЛ, вопросы по выплатам.',
    items: 'Справка 2-НДФЛ',
  },
]
