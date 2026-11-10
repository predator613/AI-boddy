import { Group, Menu, Text, Tooltip } from '@mantine/core'
import { IconBell, IconHeadset, IconLogout, IconUser } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { navItems } from '../lib/nav'
import { roleLabels, type User } from '../schemas/auth'
import { Mascot } from './Mascot'
import classes from './AppRail.module.css'

interface AppRailProps {
  user: User
  onLogout: () => void
}

export function AppRail({ user, onLogout }: AppRailProps) {
  return (
    <nav className={classes.rail} aria-label="Разделы">
      <Link to="/" aria-label="AI-buddy, на главную" style={{ marginBottom: 14, display: 'flex' }}>
        <Mascot size={40} />
      </Link>
      {navItems(user.role).map((item) => (
        <Tooltip key={item.to} label={item.label} position="right" withArrow openDelay={300}>
          <Link
            to={item.to}
            className={classes.item}
            aria-label={item.label}
            activeOptions={{ exact: item.to === '/' }}
            activeProps={{ 'data-status': 'active', 'aria-current': 'page' }}
          >
            <item.icon size={24} stroke={1.8} />
          </Link>
        </Tooltip>
      ))}
      <div className={classes.spacer} />
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
          <Menu.Item
            component={Link}
            to="/profile"
            hash="notifications"
            leftSection={<IconBell size={18} />}
          >
            Уведомления
          </Menu.Item>
          <Menu.Item component={Link} to="/support" leftSection={<IconHeadset size={18} />}>
            Помощь и поддержка
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
