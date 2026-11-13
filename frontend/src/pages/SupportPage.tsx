import {
  Accordion,
  Alert,
  Button,
  Checkbox,
  Collapse,
  Paper,
  Select,
  SimpleGrid,
  Stack,
  Text,
  Textarea,
  UnstyledButton,
  Title,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { zod4Resolver } from 'mantine-form-zod-resolver'
import { useState } from 'react'
import { IconChevronDown } from '@tabler/icons-react'
import { z } from 'zod'
import { Mascot } from '../components/Mascot'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import {
  BUDDY_TEAM_CARD,
  CURATOR_NOTE,
  DEPARTMENT_STYLE,
  FAQ,
  FAQ_LEAD,
  FAQ_STAFF,
  PERSON_CARDS,
  PROBLEM_TYPES,
  PROBLEM_TYPES_LEAD,
  PROBLEM_TYPES_STAFF,
} from '../content/support'
import { useInterns } from '../hooks/usePath'
import { getContacts } from '../api/people'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useAuthStore } from '../store/authStore'
import { useCreateTicket, useTickets } from '../hooks/useSupport'
import { errorText } from '../lib/errors'
import { StatusTag } from '../components/StatusTag'
import type { TSupportTicket } from '../schemas/support'

const ProblemSchema = z.object({
  type: z.string().min(1, 'Выбери, что случилось'),
  intern: z.string(),
  details: z.string().trim().min(10, 'Опиши подробнее, хотя бы пару слов').max(2000),
  attachDialog: z.boolean(),
})
type Problem = z.infer<typeof ProblemSchema>

