import { stageCard, tickColor } from '../../lib/pathUi'
import type { TStageView } from '../../schemas/path'
import { StatusTag } from '../StatusTag'

interface StageCardProps {
  stage: TStageView
  onOpen: (code: string) => void
}

// Карточка этапа на обзоре: номер, название, прогресс-полоски по задачам.
export function StageCard({ stage, onOpen }: StageCardProps) {
  const look = stageCard(stage)
  const todo = stage.status === 'todo'
  return (
    <button
      type="button"
      className="ab-lift"
      onClick={() => onOpen(stage.code)}
      aria-label={`Этап ${stage.code}: ${stage.title}, ${stage.doneCount} из ${stage.totalCount}`}
      style={{
        textAlign: 'left',
        border: 0,
        outline: look.outline,
        outlineOffset: -2,
        background: look.bg,
        padding: 16,
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        minHeight: 156,
        font: 'inherit',
        color: '#141414',
        cursor: 'pointer',
      }}
    >
      <span
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          width: '100%',
        }}
      >
        <span
          style={{
            fontSize: 32,
            fontWeight: 500,
            letterSpacing: '-0.03em',
            lineHeight: 1,
            color: look.numColor,
          }}
        >
          {stage.code}
        </span>
        <StatusTag
          bg={look.tagBg}
          style={{
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            fontSize: 11,
            fontWeight: 600,
          }}
        >
          {look.tag}
        </StatusTag>
      </span>
      <span style={{ fontSize: 16, fontWeight: 500, lineHeight: 1.15, flex: 1 }}>
        {stage.title}
      </span>
      <span style={{ fontSize: 12, color: todo ? '#5C5C5C' : '#141414' }}>
        {stage.doneCount} из {stage.totalCount} задач
      </span>
      <span style={{ display: 'flex', gap: 3, width: '100%' }}>
        {stage.tasks.map((t) => {
          const c = tickColor(t.status)
          return (
            <span
              key={t.id}
              style={{
                flex: 1,
                height: 6,
                background: c === '#EDEDED' ? (todo ? '#F4F4F4' : '#FFFFFF') : c,
              }}
            />
          )
        })}
      </span>
    </button>
  )
}
