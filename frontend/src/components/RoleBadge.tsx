import { roleLabels, type Role } from '../schemas/auth'
import { StatusTag } from './StatusTag'

// Цвета ролей из палитры кейса: светлая заливка и темный текст.
const roleColors: Record<Role, string> = {
  intern: '#BADAFB',
  mentor: '#DFCDF3',
  lead: '#C9FFBF',
}

export function RoleBadge({ role }: { role: Role }) {
  return <StatusTag bg={roleColors[role]}>{roleLabels[role]}</StatusTag>
}
