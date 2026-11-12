import { Text, UnstyledButton } from '@mantine/core'
import { IconArrowRight } from '@tabler/icons-react'
import { glassBackground, stageWindow } from '../../lib/pathLook'
import { tickColor } from '../../lib/pathUi'
import type { TStageView } from '../../schemas/path'
import classes from './StageCarousel.module.css'

const VISIBLE = 4

interface StageCarouselProps {
  stages: TStageView[]
  /** Раскрытая карточка (этап, задачи которого показаны ниже, или текущий). */
  selectedCode: string
  onOpen: (code: string) => void
}

const STATE = {
  done: { tag: 'готово', dot: '#04E061' },
  current: { tag: 'сейчас', dot: '#00AAFF' },
  todo: { tag: 'впереди', dot: '#FFFFFF' },
} as const

// Этапы адаптации: «стеклянные» карточки, выбранная раскрыта шире. Если наставник добавит
// больше четырех этапов, стрелка покажет остальные (ряд сдвигается за выбранным этапом).
export function StageCarousel({ stages, selectedCode, onOpen }: StageCarouselProps) {
  const idx = Math.max(
    0,
    stages.findIndex((s) => s.code === selectedCode),
  )
  const [from, to] = stageWindow(stages.length, idx, VISIBLE)
  const next = stages[(idx + 1) % Math.max(stages.length, 1)]

  return (
    <section aria-label="Этапы адаптации" className={`${classes.wrap} ab-rise-2`}>
      <div className={classes.head}>
        <Text component="h2" fz={18} fw={500} m={0}>
          Этапы адаптации
        </Text>
        <Text size="xs" c="ink.5">
          {stages.length > VISIBLE
            ? 'стрелка покажет остальные этапы'
            : 'нажми на этап, чтобы раскрыть'}
        </Text>
      </div>
      <div className={classes.row}>
        {stages.slice(from, to).map((s) => {
          const open = s.code === selectedCode
          const look = STATE[s.status]
          return (
            <div
              key={s.id}
              className={classes.card}
              data-open={open || undefined}
              style={{
                flexGrow: open ? 4 : 1,
                background: glassBackground(s.color),
                border: s.status === 'current' ? '2px solid #141414' : 0,
              }}
            >
              <UnstyledButton
                className={classes.hit}
                onClick={() => onOpen(s.code)}
                aria-expanded={open}
                aria-label={`Этап ${s.code}: ${s.title}, ${s.doneCount} из ${s.totalCount}`}
              />
              <span className={classes.top}>
                <span className={classes.num} style={{ fontSize: open ? 44 : 28 }}>
                  {s.code}
                </span>
                {open ? (
                  <span className={`${classes.tag} ab-pop`}>{look.tag}</span>
                ) : (
                  <span className={classes.dot} style={{ background: look.dot }} />
                )}
              </span>
              {open ? (
                <span className={`${classes.body} ab-pop`}>
                  <span className={classes.title}>{s.title}</span>
                  <span className={classes.hint}>{s.period ?? s.tip}</span>
                  <span style={{ flex: 1 }} />
                  <span style={{ fontSize: 12 }}>
                    {s.doneCount} из {s.totalCount} ·{' '}
                    {look.tag === 'сейчас' ? 'в работе' : look.tag}
                  </span>
                  <span className={classes.bars}>
                    {s.tasks.map((t) => (
                      <span
                        key={t.id}
                        style={{
                          background:
                            t.status === 'todo' ? 'rgba(255,255,255,0.7)' : tickColor(t.status),
                        }}
                      />
                    ))}
                  </span>
                </span>
              ) : (
                <>
                  <span style={{ flex: 1 }} />
                  <span className={classes.vertical}>{s.title}</span>
                </>
              )}
            </div>
          )
        })}
        {stages.length > 1 && next && (
          <div className={classes.nav}>
            <UnstyledButton
              className={classes.next}
              onClick={() => onOpen(next.code)}
              aria-label={`Следующий этап: ${next.title}`}
            >
              <IconArrowRight size={20} />
            </UnstyledButton>
            <Text size="11px" c="ink.5">
              {idx + 1} / {stages.length}
            </Text>
          </div>
        )}
      </div>
    </section>
  )
}
