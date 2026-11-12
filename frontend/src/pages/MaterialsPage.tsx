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
import { IconSearch, IconTrash, IconUpload } from '@tabler/icons-react'
import { useState } from 'react'
import { Mascot } from '../components/Mascot'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { useDeleteMaterial, useMaterials, useUploadMaterial } from '../hooks/useMentor'
import { useKnowledgeBase } from '../hooks/useKb'
import { useTemplates } from '../hooks/usePath'
import { errorText } from '../lib/errors'
import { MaterialInput, type TMaterial } from '../schemas/mentor'
import { toast } from '../store/toastStore'

const KIND: Record<TMaterial['kind'] | 'article', { label: string; bg: string }> = {
  article: { label: 'СТАТЬЯ', bg: '#BADAFB' },
  pdf: { label: 'PDF', bg: '#C9FFBF' },
  docx: { label: 'DOCX', bg: '#DFCDF3' },
  md: { label: 'СТАТЬЯ', bg: '#BADAFB' },
  link: { label: 'ССЫЛКА', bg: '#F4F4F4' },
  answer: { label: 'ОТВЕТ', bg: '#DFCDF3' },
}

type Row =
  | (TMaterial & { kb: false })
  | {
      id: string
      title: string
      kind: 'article'
      section: string
      stageCode: string | null
      useInAi: boolean
      meta: string
      kb: true
    }

const date = (iso: string) =>
  new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })

