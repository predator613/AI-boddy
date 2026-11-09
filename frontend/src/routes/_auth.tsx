import { AppShell, Button, Group, Text } from '@mantine/core'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Navigate, Outlet, createFileRoute, redirect } from '@tanstack/react-router'
import { useEffect } from 'react'
import { getMe } from '../api/auth'
import { RoleBadge } from '../components/RoleBadge'
import { useAuthStore } from '../store/authStore'

// Layout всех страниц, доступных только после входа.
export const Route = createFileRoute('/_auth')({
  beforeLoad: () => {
    if (!useAuthStore.getState().token) throw redirect({ to: '/login' })
  },
  component: AuthLayout,
})

function AuthLayout() {
  const token = useAuthStore((s) => s.token)
  const user = useAuthStore((s) => s.user)
  const setUser = useAuthStore((s) => s.setUser)
  const logout = useAuthStore((s) => s.logout)
  const queryClient = useQueryClient()

  // сверяем пользователя с backend; при 401 api-клиент сам сделает logout
  const meQuery = useQuery({ queryKey: ['me', token], queryFn: getMe, enabled: Boolean(token) })
  useEffect(() => {
    if (meQuery.data) setUser(meQuery.data)
  }, [meQuery.data, setUser])

  if (!token) return <Navigate to="/login" />

  const handleLogout = () => {
    logout()
    queryClient.clear()
  }

  return (
    <AppShell header={{ height: 60 }} padding="md">
      <AppShell.Header px="md">
        <Group h="100%" justify="space-between">
          <Text fw={700} size="lg">
            AI-buddy
          </Text>
          <Group gap="sm">
            {user && (
              <Group gap={8} visibleFrom="xs">
                <Text size="sm">{user.name}</Text>
                <RoleBadge role={user.role} />
              </Group>
            )}
            <Button variant="default" size="xs" onClick={handleLogout}>
              Выйти
            </Button>
          </Group>
        </Group>
      </AppShell.Header>
      <AppShell.Main bg="gray.0">
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}
