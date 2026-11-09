import { Card, Container, Group, Stack, Text, Title } from '@mantine/core'
import { createFileRoute } from '@tanstack/react-router'
import { RoleBadge } from '../../components/RoleBadge'
import { useAuthStore } from '../../store/authStore'

export const Route = createFileRoute('/_auth/')({
  component: HomePage,
})

function HomePage() {
  const user = useAuthStore((s) => s.user)
  if (!user) return null

  return (
    <Container size="md" py="lg">
      <Card withBorder radius="lg" padding="xl">
        <Stack gap="xs">
          <Title order={2}>Привет, {user.name}!</Title>
          <Group gap="xs">
            <Text c="dimmed">Роль:</Text>
            <RoleBadge role={user.role} />
          </Group>
          <Text c="dimmed">
            Команда:{' '}
            <Text span c="dark" fw={500}>
              {user.teamName}
            </Text>
          </Text>
          <Text size="sm" c="dimmed" mt="sm">
            Скоро здесь появятся чат с помощником, «Мой путь» и заявки.
          </Text>
        </Stack>
      </Card>
    </Container>
  )
}
