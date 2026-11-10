import { Box } from '@mantine/core'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Navigate, Outlet, createFileRoute, redirect } from '@tanstack/react-router'
import { useEffect } from 'react'
import { getMe } from '../api/auth'
import { AppRail } from '../components/AppRail'
import { useAuthStore } from '../store/authStore'
import classes from './_auth.module.css'

// Layout всех страниц, доступных только после входа: левая полоса-меню и контент.
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

  if (!token || !user) return <Navigate to="/login" />

  const handleLogout = () => {
    logout()
    queryClient.clear()
  }

  return (
    <div className={classes.shell}>
      <AppRail user={user} onLogout={handleLogout} />
      <Box component="main" className={classes.main}>
        <Outlet />
      </Box>
    </div>
  )
}
