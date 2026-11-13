import {
  IconBook2,
  IconChecklist,
  IconTarget,
  IconCalendarEvent,
  IconFileText,
  IconFiles,
  IconHeadset,
  IconListCheck,
  IconMessageCircle,
  IconMessageUser,
  IconMessages,
  IconRoute,
  IconUsers,
} from '@tabler/icons-react'
import type { ComponentType } from 'react'
import type { User } from '../schemas/auth'

export type NavPath =
  | '/'
  | '/chat'
  | '/mentor'
  | '/calendar'
  | '/kb'
  | '/requests'
  | '/support'
  | '/inbox'
  | '/plan'
  | '/materials'
  | '/schedule'
  | '/approvals'

interface NavItem {
  to: NavPath
  label: string
  icon: ComponentType<{ size?: number; stroke?: number }>
}

const KB: NavItem = { to: '/kb', label: 'Справочник', icon: IconBook2 }
const SUPPORT: NavItem = { to: '/support', label: 'Поддержка', icon: IconHeadset }

// Пункты меню зависят от роли: недоступное не показываем.
export function navItems(role: User['role']): NavItem[] {
  if (role === 'intern') {
    return [
      { to: '/', label: 'Мой путь', icon: IconRoute },
      { to: '/chat', label: 'Чат с помощником', icon: IconMessageCircle },
      { to: '/mentor', label: 'Чат с наставником', icon: IconMessageUser },
      { to: '/calendar', label: 'Календарь', icon: IconCalendarEvent },
      KB,
      { to: '/requests', label: 'Мои заявки', icon: IconFileText },
      SUPPORT,
    ]
  }
  // Руководитель: цели стажеров, согласования заявок и поддержка на случай сбоев.
  if (role === 'lead') {
    return [
      { to: '/', label: 'Цели стажеров', icon: IconTarget },
      { to: '/approvals', label: 'Согласования', icon: IconChecklist },
      SUPPORT,
    ]
  }
  // Наставник: кабинет наставника.
  return [
    { to: '/', label: 'Мои стажеры', icon: IconUsers },
    { to: '/inbox', label: 'Сообщения', icon: IconMessages },
    { to: '/plan', label: 'План адаптации', icon: IconListCheck },
    { to: '/materials', label: 'Материалы и справочник', icon: IconFiles },
    { to: '/schedule', label: 'Расписание', icon: IconCalendarEvent },
    SUPPORT,
  ]
}
