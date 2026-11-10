import {
  Alert,
  Button,
  Group,
  Modal,
  NativeSelect,
  NumberInput,
  Stack,
  Text,
  TextInput,
  Textarea,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { zod4Resolver } from 'mantine-form-zod-resolver'
import { z } from 'zod'
import { errorText } from '../../lib/errors'
import { ACTION_LABELS, SWATCHES } from '../../lib/mentorUi'
import type { TTaskTemplate } from '../../schemas/path'

const TaskForm = z.object({
  title: z.string().trim().min(1, 'Введи название').max(120, 'Не длиннее 120 символов'),
  note: z.string().max(300, 'Не длиннее 300 символов'),
  dueDay: z.union([z.number().int().min(1, 'От 1 до 730').max(730, 'От 1 до 730'), z.literal('')]),
  actionKind: z.string(),
})
type TTaskForm = z.infer<typeof TaskForm>

export interface TaskSubmit {
  title: string
  note: string
  dueDay: number | null
  action: TTaskTemplate['action']
}

interface TaskModalProps {
  opened: boolean
  task: TTaskTemplate | null
  stageTitle: string
  busy: boolean
  error: unknown
  onClose: () => void
  onSubmit: (v: TaskSubmit) => void
}

export function TaskModal({
  opened,
  task,
  stageTitle,
  busy,
  error,
  onClose,
  onSubmit,
}: TaskModalProps) {
  const form = useForm<TTaskForm>({
    initialValues: {
      title: task?.title ?? '',
      note: task?.note ?? '',
      dueDay: task?.dueDay ?? '',
      actionKind: task?.action?.kind ?? '',
    },
    validate: zod4Resolver(TaskForm),
  })
  const submit = form.onSubmit((v) => {
    const keep = task?.action && task.action.kind === v.actionKind ? task.action.payload : {}
    onSubmit({
      title: v.title.trim(),
      note: v.note.trim(),
      dueDay: v.dueDay === '' ? null : v.dueDay,
      action: v.actionKind ? { kind: v.actionKind as never, payload: keep } : null,
    })
  })
  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={task ? 'Изменить задачу' : `Новая задача · ${stageTitle}`}
      centered
      radius="lg"
    >
      <form onSubmit={submit} noValidate>
        <Stack gap="sm">
          <TextInput
            label="Название задачи"
            placeholder="Например, «Настроить окружение»"
            data-autofocus
            {...form.getInputProps('title')}
          />
          <Textarea
            label="Пояснение для стажера"
            autosize
            minRows={2}
            {...form.getInputProps('note')}
          />
          <Group grow align="flex-start">
            <NumberInput
              label="День стажировки"
              min={1}
              max={730}
              allowDecimal={false}
              {...form.getInputProps('dueDay')}
            />
            <NativeSelect
              label="Что может сделать AI-buddy"
              data={[
                { value: '', label: 'ничего, только напомнить' },
                ...Object.entries(ACTION_LABELS).map(([value, label]) => ({ value, label })),
              ]}
              {...form.getInputProps('actionKind')}
            />
          </Group>
          <Text size="xs" c="ink.5">
            Изменение сразу появится в «Моем пути» у всех стажеров команды.
          </Text>
          {Boolean(error) && (
            <Alert color="red" variant="light" role="alert">
              {errorText(error)}
            </Alert>
          )}
          <Group justify="flex-end">
            <Button variant="default" onClick={onClose}>
              Отмена
            </Button>
            <Button type="submit" color="ink" loading={busy}>
              Сохранить
            </Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  )
}

const StageForm = z.object({
  title: z.string().trim().min(1, 'Введи название').max(80, 'Не длиннее 80 символов'),
  tip: z.string().max(300, 'Не длиннее 300 символов'),
  color: z.string(),
})
type TStageForm = z.infer<typeof StageForm>

interface StageModalProps {
  opened: boolean
  busy: boolean
  error: unknown
  onClose: () => void
  onSubmit: (v: TStageForm) => void
}

export function StageModal({ opened, busy, error, onClose, onSubmit }: StageModalProps) {
  const form = useForm<TStageForm>({
    initialValues: { title: '', tip: '', color: SWATCHES[0]!.hex },
    validate: zod4Resolver(StageForm),
  })
  return (
    <Modal opened={opened} onClose={onClose} title="Новый этап" centered radius="lg">
      <form onSubmit={form.onSubmit((v) => onSubmit({ ...v, title: v.title.trim() }))} noValidate>
        <Stack gap="sm">
          <TextInput
            label="Название этапа"
            placeholder="Например, «Итоги стажировки»"
            data-autofocus
            {...form.getInputProps('title')}
          />
          <div>
            <Text size="sm" fw={500} mb={6}>
              Цвет карточки
            </Text>
            <Group gap={10}>
              {SWATCHES.map((c) => {
                const on = form.values.color === c.hex
                return (
                  <button
                    key={c.hex}
                    type="button"
                    aria-label={`Цвет ${c.name}`}
                    aria-pressed={on}
                    onClick={() => form.setFieldValue('color', c.hex)}
                    style={{
                      width: 40,
                      height: 40,
                      border: 0,
                      background: c.hex,
                      outline: on ? '2px solid #141414' : 'none',
                      outlineOffset: 2,
                      cursor: 'pointer',
                    }}
                  />
                )
              })}
            </Group>
          </div>
          <Textarea label="Совет для стажера" autosize minRows={2} {...form.getInputProps('tip')} />
          {Boolean(error) && (
            <Alert color="red" variant="light" role="alert">
              {errorText(error)}
            </Alert>
          )}
          <Group justify="flex-end">
            <Button variant="default" onClick={onClose}>
              Отмена
            </Button>
            <Button type="submit" color="ink" loading={busy}>
              Добавить этап
            </Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  )
}
