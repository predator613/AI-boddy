import {
  ActionIcon,
  Alert,
  Button,
  Checkbox,
  FileInput,
  Group,
  Modal,
  NativeSelect,
  Paper,
  SegmentedControl,
  Stack,
  Switch,
  Text,
  TextInput,
  UnstyledButton,
} from '@mantine/core'
import { IconTrash, IconUpload } from '@tabler/icons-react'
import { useState } from 'react'
import { Mascot } from '../components/Mascot'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { useDeleteMaterial, useMaterials, useUploadMaterial } from '../hooks/useMentor'
import { useTemplates } from '../hooks/usePath'
import { errorText } from '../lib/errors'
import { MaterialInput, type TMaterial } from '../schemas/mentor'
import { toast } from '../store/toastStore'

const KIND: Record<TMaterial['kind'], { label: string; bg: string }> = {
  pdf: { label: 'PDF', bg: '#C9FFBF' },
  docx: { label: 'DOCX', bg: '#DFCDF3' },
  md: { label: 'СТАТЬЯ', bg: '#BADAFB' },
  link: { label: 'ССЫЛКА', bg: '#F4F4F4' },
  answer: { label: 'ОТВЕТ', bg: '#DFCDF3' },
}

const date = (iso: string) =>
  new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })

// Материалы для стажеров и AI-buddy: статьи, инструкции, ссылки.
export function MaterialsPage() {
  const materials = useMaterials()
  const remove = useDeleteMaterial()
  const [open, setOpen] = useState(false)

  return (
    <Page maxWidth={1100}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <PageHeader
          overline="Материалы · справочник для стажеров и AI-buddy"
          title={
            <>
              Статьи и <Marker color="#00AAFF">инструкции</Marker>
            </>
          }
          right={
            <Button
              color="ink"
              leftSection={<IconUpload size={18} />}
              onClick={() => setOpen(true)}
            >
              Загрузить материал
            </Button>
          }
        />
        <UnstyledButton
          onClick={() => setOpen(true)}
          aria-label="Загрузить материал: выбрать файл"
          style={{
            border: '2px dashed #D6D6D6',
            background: '#FFFFFF',
            borderRadius: 16,
            padding: 24,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 16,
          }}
        >
          <Mascot size={52} />
          <span>
            <Text fz={17} fw={500}>
              Загрузи файл или вставь ссылку
            </Text>
            <Text size="sm" c="ink.5">
              PDF, DOCX, Markdown. AI-buddy прочитает и будет ссылаться на материал в ответах
            </Text>
          </span>
        </UnstyledButton>

        {materials.isPending && <LoadingBlocks label="Загрузка материалов" />}
        {materials.isError && (
          <ErrorBlock error={materials.error} onRetry={() => materials.refetch()} />
        )}
        {materials.data && (
          <Paper radius="lg" px="lg" py={4}>
            {materials.data.length === 0 && (
              <Text py="md" c="ink.5">
                Материалов пока нет
              </Text>
            )}
            {materials.data.map((m, i) => (
              <Group
                key={m.id}
                wrap="nowrap"
                gap={14}
                py={14}
                style={{ borderTop: i ? '1px solid #D6D6D6' : 0 }}
              >
                <span
                  style={{
                    width: 52,
                    height: 44,
                    flex: 'none',
                    background: KIND[m.kind].bg,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 10,
                    fontWeight: 600,
                  }}
                >
                  {KIND[m.kind].label}
                </span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <Text fw={500} truncate>
                    {m.title}
                  </Text>
                  <Text size="xs" c="ink.5">
                    {m.authorName} · {date(m.createdAt)} · {m.sizeLabel}
                  </Text>
                </div>
                {m.stageCode && <StatusTag bg="#BADAFB">этап {m.stageCode}</StatusTag>}
                <StatusTag bg={m.useInAi ? '#C9FFBF' : '#F4F4F4'}>
                  {m.useInAi ? 'AI-buddy использует' : 'только чтение'}
                </StatusTag>
                <ActionIcon
                  variant="subtle"
                  color="ink"
                  aria-label={`Удалить «${m.title}»`}
                  loading={remove.isPending && remove.variables === m.id}
                  onClick={() =>
                    remove.mutate(m.id, { onSuccess: () => toast(`Материал «${m.title}» удален`) })
                  }
                >
                  <IconTrash size={16} />
                </ActionIcon>
              </Group>
            ))}
          </Paper>
        )}
      </Stack>
      {open && <UploadModal onClose={() => setOpen(false)} />}
    </Page>
  )
}

