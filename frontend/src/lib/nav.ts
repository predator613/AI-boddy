import {
  IconBook2,
  IconFileText,
  IconHeadset,
  IconMessageCircle,
  IconRoute,
  IconUsers,
} from '@tabler/icons-react'
import type { ComponentType } from 'react'
import type { User } from '../schemas/auth'

interface NavItem {
  to: '/' | '/chat' | '/kb' | '/requests' | '/support'
  label: string
  icon: ComponentType<{ size?: number; stroke?: number }>
}

// Пункты меню зависят от роли: недоступное не показываем.
export function navItems(role: User['role']): NavItem[] {
  const home: NavItem =
    role === 'intern'
      ? { to: '/', label: 'Мой путь', icon: IconRoute }
      : { to: '/', label: 'Мои стажеры', icon: IconUsers }
  const items: NavItem[] = [
    home,
    { to: '/chat', label: 'Чат с помощником', icon: IconMessageCircle },
    { to: '/kb', label: 'Справочник', icon: IconBook2 },
  ]
  if (role === 'intern') items.push({ to: '/requests', label: 'Мои заявки', icon: IconFileText })
  items.push({ to: '/support', label: 'Поддержка', icon: IconHeadset })
  return items
}