// Материалы для стажеров и AI-buddy: статьи, инструкции, ссылки.
export function MaterialsPage() {
  const materials = useMaterials()
  const remove = useDeleteMaterial()
  const kb = useKnowledgeBase()
  const [open, setOpen] = useState(false)
  const [q, setQ] = useState('')
  const [sec, setSec] = useState<string | null>(null)

  // Справочник и загруженные материалы – один список: так наставник видит то же, что стажер.
  const sectionTitle = (id: string) => kb.data?.sections.find((x) => x.id === id)?.title ?? ''
  const rows: Row[] = [
    ...(materials.data ?? []).map((m) => ({ ...m, kb: false as const })),
    ...(kb.data?.articles ?? []).map((a) => ({
      id: a.id,
      title: a.title,
      kind: 'article' as const,
      section: sectionTitle(a.sectionId),
      stageCode: a.stageCode,
      useInAi: true,
      meta: `${a.minutes} мин`,
      kb: true as const,
    })),
  ]
  const sections = [
    ...new Set([
      ...(kb.data?.sections ?? []).map((x) => x.title),
      ...(materials.data ?? []).flatMap((m) => (m.section ? [m.section] : [])),
    ]),
  ]
  const query = q.trim().toLowerCase()
  const shown = rows.filter(
    (r) => (!sec || r.section === sec) && (!query || r.title.toLowerCase().includes(query)),
  )

  return (
    <Page maxWidth={1300}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <PageHeader
          overline="Материалы"
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
            padding: 16,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 16,
          }}
        >
          <Mascot size={40} />
          <span>
            <Text fz={17} fw={500}>
              Загрузи файл или вставь ссылку
            </Text>
            <Text size="sm" c="ink.5">
              PDF, DOCX, Markdown. При загрузке выбери раздел или создай новый
            </Text>
          </span>
        </UnstyledButton>

        {materials.isPending && <LoadingBlocks label="Загрузка материалов" />}
        {materials.isError && (
          <ErrorBlock error={materials.error} onRetry={() => materials.refetch()} />
        )}
        {materials.data && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24, alignItems: 'flex-start' }}>
            <Stack gap="md" style={{ flex: '999 1 520px', minWidth: 0 }}>
              <TextInput
                aria-label="Поиск по материалам"
                placeholder="Найди статью: доступ, ревью, цели…"
                leftSection={<IconSearch size={18} />}
                value={q}
                onChange={(e) => setQ(e.currentTarget.value)}
                size="md"
                radius="md"
              />
              <Paper radius="lg" px="lg" py={4}>
                {shown.length === 0 && (
                  <Text py="md" c="ink.5">
                    Ничего не нашлось. Загрузи материал – AI-buddy начнет на него ссылаться.
                  </Text>
                )}
                {shown.map((m, i) => (
                  <Group
                    key={m.id}
                    wrap="nowrap"
                    gap={14}
                    py={12}
                    style={{ borderTop: i ? '1px solid #EDEDED' : 0 }}
                  >
                    <span
                      style={{
                        width: 52,
                        height: 40,
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
                        {[m.section, m.kb ? m.meta : `${m.authorName} · ${date(m.createdAt)}`]
                          .filter(Boolean)
                          .join(' · ')}
                      </Text>
                    </div>
                    {m.stageCode && <StatusTag bg="#BADAFB">этап {m.stageCode}</StatusTag>}
                    {!m.kb && !m.useInAi && <StatusTag bg="#F4F4F4">только чтение</StatusTag>}
                    {m.kb ? (
                      <span style={{ width: 28 }} />
                    ) : (
                      <ActionIcon
                        variant="subtle"
                        color="ink"
                        aria-label={`Удалить «${m.title}»`}
                        loading={remove.isPending && remove.variables === m.id}
                        onClick={() =>
                          remove.mutate(m.id, {
                            onSuccess: () => toast(`Материал «${m.title}» удален`),
                          })
                        }
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    )}
                  </Group>
                ))}
              </Paper>
            </Stack>
            <nav
              aria-label="Разделы справочника"
              style={{
                flex: '1 1 260px',
                maxWidth: 320,
                display: 'flex',
                flexDirection: 'column',
                gap: 6,
              }}
            >
              <Text fw={600} fz={18} mb={6}>
                Разделы
              </Text>
              {sections.map((t) => {
                const on = sec === t
                return (
                  <UnstyledButton
                    key={t}
                    onClick={() => setSec(on ? null : t)}
                    aria-pressed={on}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      gap: 8,
                      padding: '4px 0',
                    }}
                  >
                    <span
                      style={{
                        fontSize: 16,
                        background: on ? '#00AAFF' : 'transparent',
                        padding: '0 4px',
                        marginLeft: -4,
                      }}
                    >
                      {t}
                    </span>
                    <span style={{ fontSize: 12, color: '#5C5C5C' }}>
                      {rows.filter((r) => r.section === t).length}
                    </span>
                  </UnstyledButton>
                )
              })}
            </nav>
          </div>
        )}
      </Stack>
      {open && <UploadModal onClose={() => setOpen(false)} />}
    </Page>
  )
}

const NEW = '__new__'

function UploadModal({ onClose }: { onClose: () => void }) {
  const upload = useUploadMaterial()
  const templates = useTemplates()
  const [mode, setMode] = useState<'file' | 'link'>('file')
  const [title, setTitle] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [url, setUrl] = useState('')
  const [stageCode, setStageCode] = useState('')
  const kbs = useKnowledgeBase()
  const [section, setSection] = useState('')
  const [newSection, setNewSection] = useState('')
  const [useInAi, setUseInAi] = useState(true)
  const [notify, setNotify] = useState(true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const stages = [...(templates.data ?? [])].sort((a, b) => a.position - b.position)

  const submit = () => {
    const input = {
      title,
      stageCode,
      section: section === NEW ? newSection : section,
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
          label="Раздел справочника"
          value={section}
          onChange={(e) => setSection(e.currentTarget.value)}
          data={[
            { value: '', label: 'без раздела' },
            ...(kbs.data?.sections ?? []).map((x) => ({ value: x.title, label: x.title })),
            { value: NEW, label: '+ Новый раздел' },
          ]}
        />
        {section === NEW && (
          <TextInput
            label="Название нового раздела"
            placeholder="Например, «Дежурства»"
            value={newSection}
            onChange={(e) => setNewSection(e.currentTarget.value)}
            autoFocus
          />
        )}
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