// Поддержка: помощник разгружает наставника, но не заменяет его – здесь всегда можно позвать людей.
export function SupportPage() {
  const role = useAuthStore((s) => s.user?.role)
  // Наставнику – свои вопросы: про стажеров, доступы и работу AI-buddy.
  const staff = role !== 'intern'
  // Руководителю – только технические сбои: согласования, цели, уведомления.
  const lead = role === 'lead'
  const faq = lead ? FAQ_LEAD : staff ? FAQ_STAFF : FAQ
  const types = lead ? PROBLEM_TYPES_LEAD : staff ? PROBLEM_TYPES_STAFF : PROBLEM_TYPES
  const interns = useInterns({ enabled: staff })
  const create = useCreateTicket()
  const sent = create.data ?? null
  const form = useForm<Problem>({
    mode: 'uncontrolled',
    initialValues: { type: types[0], intern: '', details: '', attachDialog: true },
    validate: zod4Resolver(ProblemSchema),
  })
  // Имена – из связей пользователя в backend. Стажеру показываем наставника и руководителя,
  // всем – отделы компании.
  const query = useQuery({ queryKey: ['contacts'], queryFn: getContacts })
  const data = query.data
  const people =
    role === 'intern' && data
      ? [
          {
            id: 'mentor',
            name: data.mentor?.name ?? 'Наставник',
            ...PERSON_CARDS.mentor,
            role: data.mentor ? PERSON_CARDS.mentor.role : 'пока не назначен',
          },
          ...(data.lead ? [{ id: 'lead', name: data.lead.name, ...PERSON_CARDS.lead }] : []),
        ].map((c) => ({ ...c, initials: c.name.slice(0, 1) }))
      : staff && !lead && data?.lead
        ? [
            {
              id: 'lead',
              name: data.lead.name,
              initials: data.lead.name.slice(0, 1),
              ...PERSON_CARDS.lead,
              when: 'Цели стажеров, согласования, оценка по итогам стажировки.',
            },
          ]
        : []
  const departments = (data?.departments ?? []).map((d) => ({
    id: d.code,
    name: d.title,
    role: `ответ до ${d.slaHours} ч`,
    // Куратор стажировки (HR BP) работает в HR: отдельной карточкой его не показываем.
    when: d.code === 'hr' ? `${d.about} ${CURATOR_NOTE}` : d.about,
    initials: DEPARTMENT_STYLE[d.code]?.initials ?? d.title.slice(0, 2),
    bg: DEPARTMENT_STYLE[d.code]?.bg ?? '#FFFFFF',
  }))
  const contacts = [...people, ...(staff ? [BUDDY_TEAM_CARD] : []), ...departments]
  const navigate = useNavigate()
  // Что делает кнопка карточки: наставник – переписка, отдел – заявка через AI-buddy,
  // остальное – форма «Сообщить о проблеме» ниже. Руководителю стажер пишет через наставника.
  const contactAction = (id: string, name: string) => {
    if (role === 'intern' && id === 'mentor' && data?.mentor) {
      return { label: 'Написать', run: () => navigate({ to: '/mentor' }) }
    }
    if (role === 'intern' && DEPARTMENT_STYLE[id]) {
      return {
        label: 'Оформить заявку',
        run: () => navigate({ to: '/chat', search: { q: `Нужна заявка: ${name.toLowerCase()}` } }),
      }
    }
    if (id === 'lead' || id === 'mentor') return null
    return {
      label: 'Сообщить о проблеме',
      run: () => {
        document.getElementById('support-form')?.scrollIntoView?.({ behavior: 'smooth' })
        document.getElementById('support-details')?.focus()
      },
    }
  }

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
              overline={lead || !staff ? 'Поддержка' : 'Поддержка наставника'}
              title={
                lead ? (
                  <>
                    Что-то <Marker color="#965EEB">не работает?</Marker>
                  </>
                ) : staff ? (
                  <>
                    Нужна помощь <Marker color="#965EEB">со стажерами?</Marker>
                  </>
                ) : (
                  <>
                    Не получается? <Marker color="#965EEB">Позову людей</Marker>
                  </>
                )
              }
            />
            <Text c="ink.5" mt="sm" maw={600}>
              {lead
                ? 'Здесь можно сообщить о техническом сбое: не согласуется заявка, не сохраняются цели, не приходят уведомления.'
                : staff
                  ? 'Здесь вопросы про наставничество, доступы стажеров и работу AI-buddy. Вопросы самих стажеров приходят в «Сообщения».'
                  : 'AI-buddy разгружает наставника, но не заменяет его. Если вопрос сложный или ответ показался неверным, напиши человеку напрямую.'}
            </Text>
          </div>
        </Paper>

        <Stack gap="sm">
          <Title order={2}>Кому написать</Title>
          <SimpleGrid cols={{ base: 1, xs: 2, md: 3 }} spacing="md">
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
                {(() => {
                  const act = contactAction(c.id, c.name)
                  return act ? (
                    <Button
                      variant="default"
                      size="sm"
                      onClick={act.run}
                      style={{ alignSelf: 'flex-start', borderColor: '#141414' }}
                    >
                      {act.label}
                    </Button>
                  ) : (
                    <Text size="xs" c="ink.5">
                      {c.id === 'lead' && role === 'intern'
                        ? 'Вопросы к руководителю обсуди с наставником или на 1:1'
                        : 'Пока не назначен'}
                    </Text>
                  )
                })()}
              </div>
            ))}
          </SimpleGrid>
        </Stack>

        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24, alignItems: 'flex-start' }}>
          <Paper radius="lg" p="lg" style={{ flex: '999 1 420px', minWidth: 0 }}>
            <Title order={2} mb="sm">
              {staff ? 'Частые вопросы наставников' : 'Частые вопросы'}
            </Title>
            <Accordion defaultValue={faq[0].q} chevronPosition="right" variant="default">
              {faq.map((f) => (
                <Accordion.Item key={f.q} value={f.q}>
                  <Accordion.Control>{f.q}</Accordion.Control>
                  <Accordion.Panel>{f.a}</Accordion.Panel>
                </Accordion.Item>
              ))}
            </Accordion>
          </Paper>

          <Paper id="support-form" radius="lg" p="lg" style={{ flex: '1 1 320px', minWidth: 0 }}>
            <Title order={2}>Сообщить о проблеме</Title>
            <Text size="sm" c="ink.5" mt={6} mb="md">
              {lead
                ? 'Например, не согласуется заявка или не сохраняются цели стажера.'
                : staff
                  ? 'Например, AI-buddy дал стажеру неверный ответ или не сработала автоматизация.'
                  : 'Например, если AI-buddy ответил неверно или не сработала заявка.'}
            </Text>
            {sent ? (
              <Alert
                color="green"
                variant="light"
                role="status"
                icon={<Mascot size={24} mood="happy" />}
              >
                Обращение {sent.number} принято. Ответ придет в уведомления и в список ниже.
                <div>
                  <Button
                    mt="sm"
                    size="xs"
                    variant="outline"
                    color="ink"
                    onClick={() => {
                      form.reset()
                      create.reset()
                    }}
                  >
                    Написать еще
                  </Button>
                </div>
              </Alert>
            ) : (
              <form
                noValidate
                onSubmit={form.onSubmit((v) =>
                  create.mutate({
                    type: v.type,
                    details: v.details,
                    internId: v.intern || null,
                    attachDialog: lead ? false : v.attachDialog,
                  }),
                )}
              >
                <Stack gap="sm">
                  <Select
                    label="Что случилось"
                    data={types}
                    allowDeselect={false}
                    key={form.key('type')}
                    {...form.getInputProps('type')}
                  />
                  {staff && (
                    <Select
                      label="Какой стажер"
                      data={[
                        ...(interns.data ?? []).map((i) => ({ value: i.id, label: i.name })),
                        { value: '', label: 'не про конкретного стажера' },
                      ]}
                      allowDeselect={false}
                      key={form.key('intern')}
                      {...form.getInputProps('intern')}
                    />
                  )}
                  <Textarea
                    id="support-details"
                    label="Подробности"
                    placeholder="Опиши, что произошло"
                    autosize
                    minRows={4}
                    key={form.key('details')}
                    {...form.getInputProps('details')}
                  />
                  {!lead && (
                    <Checkbox
                      label={
                        staff
                          ? 'Приложить диалог стажера с AI-buddy'
                          : 'Приложить последний диалог с AI-buddy'
                      }
                      key={form.key('attachDialog')}
                      {...form.getInputProps('attachDialog', { type: 'checkbox' })}
                    />
                  )}
                  {create.isError && (
                    <Alert color="red" variant="light" role="alert">
                      {errorText(create.error)}
                    </Alert>
                  )}
                  <Button type="submit" color="ink" loading={create.isPending}>
                    Отправить
                  </Button>
                </Stack>
              </form>
            )}
            <MyTickets />
          </Paper>
        </div>
        <Text size="xs" c="ink.5">
          Демо-версия: люди и данные синтетические.
        </Text>
      </Stack>
    </Page>
  )
}

