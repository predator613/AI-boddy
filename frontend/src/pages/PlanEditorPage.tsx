import { ActionIcon, Button, Group, Modal, Paper, SimpleGrid, Stack, Text } from '@mantine/core'
import { IconArrowLeft, IconArrowRight, IconPencil, IconPlus, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { AutomationsPanel } from '../components/mentor/AutomationsPanel'
import { StageModal, TaskModal, type TaskSubmit } from '../components/mentor/PlanForms'
import { ACTION_LABELS, tasksWord as plural } from '../lib/mentorUi'
import {
  useCreateStage,
  useCreateTask,
  useDeleteStage,
  useDeleteTask,
  useTemplates,
  useUpdateStage,
  useUpdateTask,
} from '../hooks/usePath'
import type { TStageTemplate, TTaskTemplate } from '../schemas/path'
import { toast } from '../store/toastStore'

const SAVED = 'Сохранено. Изменение уже в «Моем пути» у стажеров команды'

type TaskDialog = { stage: TStageTemplate; task: TTaskTemplate | null } | null

// План адаптации команды: этапы и задачи. Работает с настоящим backend (/api/console/*).
export function PlanEditorPage() {
  const templates = useTemplates()
  const createStage = useCreateStage()
  const updateStage = useUpdateStage()
  const deleteStage = useDeleteStage()
  const createTask = useCreateTask()
  const updateTask = useUpdateTask()
  const deleteTask = useDeleteTask()
  const [taskDialog, setTaskDialog] = useState<TaskDialog>(null)
  const [stageOpen, setStageOpen] = useState(false)
  const [stageToDelete, setStageToDelete] = useState<TStageTemplate | null>(null)

  const stages = [...(templates.data ?? [])].sort((a, b) => a.position - b.position)
  const taskSaving = createTask.isPending || updateTask.isPending
  const taskError = createTask.error ?? updateTask.error

  const saveTask = (v: TaskSubmit) => {
    if (!taskDialog) return
    const done = { onSuccess: () => (setTaskDialog(null), toast(SAVED)) }
    if (taskDialog.task)
      updateTask.mutate({ id: taskDialog.task.id, position: taskDialog.task.position, ...v }, done)
    else createTask.mutate({ stageId: taskDialog.stage.id, ...v }, done)
  }

  const move = (s: TStageTemplate, dir: -1 | 1) =>
    updateStage.mutate(
      {
        id: s.id,
        title: s.title,
        color: s.color,
        tip: s.tip,
        suggestions: s.suggestions,
        position: s.position + dir,
      },
      { onSuccess: () => toast(`Этап «${s.title}» перемещен`) },
    )

  const openTask = (d: TaskDialog) => {
    createTask.reset()
    updateTask.reset()
    setTaskDialog(d)
  }

  return (
    <Page maxWidth={1400}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <PageHeader
          overline="План адаптации · шаблон твоей команды"
          title={
            <>
              Этапы и <Marker>задачи</Marker>
            </>
          }
          right={
            <Button
              color="ink"
              leftSection={<IconPlus size={18} />}
              onClick={() => (createStage.reset(), setStageOpen(true))}
            >
              Этап
            </Button>
          }
        />
        {templates.isPending && <LoadingBlocks label="Загрузка плана" />}
        {templates.isError && (
          <ErrorBlock error={templates.error} onRetry={() => templates.refetch()} />
        )}
        {templates.data && (
          <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="md" style={{ alignItems: 'start' }}>
            {stages.map((s, i) => {
              const tasks = [...(s.tasks ?? [])].sort(
                (a, b) => (a.dueDay ?? 999) - (b.dueDay ?? 999) || a.position - b.position,
              )
              return (
                <section
                  key={s.id}
                  aria-label={`Этап ${s.title}`}
                  className="ab-rise"
                  style={{
                    background: s.color,
                    padding: 14,
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 10,
                  }}
                >
                  <Group justify="space-between" wrap="nowrap">
                    <Text fz={28} fw={500} lh={1} style={{ letterSpacing: '-0.03em' }}>
                      {String(i + 1).padStart(2, '0')}
                    </Text>
                    <Group gap={2} wrap="nowrap">
                      <ActionIcon
                        variant="subtle"
                        color="ink"
                        aria-label={`Сдвинуть «${s.title}» левее`}
                        disabled={i === 0}
                        onClick={() => move(s, -1)}
                      >
                        <IconArrowLeft size={16} />
                      </ActionIcon>
                      <ActionIcon
                        variant="subtle"
                        color="ink"
                        aria-label={`Сдвинуть «${s.title}» правее`}
                        disabled={i === stages.length - 1}
                        onClick={() => move(s, 1)}
                      >
                        <IconArrowRight size={16} />
                      </ActionIcon>
                      <ActionIcon
                        variant="subtle"
                        color="ink"
                        aria-label={`Удалить этап «${s.title}»`}
                        onClick={() => setStageToDelete(s)}
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    </Group>
                  </Group>
                  <Group justify="space-between" align="baseline" wrap="nowrap">
                    <Text fz={17} fw={500} lh={1.15}>
                      {s.title}
                    </Text>
                    <Text size="xs" style={{ whiteSpace: 'nowrap' }}>
                      {plural(tasks.length)}
                    </Text>
                  </Group>
                  {tasks.map((t) => (
                    <Stack key={t.id} gap={6} p="sm" style={{ background: '#FFFFFF' }}>
                      <Group gap={4} wrap="nowrap" align="flex-start">
                        <Text fz={14} fw={500} lh={1.25} style={{ flex: 1 }}>
                          {t.title}
                        </Text>
                        <ActionIcon
                          size="sm"
                          variant="subtle"
                          color="ink"
                          aria-label={`Изменить задачу «${t.title}»`}
                          onClick={() => openTask({ stage: s, task: t })}
                        >
                          <IconPencil size={15} />
                        </ActionIcon>
                        <ActionIcon
                          size="sm"
                          variant="subtle"
                          color="ink"
                          aria-label={`Удалить задачу «${t.title}»`}
                          loading={deleteTask.isPending && deleteTask.variables === t.id}
                          onClick={() =>
                            deleteTask.mutate(t.id, {
                              onSuccess: () => toast(`Задача «${t.title}» удалена`),
                            })
                          }
                        >
                          <IconTrash size={15} />
                        </ActionIcon>
                      </Group>
                      <Group gap={6}>
                        {t.dueDay && (
                          <Text size="11px" c="ink.5">
                            день {t.dueDay}
                          </Text>
                        )}
                        {t.action && (
                          <StatusTag bg="#F4F4F4" style={{ fontSize: 11, padding: '1px 6px' }}>
                            AI-buddy {ACTION_LABELS[t.action.kind]}
                          </StatusTag>
                        )}
                      </Group>
                    </Stack>
                  ))}
                  <Button
                    variant="outline"
                    color="ink"
                    size="sm"
                    style={{ borderStyle: 'dashed' }}
                    leftSection={<IconPlus size={16} />}
                    onClick={() => openTask({ stage: s, task: null })}
                  >
                    Задача
                  </Button>
                </section>
              )
            })}
          </SimpleGrid>
        )}
        {templates.data && stages.length === 0 && (
          <Paper radius="lg" p="lg">
            <Text>
              В шаблоне пока нет этапов. Добавь первый – он сразу появится у стажеров команды.
            </Text>
          </Paper>
        )}
        <AutomationsPanel />
      </Stack>

      {taskDialog && (
        <TaskModal
          opened
          task={taskDialog.task}
          stageTitle={taskDialog.stage.title}
          busy={taskSaving}
          error={taskError}
          onClose={() => setTaskDialog(null)}
          onSubmit={saveTask}
        />
      )}
      {stageOpen && (
        <StageModal
          opened
          busy={createStage.isPending}
          error={createStage.error}
          onClose={() => setStageOpen(false)}
          onSubmit={(v) =>
            createStage.mutate(v, {
              onSuccess: () => (setStageOpen(false), toast(`Добавлен этап «${v.title}»`)),
            })
          }
        />
      )}
      <Modal
        opened={stageToDelete !== null}
        onClose={() => setStageToDelete(null)}
        title="Удалить этап?"
        centered
        radius="lg"
      >
        <Stack gap="md">
          <Text>
            Этап «{stageToDelete?.title}» и {plural(stageToDelete?.tasks?.length ?? 0)} пропадут из
            пути у всех стажеров команды.
          </Text>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setStageToDelete(null)}>
              Отмена
            </Button>
            <Button
              color="red"
              loading={deleteStage.isPending}
              onClick={() =>
                stageToDelete &&
                deleteStage.mutate(stageToDelete.id, {
                  onSuccess: () => {
                    toast(`Этап «${stageToDelete.title}» удален`)
                    setStageToDelete(null)
                  },
                })
              }
            >
              Удалить
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Page>
  )
}
