import {
  ActionIcon,
  Button,
  Modal,
  NativeSelect,
  NumberInput,
  Paper,
  Text,
  TextInput,
  UnstyledButton,
} from '@mantine/core'
import { IconPencil, IconPlus, IconTrash, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { useAgreeGoals, useGoalEditor, useGoalStep, useInternGoals } from '../hooks/useGoals'
import { useInterns } from '../hooks/usePath'
import { errorText } from '../lib/errors'
import {
  GoalInput,
  COMPETENCIES,
  agreedLabel,
  type TGoal,
  type TGoalStep,
  type TGoalsView,
} from '../schemas/goals'
import type { TInternSummary } from '../schemas/path'
import { useAuthStore } from '../store/authStore'
import { toast } from '../store/toastStore'

// Главная руководителя: только цели стажеров. Черновик целей можно править и согласовать,
// после согласования руководитель подтверждает свои шаги – из них считается прогресс.

const SOURCE: Record<TGoalStep['source'], string> = {
  task: 'задача плана',
  mentor: 'наставник',
  lead: 'руководитель',
}

export function LeadGoalsPage() {
  const user = useAuthStore((s) => s.user)
  const interns = useInterns()
  const [sel, setSel] = useState<string | null>(null)
  const selId = sel ?? interns.data?.[0]?.id ?? null
  return (
    <Page maxWidth={1240}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24, flex: 1, minWidth: 0 }}>
        <PageHeader
          overline={`Руководитель · ${user?.teamName ?? ''}`}
          title={
            <>
              Цели <Marker color="#00AAFF">стажеров</Marker>
            </>
          }
        />
        {interns.isPending && <LoadingBlocks label="Загрузка стажеров" />}
        {interns.isError && <ErrorBlock error={interns.error} onRetry={() => interns.refetch()} />}
        {interns.data?.length === 0 && <Text>В команде пока нет стажеров.</Text>}
        {interns.data && interns.data.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24, alignItems: 'flex-start' }}>
            <nav
              aria-label="Стажеры команды"
              style={{
                flex: '0 1 300px',
                minWidth: 260,
                display: 'flex',
                flexDirection: 'column',
                gap: 12,
              }}
            >
              {interns.data.map((i) => (
                <InternCard
                  key={i.id}
                  intern={i}
                  selected={i.id === selId}
                  onPick={() => setSel(i.id)}
                />
              ))}
              <Text fz={12} c="ink.5" px={4} style={{ lineHeight: 1.45 }}>
                Цели обсуждаются со стажером на первой неделе. После согласования стажер их не
                меняет, а прогресс считается по шагам.
              </Text>
            </nav>
            {selId && (
              <GoalsPanel
                internId={selId}
                name={interns.data.find((i) => i.id === selId)?.name ?? ''}
              />
            )}
          </div>
        )}
      </div>
    </Page>
  )
}

/** Шаги руководителя, которые еще не подтверждены. */
const leadSteps = (v: TGoalsView | undefined) =>
  v?.agreed
    ? v.goals.flatMap((g) => g.steps.filter((s) => s.source === 'lead' && s.status !== 'done'))
    : []

function InternCard({
  intern: i,
  selected,
  onPick,
}: {
  intern: TInternSummary
  selected: boolean
  onPick: () => void
}) {
  const goals = useInternGoals(i.id)
  const wait = leadSteps(goals.data).length
  const state = !goals.data
    ? { text: '…', bg: '#F4F4F4' }
    : !goals.data.agreed
      ? { text: goals.data.goals.length ? 'ждут согласования' : 'целей еще нет', bg: '#BADAFB' }
      : wait
        ? { text: `твоих шагов: ${wait}`, bg: '#DFCDF3' }
        : { text: 'согласованы', bg: '#C9FFBF' }
  return (
    <UnstyledButton
      onClick={onPick}
      aria-pressed={selected}
      style={{
        background: '#FFFFFF',
        borderRadius: 16,
        padding: 16,
        outline: selected ? '2px solid #141414' : 'none',
        outlineOffset: -2,
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
      }}
    >
      <span style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <span
          style={{
            width: 40,
            height: 40,
            borderRadius: 20,
            background: '#BADAFB',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontWeight: 600,
          }}
        >
          {i.name.slice(0, 1)}
        </span>
        <span style={{ display: 'flex', flexDirection: 'column' }}>
          <Text fz={17} fw={500}>
            {i.name}
          </Text>
          <Text fz={12} c="ink.5">
            неделя {i.week}
            {i.mentor ? ` · наставник ${i.mentor.name}` : ''}
          </Text>
        </span>
      </span>
      <StatusTag bg={state.bg} style={{ alignSelf: 'flex-start' }}>
        {state.text}
      </StatusTag>
    </UnstyledButton>
  )
}

