import { render, screen } from '@testing-library/react'
import { MantineProvider } from '@mantine/core'
import {
  dueText,
  meetingParts,
  resourceTag,
  stageCard,
  taskBadge,
  taskDot,
  tickColor,
} from '../src/lib/pathUi'
import { Mascot } from '../src/components/Mascot'
import { StageDetail } from '../src/components/path/StageDetail'
import { errorText } from '../src/lib/errors'
import { ApiError } from '../src/api/client'
import { buildMockPath } from '../src/mocks/pathData'
import { theme } from '../src/theme'

describe('lib/pathUi', () => {
  it('сроки', () => {
    expect(dueText(null)).toBe('')
    expect(dueText(0)).toBe('срок сегодня')
    expect(dueText(1)).toBe('срок завтра')
    expect(dueText(3)).toBe('срок через 3 дня')
    expect(dueText(5)).toBe('срок через 5 дней')
    expect(dueText(21)).toBe('срок через 21 день')
    expect(dueText(-1)).toBe('просрочено на 1 день')
    expect(dueText(-12)).toBe('просрочено на 12 дней')
  })

  it('встреча', () => {
    const now = new Date('2026-11-10T09:00:00+03:00')
    expect(meetingParts('2026-11-12T15:00:00+03:00', now)).toMatchObject({
      date: '12',
      time: '15:00',
      inText: 'через 2 дня',
    })
    expect(meetingParts('2026-11-10T15:00:00+03:00', now).inText).toBe('сегодня')
    expect(meetingParts('2026-11-11T15:00:00+03:00', now).inText).toBe('завтра')
  })

  it('статусы, теги и этапы', () => {
    expect(taskBadge({ status: 'done', isNext: false }).label).toBe('готово')
    expect(taskBadge({ status: 'in_progress', isNext: false }).label).toBe('в работе')
    expect(taskBadge({ status: 'waiting', isNext: false }).label).toBe('на согласовании')
    expect(taskBadge({ status: 'todo', isNext: false }).label).toBe('впереди')
    expect(taskBadge({ status: 'todo', isNext: true }).label).toBe('следующий шаг')
    expect(taskDot('waiting').border).toContain('#00AAFF')
    expect(tickColor('todo')).toBe('#EDEDED')
    expect(resourceTag('ok').bg).toBe('#C9FFBF')
    expect(resourceTag('wait').bg).toBe('#BADAFB')
    expect(resourceTag('mentor').bg).toBe('#DFCDF3')
    expect(resourceTag('none').bg).toBe('transparent')
    expect(stageCard({ status: 'done', color: '#C9FFBF' }).tag).toBe('готово')
    expect(stageCard({ status: 'todo', color: '#C9FFBF' }).bg).toBe('#FFFFFF')
  })

  it('тексты ошибок', () => {
    expect(errorText(new ApiError(0, 'network_error', 'Сервер недоступен'))).toBe(
      'Сервер недоступен',
    )
    expect(errorText(new Error('x'))).toMatch(/Не удалось/)
  })
})

describe('компоненты', () => {
  const wrap = (ui: React.ReactNode) =>
    render(<MantineProvider theme={theme}>{ui}</MantineProvider>)

  it('маскот во всех состояниях', () => {
    const { container } = wrap(
      <>
        {(['calm', 'happy', 'think', 'mentor', 'sad'] as const).map((m) => (
          <Mascot key={m} mood={m} label="AI-buddy" glow />
        ))}
      </>,
    )
    expect(container.querySelectorAll('svg[data-mood]')).toHaveLength(5)
    expect(screen.getByRole('img', { name: 'AI-buddy (думает)' })).toBeInTheDocument()
  })

  it('заголовок боковой панели зависит от вида карточек', () => {
    const stages = buildMockPath().stages
    wrap(<StageDetail stage={stages[2]} />)
    expect(screen.getByText('Инструменты')).toBeInTheDocument()
  })

  it('материалы и этап без карточек', () => {
    const stages = buildMockPath().stages
    wrap(
      <>
        <StageDetail stage={stages[3]} />
        <StageDetail stage={{ ...stages[0], resources: [], tip: '' }} />
        <StageDetail
          stage={{
            ...stages[0],
            resources: [{ id: 'x', kind: 'link', name: 'Чат', note: '', tag: '', tagKind: 'none' }],
          }}
        />
      </>,
    )
    expect(screen.getByText('Материалы')).toBeInTheDocument()
    expect(screen.getByText('Полезное')).toBeInTheDocument()
  })
})
