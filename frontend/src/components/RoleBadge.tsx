import { Badge } from '@mantine/core'
import { roleLabels, type Role } from '../schemas/auth'

const roleColors: Record<Role, string> = {
  intern: 'blue',
  mentor: 'teal',
  lead: 'grape',
}

export function RoleBadge({ role }: { role: Role }) {
  return (
    <Badge color={roleColors[role]} variant="light" radius="sm">
      {roleLabels[role]}
    </Badge>
  )
}
