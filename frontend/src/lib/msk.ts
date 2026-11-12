// Время в календаре считается по Москве: рабочее окно 10–18 и сетка не сдвигаются
// у стажера из другого часового пояса (backend проверяет тоже по Europe/Moscow).
const FMT = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Europe/Moscow',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  weekday: 'short',
  hourCycle: 'h23',
})
const WD: Record<string, number> = { Mon: 0, Tue: 1, Wed: 2, Thu: 3, Fri: 4, Sat: 5, Sun: 6 }

export interface MskParts {
  ymd: string
  hour: number
  minute: number
  /** 0 – понедельник, 6 – воскресенье */
  weekday: number
}

export function msk(at: Date | string): MskParts {
  const p = Object.fromEntries(FMT.formatToParts(new Date(at)).map((x) => [x.type, x.value]))
  return {
    ymd: `${p.year}-${p.month}-${p.day}`,
    hour: Number(p.hour),
    minute: Number(p.minute),
    weekday: WD[p.weekday ?? 'Mon'] ?? 0,
  }
}

/** Календарная дата дня сетки (локальная дата объекта) в виде «YYYY-MM-DD». */
export const ymd = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

/** Начало записи в указанный день сетки и час (10.5 = 10:30) по Москве. */
export function mskIso(day: Date, hours: number) {
  const hh = String(Math.floor(hours)).padStart(2, '0')
  const mm = hours % 1 ? '30' : '00'
  return `${ymd(day)}T${hh}:${mm}:00+03:00`
}
