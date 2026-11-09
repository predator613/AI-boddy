import {
  Alert,
  Avatar,
  Group,
  Loader,
  SimpleGrid,
  Skeleton,
  Stack,
  Text,
  UnstyledButton,
} from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { getTestUsers } from '../api/auth'
import { loginErrorText, useLogin } from '../hooks/useLogin'
import type { TestUser } from '../schemas/auth'
import { RoleBadge } from './RoleBadge'
import classes from './TestUserPicker.module.css'

export const DEMO_PASSWORD = 'demo123'

interface TestUserPickerProps {
  onSuccess?: () => void
}

export function TestUserPicker({ onSuccess }: TestUserPickerProps) {
  const usersQuery = useQuery({ queryKey: ['test-users'], queryFn: getTestUsers, retry: false })
  const mutation = useLogin(onSuccess)

  if (usersQuery.isPending) {
    return (
      <SimpleGrid
        cols={{ base: 1, xs: 2 }}
        spacing="sm"
        aria-label="Загрузка тестовых пользователей"
      >
        {[0, 1, 2, 3].map((i) => (
          <Skeleton key={i} height={76} radius="md" />
        ))}
      </SimpleGrid>
    )
  }

  // нет демо-режима или backend недоступен – блок просто не показываем
  if (usersQuery.isError || usersQuery.data.length === 0) return null

  const pick = (user: TestUser) => mutation.mutate({ login: user.login, password: DEMO_PASSWORD })

  return (
    <Stack gap="sm">
      <Text size="sm" c="dimmed">
        Быстрый вход под тестовым пользователем
      </Text>
      <SimpleGrid cols={{ base: 1, xs: 2 }} spacing="sm">
        {usersQuery.data.map((user) => {
          const isLoading = mutation.isPending && mutation.variables?.login === user.login
          return (
            <UnstyledButton
              key={user.login}
              className={classes.card}
              onClick={() => pick(user)}
              disabled={mutation.isPending}
              aria-label={`Войти как ${user.name}`}
            >
              <Group wrap="nowrap" gap="sm">
                <Avatar color="blue" radius="xl">
                  {user.name.charAt(0)}
                </Avatar>
                <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
                  <Group gap={6} wrap="nowrap" justify="space-between">
                    <Text fw={600} truncate>
                      {user.name}
                    </Text>
                    {isLoading ? (
                      <Loader size="xs" />
                    ) : (
                      <Text size="xs" c="dimmed">
                        {user.teamName}
                      </Text>
                    )}
                  </Group>
                  <div>
                    <RoleBadge role={user.role} />
                  </div>
                </Stack>
              </Group>
            </UnstyledButton>
          )
        })}
      </SimpleGrid>
      {mutation.isError && (
        <Alert color="red" variant="light" role="alert">
          {loginErrorText(mutation.error)}
        </Alert>
      )}
    </Stack>
  )
}
