import { Group, Menu, Text, Tooltip } from '@mantine/core'
import { IconLogout, IconUser } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Fragment } from 'react'
import { getRequests } from '../api/assistant'
import { navItems } from '../lib/nav'
import { roleLabels, type User } from '../schemas/auth'
import { Mascot } from './Mascot'
import classes from './AppRail.module.css'

interface AppRailProps {
  user: User
  onLogout: () => void
}

export function AppRail({ user, onLogout }: AppRailProps) {
  // Счетчик на «Мои заявки»: сколько заявок еще ждут решения.
  const requests = useQuery({
    queryKey: ['requests'],
    queryFn: getRequests,
    enabled: user.role === 'intern',
  })
  const waiting = requests.data?.filter((r) => r.status === 'waiting').length ?? 0
  const items = navItems(user.role)
  return (
    <nav className={classes.rail} aria-label="Разделы">
      <Link to="/" aria-label="AI-buddy, на главную" style={{ marginBottom: 14, display: 'flex' }}>
        <Mascot size={40} />
      </Link>
      {items.map((item) => (
        <Fragment key={item.to}>
          {/* Поддержка стоит отдельно, внизу у аватара: так она не теряется среди разделов */}
          {item.to === '/support' && <div className={classes.spacer} />}
          <Tooltip label={item.label} position="right" withArrow openDelay={300}>
            <Link
              to={item.to}
              className={classes.item}
              aria-label={item.label}
              activeOptions={{ exact: item.to === '/' }}
              activeProps={{ 'data-status': 'active', 'aria-current': 'page' }}
            >
              <item.icon size={24} stroke={1.8} />
              {item.to === '/requests' && waiting > 0 && (
                <span className={classes.badge} aria-label={`ждут решения: ${waiting}`}>
                  {waiting}
                </span>
              )}
            </Link>
          </Tooltip>
        </Fragment>
      ))}
      {!items.some((i) => i.to === '/support') && <div className={classes.spacer} />}
      <Menu position="right-end" offset={12} width={260} shadow="md">
        <Menu.Target>
          <button type="button" className={classes.avatar} aria-label="Меню профиля">
            {user.name.charAt(0)}
          </button>
        </Menu.Target>
        <Menu.Dropdown>
          <Group gap={10} p="xs" wrap="nowrap">
            <span
              className={classes.avatar}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flex: 'none',
              }}
            >
              {user.name.charAt(0)}
            </span>
            <div>
              <Text size="sm" fw={500}>
                {user.name}
              </Text>
              <Text size="xs" c="ink.5">
                {roleLabels[user.role]} · {user.teamName}
              </Text>
            </div>
          </Group>
          <Menu.Divider />
          <Menu.Item component={Link} to="/profile" leftSection={<IconUser size={18} />}>
            Профиль и настройки
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item leftSection={<IconLogout size={18} />} onClick={onLogout}>
            Выйти
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>
    </nav>
  )
}
