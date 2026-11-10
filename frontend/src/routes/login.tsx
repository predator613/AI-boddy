import { Center, Divider, Paper, Stack, Text, Title } from '@mantine/core'
import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router'
import { LoginForm } from '../components/LoginForm'
import { Mascot } from '../components/Mascot'
import { TestUserPicker } from '../components/TestUserPicker'
import { useAuthStore } from '../store/authStore'

export const Route = createFileRoute('/login')({
  beforeLoad: () => {
    if (useAuthStore.getState().token) throw redirect({ to: '/' })
  },
  component: LoginPage,
})

function LoginPage() {
  const navigate = useNavigate()
  const goHome = () => navigate({ to: '/' })

  return (
    <Center mih="100vh" p="md" bg="#F4F4F4">
      <Paper withBorder shadow="sm" radius="lg" p="xl" w="100%" maw={520}>
        <Stack gap="lg">
          <Stack gap={4}>
            <Mascot size={56} glow />
            <Title order={2} fz={28}>
              Вход в AI-buddy
            </Title>
            <Text c="dimmed" size="sm">
              Помощник стажера: процессы, этапы адаптации и заявки в одном месте
            </Text>
          </Stack>

          <TestUserPicker onSuccess={goHome} />

          <Divider label="или войдите по логину" labelPosition="center" />

          <LoginForm onSuccess={goHome} />

          <Text size="xs" c="dimmed" ta="center">
            Демо-версия. Все данные синтетические, пароль тестовых пользователей – demo123
          </Text>
        </Stack>
      </Paper>
    </Center>
  )
}