function UploadModal({ onClose }: { onClose: () => void }) {
  const upload = useUploadMaterial()
  const templates = useTemplates()
  const [mode, setMode] = useState<'file' | 'link'>('file')
  const [title, setTitle] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [url, setUrl] = useState('')
  const [stageCode, setStageCode] = useState('')
  const [useInAi, setUseInAi] = useState(true)
  const [notify, setNotify] = useState(true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const stages = [...(templates.data ?? [])].sort((a, b) => a.position - b.position)

  const submit = () => {
    const input = {
      title,
      stageCode,
      useInAi,
      notify,
      url: mode === 'link' ? url : '',
      file: mode === 'file' ? file : null,
    }
    const parsed = MaterialInput.safeParse(input)
    if (!parsed.success) {
      setErrors(Object.fromEntries(parsed.error.issues.map((i) => [String(i.path[0]), i.message])))
      return
    }
    upload.mutate(input, {
      onSuccess: () => {
        toast(notify ? 'Материал загружен. Стажеры получили уведомление' : 'Материал загружен')
        onClose()
      },
    })
  }

  return (
    <Modal opened onClose={onClose} title="Загрузить материал" centered radius="lg">
      <Stack gap="sm">
        <SegmentedControl
          value={mode}
          onChange={(v) => setMode(v as 'file' | 'link')}
          data={[
            { value: 'file', label: 'Загрузить файл' },
            { value: 'link', label: 'Вставить ссылку' },
          ]}
        />
        {mode === 'file' ? (
          <FileInput
            label="Файл"
            placeholder="PDF, DOCX или MD"
            accept=".pdf,.docx,.md,.txt"
            value={file}
            onChange={(f) => {
              setFile(f)
              setErrors({})
              if (f && !title) setTitle(f.name.replace(/\.[^.]+$/, '').replace(/[_-]+/g, ' '))
            }}
            error={errors.file}
            clearable
          />
        ) : (
          <TextInput
            label="Ссылка"
            placeholder="https://…"
            value={url}
            onChange={(e) => (setUrl(e.currentTarget.value), setErrors({}))}
            error={errors.file}
          />
        )}
        <TextInput
          label="Название"
          placeholder="Например, «Как проходит код-ревью»"
          value={title}
          onChange={(e) => (setTitle(e.currentTarget.value), setErrors({}))}
          error={errors.title}
        />
        <NativeSelect
          label="Показывать на этапе"
          value={stageCode}
          onChange={(e) => setStageCode(e.currentTarget.value)}
          data={[
            { value: '', label: 'на всех этапах' },
            ...stages.map((s, i) => ({
              value: String(i + 1).padStart(2, '0'),
              label: `${String(i + 1).padStart(2, '0')} · ${s.title}`,
            })),
          ]}
        />
        <Switch
          color="green.6"
          label="AI-buddy использует материал в ответах"
          checked={useInAi}
          onChange={(e) => setUseInAi(e.currentTarget.checked)}
        />
        <Checkbox
          color="green.6"
          label="Сообщить стажерам о новом материале"
          checked={notify}
          onChange={(e) => setNotify(e.currentTarget.checked)}
        />
        {upload.isError && (
          <Alert color="red" variant="light" role="alert">
            {errorText(upload.error)}
          </Alert>
        )}
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>
            Отмена
          </Button>
          <Button color="ink" loading={upload.isPending} onClick={submit}>
            Загрузить
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}