function GoalsPanel({ internId, name }: { internId: string; name: string }) {
  const query = useInternGoals(internId)
  const agree = useAgreeGoals(internId)
  const editor = useGoalEditor(internId)
  const step = useGoalStep(internId)
  const [modal, setModal] = useState<TGoal | 'new' | null>(null)

  if (query.isPending)
    return (
      <div style={{ flex: '999 1 520px' }}>
        <LoadingBlocks label="Загрузка целей" />
      </div>
    )
  if (query.isError)
    return (
      <div style={{ flex: '999 1 520px' }}>
        <ErrorBlock error={query.error} onRetry={() => query.refetch()} />
      </div>
    )
  const v = query.data
  const draft = !v.agreed
  const editable = draft && v.canEdit
  return (
    <Paper
      radius="lg"
      p="lg"
      className="ab-pop"
      style={{
        flex: '999 1 520px',
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
      }}
      aria-label={`Цели: ${name}`}
      component="section"
    >
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: 12,
        }}
      >
        <div>
          <Text component="h2" fz={22} fw={500} m={0}>
            {name}
          </Text>
          <Text fz={13} c="ink.5">
            {draft
              ? 'Черновик со встречи. Проверь цели и шаги, затем согласуй.'
              : 'Шаги с пометкой «руководитель» подтверждаешь ты.'}
          </Text>
        </div>
        <StatusTag bg={draft ? '#BADAFB' : '#C9FFBF'}>{agreedLabel(v)}</StatusTag>
      </div>

      {v.goals.map((g) => (
        <div
          key={g.id}
          style={{
            background: '#F4F4F4',
            borderRadius: 12,
            padding: '14px 16px',
            display: 'flex',
            flexDirection: 'column',
            gap: 10,
          }}
        >
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              gap: 12,
              alignItems: 'flex-start',
            }}
          >
            <span style={{ display: 'flex', flexDirection: 'column', gap: 3, minWidth: 0 }}>
              <Text fw={500} fz={15}>
                {g.title}
              </Text>
              <Text fz={12} c="ink.5">
                критерий: {g.criterion}
                {g.dueWeek ? ` (неделя ${g.dueWeek})` : ''}
                {g.competency ? ` · ${g.competency}` : ''}
              </Text>
            </span>
            {editable ? (
              <span style={{ display: 'flex', gap: 4, flex: 'none' }}>
                <ActionIcon
                  variant="subtle"
                  color="ink"
                  aria-label={`Изменить цель «${g.title}»`}
                  onClick={() => setModal(g)}
                >
                  <IconPencil size={16} />
                </ActionIcon>
                <ActionIcon
                  variant="subtle"
                  color="ink"
                  aria-label={`Удалить цель «${g.title}»`}
                  loading={editor.remove.isPending && editor.remove.variables === g.id}
                  onClick={() => editor.remove.mutate(g.id)}
                >
                  <IconTrash size={16} />
                </ActionIcon>
              </span>
            ) : (
              <Text fw={600} fz={16} style={{ flex: 'none' }}>
                {g.progressPct}%
              </Text>
            )}
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
            {g.steps.map((s) => {
              const canConfirm = !draft && s.source === 'lead' && s.status !== 'done'
              return (
                <span
                  key={s.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    fontSize: 12,
                    background: '#FFFFFF',
                    borderRadius: 14,
                    padding: '4px 6px 4px 10px',
                    outline: canConfirm ? '2px solid #965EEB' : 'none',
                  }}
                >
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: 4,
                      background:
                        s.status === 'done'
                          ? '#04E061'
                          : s.status === 'in_progress'
                            ? '#00AAFF'
                            : '#D6D6D6',
                    }}
                  />
                  {s.title}
                  <span style={{ color: '#5C5C5C' }}>· {SOURCE[s.source]}</span>
                  {canConfirm && (
                    <Button
                      size="compact-xs"
                      color="ink"
                      radius="xl"
                      loading={step.isPending && step.variables?.stepId === s.id}
                      onClick={() =>
                        step.mutate(
                          { goalId: g.id, stepId: s.id, done: true },
                          {
                            onSuccess: () =>
                              toast(`Шаг «${s.title}» подтвержден, прогресс пересчитан`),
                          },
                        )
                      }
                    >
                      Подтвердить
                    </Button>
                  )}
                </span>
              )
            })}
          </div>
        </div>
      ))}
      {v.goals.length === 0 && (
        <Text fz={14}>Целей пока нет. Добавь их после встречи со стажером.</Text>
      )}
      {step.isError && (
        <Text c="red.7" fz={13}>
          {errorText(step.error)}
        </Text>
      )}

      {editable && (
        <>
          <Button
            variant="default"
            leftSection={<IconPlus size={16} />}
            disabled={v.goals.length >= 6}
            onClick={() => setModal('new')}
          >
            Цель {v.goals.length >= 6 ? '(не больше 6)' : ''}
          </Button>
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'center',
              gap: 12,
              background: '#DFF0FF',
              borderRadius: 12,
              padding: '14px 16px',
            }}
          >
            <Text fz={13} style={{ flex: '1 1 280px', lineHeight: 1.45 }}>
              После согласования цели появятся у стажера в «Моем пути», наставник получит
              уведомление, а прогресс начнет считаться по шагам.
            </Text>
            {v.canAgree && (
              <Button
                color="ink"
                loading={agree.isPending}
                onClick={() =>
                  agree.mutate(undefined, {
                    onSuccess: () => toast(`Цели согласованы. ${name} увидит их в «Моем пути»`),
                  })
                }
              >
                Согласовать цели
              </Button>
            )}
          </div>
          {agree.isError && (
            <Text c="red.7" fz={13}>
              {errorText(agree.error)}
            </Text>
          )}
        </>
      )}

      {modal && (
        <GoalModal
          goal={modal === 'new' ? null : modal}
          okrs={v.okrs}
          busy={editor.create.isPending || editor.update.isPending}
          onClose={() => setModal(null)}
          onSave={(input) => {
            const done = { onSuccess: () => setModal(null) }
            if (modal === 'new') editor.create.mutate(input, done)
            else editor.update.mutate({ goalId: modal.id, ...input }, done)
          }}
          error={editor.create.error ?? editor.update.error}
        />
      )}
    </Paper>
  )
}

