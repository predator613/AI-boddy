// Внешний вид нового «Моего пути»: штрихи кольца, стеклянные карточки этапов, шкала недель.
import { tickColor } from './pathUi'
import type { TStageView } from '../schemas/path'

/** Второй, более насыщенный тон для «стеклянной» карточки этапа. */
const DEEPER: Record<string, string> = {
  '#C9FFBF': '#9DF5A8',
  '#BADAFB': '#8EC7F7',
  '#DFCDF3': '#C3A6EC',
  '#F7C1C5': '#F29AA2',
}

/** Фон карточки этапа: мягкие пятна цвета этапа поверх светлой основы. */
export function glassBackground(color: string): string {
  const a = color.toUpperCase()
  const b = DEEPER[a] ?? a
  return [
    `radial-gradient(70% 55% at 75% 18%, ${a} 0%, transparent 70%)`,
    `radial-gradient(80% 60% at 20% 85%, ${b} 0%, transparent 72%)`,
    'linear-gradient(160deg, rgba(255, 255, 255, 0.95) 0%, rgba(255, 255, 255, 0.55) 100%)',
    a,
  ].join(', ')
}

/** Цвета штрихов кольца: каждой задаче достается поровну штрихов из count. */
export function tickColors(stages: Pick<TStageView, 'tasks'>[], count = 60): string[] {
  const statuses = stages.flatMap((s) => s.tasks.map((t) => t.status))
  if (statuses.length === 0) return Array.from({ length: count }, () => '#E4E4E4')
  return Array.from({ length: count }, (_, i) => {
    const st = statuses[Math.floor((i * statuses.length) / count)]!
    const c = tickColor(st)
    return c === '#EDEDED' ? '#E4E4E4' : c
  })
}

/** Окно из visible карточек этапов так, чтобы выбранный этап был виден. */
export function stageWindow(total: number, selected: number, visible = 4): [number, number] {
  const from = Math.min(Math.max(0, selected - visible + 1), Math.max(0, total - visible))
  return [from, Math.min(total, from + visible)]
}

/** Шкала недель в шапке: прошедшие – зеленые, текущая – синяя и выше, ключевые – темнее. */
export function weekBars(current: number, total: number, milestones: number[]) {
  return Array.from({ length: total }, (_, i) => {
    const w = i + 1
    const key = milestones.includes(w)
    return {
      week: w,
      height: w === current ? 14 : key ? 10 : 6,
      color: w < current ? '#04E061' : w === current ? '#00AAFF' : key ? '#C4C4C4' : '#DDDDDD',
    }
  })
}
