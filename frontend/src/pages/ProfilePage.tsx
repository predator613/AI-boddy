import {
  Button,
  Group,
  Paper,
  SegmentedControl,
  SimpleGrid,
  Stack,
  Switch,
  Text,
  Title,
} from '@mantine/core'
import { useQueryClient } from '@tanstack/react-query'
import { PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { RoleBadge } from '../components/RoleBadge'
import { AutomationsPanel } from '../components/mentor/AutomationsPanel'
import { useMyPath } from '../hooks/usePath'
import { useAuthStore } from '../store/authStore'
import { useSettingsStore, type AnswerStyle, type NotificationKey } from '../store/settingsStore'

const NOTIFICATIONS: { key: NotificationKey; title: string; note: string }[] = [
  { key: 'tasks', title: 'Задачи и сроки', note: 'напоминать о задачах этапа' },
  { key: 'requests', title: 'Статус заявок', note: 'когда заявку одобрят или отклонят' },
  { key: 'meetings', title: 'Встречи с наставником', note: 'за час до 1:1' },
  { key: 'tips', title: 'Подсказки AI-buddy', note: 'советы на главной раз в день' },
]

export function ProfilePage() {
  const user = useAuthStore((s) => s.user)
  const path = useMyPath({ enabled: user?.role === 'intern' })
  const logout = useAuthStore((s) => s.logout)
  const queryClient = useQueryClient()
  const { notifications, answerStyle, toggle, setAnswerStyle } = useSettingsStore()
  if (!user) return null

  return (
    <Page maxWidth={1240}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <PageHeader overline="Профиль и настройки" title="Профиль" />

        <Paper radius="lg" p="lg">
          <Group gap="lg" wrap="wrap">
            <span
              style={{
                width: 72,
                height: 72,
                borderRadius: 36,
                background: '#BADAFB',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 28,
                fontWeight: 600,
              }}
            >
              {user.name.charAt(0)}
            </span>
            <div style={{ flex: '1 1 200px' }}>
              <Text fz={24} fw={500}>
                {user.name}
              </Text>
              <Text size="sm" c="ink.5">
                {user.login}
              </Text>
            </div>
            <dl
              style={{
                display: 'grid',
                gridTemplateColumns: 'auto auto',
                gap: '6px 16px',
                margin: 0,
                fontSize: 14,
              }}
            >
              <dt style={{ color: '#5C5C5C' }}>Роль</dt>
              <dd style={{ margin: 0 }}>
                <RoleBadge role={user.role} />
              </dd>
              <dt style={{ color: '#5C5C5C' }}>Команда</dt>
              <dd style={{ margin: 0 }}>{user.teamName}</dd>
              {user.role === 'intern' && (
                <>
                  <dt style={{ color: '#5C5C5C' }}>Наставник</dt>
                  <dd style={{ margin: 0 }}>
                    {path.data ? (path.data.intern.mentor?.name ?? 'пока не назначен') : '…'}
                  </dd>
                  <dt style={{ color: '#5C5C5C' }}>Стажировка</dt>
                  <dd style={{ margin: 0 }}>
                    {path.data
                      ? `неделя ${path.data.week.current} из ${path.data.week.total}`
                      : '…'}
                  </dd>
                </>
              )}
            </dl>
          </Group>
          <Text size="xs" c="ink.5" mt="md" pt="sm" style={{ borderTop: '1px solid #D6D6D6' }}>
            Данные приходят из системы учетных записей компании, изменить их здесь нельзя.
          </Text>
        </Paper>

        <SimpleGrid cols={{ base: 1, md: 2 }} spacing="lg">
          <Paper radius="lg" p="lg" id="notifications">
            <Title order={2} mb="xs">
              Уведомления
            </Title>
            {NOTIFICATIONS.map((n) => (
              <Group
                key={n.key}
                justify="space-between"
                wrap="nowrap"
                py={14}
                style={{ borderTop: '1px solid #D6D6D6' }}
              >
                <div>
                  <Text size="15px">{n.title}</Text>
                  <Text size="xs" c="ink.5">
                    {n.note}
                  </Text>
                </div>
                <Switch
                  size="md"
                  aria-label={n.title}
                  checked={notifications[n.key]}
                  onChange={() => toggle(n.key)}
                />
              </Group>
            ))}
          </Paper>

          {/* Чат с AI-buddy есть только у стажера: наставнику и руководителю настройка ни на что не влияет. */}
          {user?.role === 'intern' && (
            <Paper radius="lg" p="lg">
              <Title order={2} mb="md">
                Помощник
              </Title>
              <Text size="sm" mb={8}>
                Как отвечать
              </Text>
              <SegmentedControl
                fullWidth
                aria-label="Подробность ответов"
                value={answerStyle}
                onChange={(v) => setAnswerStyle(v as AnswerStyle)}
                data={[
                  { value: 'short', label: 'Коротко' },
                  { value: 'normal', label: 'Обычно' },
                  { value: 'detailed', label: 'Подробно' },
                ]}
              />
              <Text size="xs" c="ink.5" mt={6}>
                {answerStyle === 'short'
                  ? 'AI-buddy отвечает в 1–2 предложениях, без списков.'
                  : answerStyle === 'detailed'
                    ? 'AI-buddy объясняет причины и расписывает шаги.'
                    : 'AI-buddy отвечает в 2–6 предложениях, шаги списком, если нужно.'}
              </Text>
              <Group
                justify="space-between"
                mt="lg"
                pt="md"
                style={{ borderTop: '1px solid #D6D6D6' }}
              >
                <div>
                  <Text size="15px">Язык</Text>
                  <Text size="xs" c="ink.5">
                    интерфейса и ответов
                  </Text>
                </div>
                <Text size="sm">Русский</Text>
              </Group>
            </Paper>
          )}
        </SimpleGrid>

        {user?.role === 'mentor' && <AutomationsPanel />}

        <Button
          variant="outline"
          color="ink"
          style={{ alignSelf: 'flex-start' }}
          onClick={() => {
            logout()
            queryClient.clear()
          }}
        >
          Выйти из аккаунта
        </Button>
      </Stack>
    </Page>
  )
}