interface StepRow {
  title: string
  source: TGoalStep['source']
}

interface GoalModalProps {
  goal: TGoal | null
  okrs: TGoalsView['okrs']
  busy: boolean
  error: unknown
  onClose: () => void
  onSave: (input: {
    title: string
    criterion: string
    dueWeek: number
    okrId: string
    competency: string
    steps: StepRow[]
  }) => void
}

export function GoalModal({ goal, okrs, busy, error, onClose, onSave }: GoalModalProps) {
  const [title, setTitle] = useState(goal?.title ?? '')
  const [criterion, setCriterion] = useState(goal?.criterion ?? '')
  const [dueWeek, setDueWeek] = useState<number>(goal?.dueWeek ?? 8)
  const [okrId, setOkrId] = useState(goal?.okr?.id ?? '')
  const [competency, setCompetency] = useState(goal?.competency ?? '')
  const [steps, setSteps] = useState<StepRow[]>(
    goal?.steps.map((s) => ({ title: s.title, source: s.source })) ?? [
      { title: '', source: 'task' },
      { title: '', source: 'mentor' },
    ],
  )
  const [formError, setFormError] = useState<string | null>(null)

  const patchStep = (k: number, p: Partial<StepRow>) =>
    setSteps((list) => list.map((s, j) => (j === k ? { ...s, ...p } : s)))

  const submit = () => {
    const filled = steps.filter((s) => s.title.trim())
    const parsed = GoalInput.safeParse({
      title,
      criterion,
      dueWeek,
      okrId,
      competency,
      steps: filled,
    })
    if (!parsed.success) return setFormError(parsed.error.issues[0]?.message ?? 'Проверь поля')
    if (!filled.length) return setFormError('Добавь хотя бы один шаг: из шагов считается прогресс')
    setFormError(null)
    onSave({ title, criterion, dueWeek, okrId, competency, steps: filled })
  }

  return (
    <Modal
      opened
      onClose={onClose}
      title={goal ? 'Изменить цель' : 'Новая цель'}
      radius="lg"
      size="lg"
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <TextInput
          label="Цель"
          value={title}
          onChange={(e) => setTitle(e.currentTarget.value)}
          placeholder="Например, «Пройти путь задачи до релиза»"
        />
        <TextInput
          label="Критерий успеха"
          value={criterion}
          onChange={(e) => setCriterion(e.currentTarget.value)}
          placeholder="Как поймем, что цель достигнута"
        />
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 120px', gap: 12 }}>
          <NativeSelect
            label="OKR команды"
            value={okrId}
            onChange={(e) => setOkrId(e.currentTarget.value)}
            data={[
              { value: '', label: 'без OKR' },
              ...okrs.map((o) => ({ value: o.id, label: `${o.period} · ${o.title}` })),
            ]}
          />
          <NumberInput
            label="К неделе"
            min={1}
            max={52}
            value={dueWeek}
            onChange={(v) => setDueWeek(Number(v) || 0)}
          />
        </div>
        <NativeSelect
          label="Компетенция из матрицы роли"
          value={competency}
          onChange={(e) => setCompetency(e.currentTarget.value)}
          data={[{ value: '', label: 'не указана' }, ...COMPETENCIES]}
        />
        <Text fz={13} fw={500}>
          Шаги – из них считается прогресс
        </Text>
        {steps.map((s, k) => (
          <div
            key={k}
            style={{
              display: 'grid',
              gridTemplateColumns: '1fr 170px 32px',
              gap: 8,
              alignItems: 'center',
            }}
          >
            <TextInput
              aria-label={`Шаг ${k + 1}`}
              placeholder={`Шаг ${k + 1}`}
              value={s.title}
              onChange={(e) => patchStep(k, { title: e.currentTarget.value })}
            />
            <NativeSelect
              aria-label={`Кто закрывает шаг ${k + 1}`}
              value={s.source}
              onChange={(e) =>
                patchStep(k, { source: e.currentTarget.value as TGoalStep['source'] })
              }
              data={[
                { value: 'task', label: 'задача плана' },
                { value: 'mentor', label: 'наставник' },
                { value: 'lead', label: 'руководитель' },
              ]}
            />
            <ActionIcon
              variant="subtle"
              color="ink"
              aria-label={`Убрать шаг ${k + 1}`}
              onClick={() => setSteps((l) => l.filter((_, j) => j !== k))}
            >
              <IconX size={16} />
            </ActionIcon>
          </div>
        ))}
        <Button
          variant="default"
          size="xs"
          style={{ alignSelf: 'flex-start' }}
          disabled={steps.length >= 8}
          onClick={() => setSteps((l) => [...l, { title: '', source: 'task' }])}
        >
          + Шаг
        </Button>
        {(formError || Boolean(error)) && (
          <Text role="alert" fz={13} style={{ background: '#F7C1C5', padding: '8px 12px' }}>
            {formError ?? errorText(error)}
          </Text>
        )}
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <Button variant="default" onClick={onClose}>
            Отмена
          </Button>
          <Button color="ink" loading={busy} onClick={submit}>
            Сохранить
          </Button>
        </div>
      </div>
    </Modal>
  )
}
