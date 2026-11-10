import {
  Accordion,
  Alert,
  Button,
  Checkbox,
  Paper,
  Select,
  SimpleGrid,
  Stack,
  Text,
  Textarea,
  Title,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { zod4Resolver } from 'mantine-form-zod-resolver'
import { useState } from 'react'
import { z } from 'zod'
import { Mascot } from '../components/Mascot'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { CONTACTS, FAQ, PROBLEM_TYPES } from '../content/support'
import { useAuthStore } from '../store/authStore'

const ProblemSchema = z.object({
  type: z.string().min(1, 'Выбери, что случилось'),
  details: z.string().trim().min(10, 'Опиши подробнее, хотя бы пару слов').max(1000),
  attachDialog: z.boolean(),
})
type Problem = z.infer<typeof ProblemSchema>

// Поддержка: помощник разгружает наставника, но не заменяет его – здесь всегда можно позвать людей.
export function SupportPage() {
  const role = useAuthStore((s) => s.user?.role)
  const [sent, setSent] = useState(false)
  const form = useForm<Problem>({
    mode: 'uncontrolled',
    initialValues: { type: PROBLEM_TYPES[0], details: '', attachDialog: true },
    validate: zod4Resolver(ProblemSchema),
  })
  // Стажеру показываем наставника и руководителя, наставнику – общие службы.
  const contacts =
    role === 'intern' ? CONTACTS : CONTACTS.filter((c) => c.id === 'hr' || c.id === 'it')

  return (
    <Page maxWidth={1240}>
      <Stack gap={28} style={{ flex: 1, minWidth: 0 }}>
        <Paper
          radius="lg"
          p="xl"
          style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 28 }}
        >
          <Mascot mood="mentor" size={120} glow />
          <div style={{ flex: '1 1 320px' }}>
            <PageHeader
              overline="Поддержка"
              title={
                <>
                  Не получается? <Marker color="#965EEB">Позову людей</Marker>
                </>
              }
            />
            <Text c="ink.5" mt="sm" maw={560}>
              AI-buddy разгружает наставника, но не заменяет его. Если вопрос сложный или ответ
              показался неверным, напиши человеку напрямую.
            </Text>
          </div>
        </Paper>

        <Stack gap="sm">
          <Title order={2}>Кому написать</Title>
          <SimpleGrid cols={{ base: 1, xs: 2, md: 4 }} spacing="md">
            {contacts.map((c) => (
              <div
                key={c.id}
                style={{
                  background: c.bg,
                  padding: 18,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 10,
                  minHeight: 176,
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span
                    style={{
                      width: 40,
                      height: 40,
                      borderRadius: 20,
                      background: '#FFFFFF',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontWeight: 600,
                      flex: 'none',
                    }}
                  >
                    {c.initials}
                  </span>
                  <div>
                    <Text fw={500}>{c.name}</Text>
                    <Text size="xs">{c.role}</Text>
                  </div>
                </div>
                <Text size="sm" style={{ flex: 1 }}>
                  {c.when}
                </Text>
                <Button
                  variant="default"
                  size="sm"
                  style={{ alignSelf: 'flex-start', borderColor: '#141414' }}
                >
                  Написать
                </Button>
              </div>
            ))}
          </SimpleGrid>
        </Stack>

        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24, alignItems: 'flex-start' }}>
          <Paper radius="lg" p="lg" style={{ flex: '999 1 420px', minWidth: 0 }}>
            <Title order={2} mb="sm">
              Частые вопросы
            </Title>
            <Accordion defaultValue={FAQ[0].q} chevronPosition="right" variant="default">
              {FAQ.map((f) => (
                <Accordion.Item key={f.q} value={f.q}>
                  <Accordion.Control>{f.q}</Accordion.Control>
                  <Accordion.Panel>{f.a}</Accordion.Panel>
                </Accordion.Item>
              ))}
            </Accordion>
          </Paper>

          <Paper radius="lg" p="lg" style={{ flex: '1 1 320px', minWidth: 0 }}>
            <Title order={2}>Сообщить о проблеме</Title>
            <Text size="sm" c="ink.5" mt={6} mb="md">
              Например, если AI-buddy ответил неверно или не сработала заявка.
            </Text>
            {sent ? (
              <Alert
                color="green"
                variant="light"
                role="status"
                icon={<Mascot size={24} mood="happy" />}
              >
                Спасибо! Сообщение отправлено, ответ придет сюда и в чат команды.
                <div>
                  <Button
                    mt="sm"
                    size="xs"
                    variant="outline"
                    color="ink"
                    onClick={() => {
                      form.reset()
                      setSent(false)
                    }}
                  >
                    Написать еще
                  </Button>
                </div>
              </Alert>
            ) : (
              <form noValidate onSubmit={form.onSubmit(() => setSent(true))}>
                <Stack gap="sm">
                  <Select
                    label="Что случилось"
                    data={PROBLEM_TYPES}
                    allowDeselect={false}
                    key={form.key('type')}
                    {...form.getInputProps('type')}
                  />
                  <Textarea
                    label="Подробности"
                    placeholder="Опиши, что произошло"
                    autosize
                    minRows={4}
                    key={form.key('details')}
                    {...form.getInputProps('details')}
                  />
                  <Checkbox
                    label="Приложить последний диалог с AI-buddy"
                    key={form.key('attachDialog')}
                    {...form.getInputProps('attachDialog', { type: 'checkbox' })}
                  />
                  <Button type="submit" color="ink">
                    Отправить
                  </Button>
                </Stack>
              </form>
            )}
          </Paper>
        </div>
        <Text size="xs" c="ink.5">
          Демо-версия: люди и данные синтетические.
        </Text>
      </Stack>
    </Page>
  )
}
