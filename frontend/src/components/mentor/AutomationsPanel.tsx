import { Group, Paper, Stack, Switch, Text } from '@mantine/core'
import { useAutomations, useSetAutomation } from '../../hooks/useMentor'
import { toast } from '../../store/toastStore'
import { ErrorBlock } from '../QueryState'

// Что AI-buddy делает сам по расписанию (имитация ИИ-крона и feature flags Авито).
export function AutomationsPanel() {
  const list = useAutomations()
  const set = useSetAutomation()
  return (
    <Paper radius="lg" p="lg">
      <Group justify="space-between" align="baseline" mb="xs" wrap="wrap">
        <Text component="h2" fz={18} fw={600} m={0}>
          Автоматизации AI-buddy
        </Text>
        <Text size="xs" c="ink.5">
          работают по расписанию, без твоего участия
        </Text>
      </Group>
      {list.isError && <ErrorBlock error={list.error} onRetry={() => list.refetch()} />}
      <Stack gap={0}>
        {list.data?.map((a) => (
          <Group
            key={a.id}
            justify="space-between"
            wrap="nowrap"
            py={10}
            style={{ borderTop: '1px solid #D6D6D6' }}
          >
            <div>
              <Text fw={500}>{a.title}</Text>
              <Text size="xs" c="ink.5">
                {a.note}
              </Text>
            </div>
            <Switch
              color="green.6"
              aria-label={a.title}
              checked={a.enabled}
              disabled={set.isPending && set.variables?.id === a.id}
              onChange={(e) => {
                const enabled = e.currentTarget.checked
                set.mutate(
                  { id: a.id, enabled },
                  {
                    onSuccess: () =>
                      toast(`${enabled ? 'Включено' : 'Выключено'}: ${a.title.toLowerCase()}`),
                  },
                )
              }}
            />
          </Group>
        ))}
      </Stack>
    </Paper>
  )
}