const TICKET_STATUS: Record<TSupportTicket['status'], { label: string; bg: string }> = {
  new: { label: 'принято', bg: '#F4F4F4' },
  in_progress: { label: 'в работе', bg: '#BADAFB' },
  done: { label: 'решено', bg: '#C9FFBF' },
}

// Мои обращения: свернуты в одну строку, чтобы не перегружать страницу.
function MyTickets() {
  const tickets = useTickets()
  const [open, setOpen] = useState(false)
  const list = tickets.data ?? []
  if (list.length === 0) return null
  const active = list.filter((t) => t.status !== 'done').length
  return (
    <div style={{ marginTop: 20, borderTop: '1px solid #D6D6D6', paddingTop: 14 }}>
      <UnstyledButton
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 14 }}
      >
        Мои обращения · {list.length}
        {active > 0 && <span style={{ color: '#5C5C5C' }}>, в работе {active}</span>}
        <IconChevronDown
          size={16}
          aria-hidden="true"
          style={{ transform: open ? 'rotate(180deg)' : 'none', transition: 'transform .2s' }}
        />
      </UnstyledButton>
      <Collapse expanded={open}>
        <ul
          aria-label="Мои обращения"
          style={{ listStyle: 'none', margin: '12px 0 0', padding: 0 }}
        >
          {list.map((t) => (
            <li key={t.id} style={{ padding: '10px 0', borderTop: '1px solid #EDEDED' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <Text size="sm" fw={500} style={{ flex: 1, minWidth: 0 }}>
                  {t.number} · {t.type}
                </Text>
                <StatusTag bg={TICKET_STATUS[t.status].bg}>
                  {TICKET_STATUS[t.status].label}
                </StatusTag>
              </div>
              {t.answer && (
                <Text size="sm" c="ink.5" mt={4}>
                  Ответ: {t.answer}
                </Text>
              )}
            </li>
          ))}
        </ul>
      </Collapse>
    </div>
  )
}
