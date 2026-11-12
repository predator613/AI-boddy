// Короткие подписи дат для интерфейса.
const MONTHS = ['янв', 'фев', 'мар', 'апр', 'мая', 'июн', 'июл', 'авг', 'сен', 'окт', 'нояб', 'дек']

/** «2026-11-10» → «10 нояб». */
export function shortDate(iso: string | null): string {
  if (!iso) return 'без срока'
  const [, m, d] = iso.slice(0, 10).split('-').map(Number)
  return m && d ? `${d} ${MONTHS[m - 1]}` : iso
}
