// Как статусы из API выглядят на макете «Мой путь»: подписи и цвета взяты из дизайна.
// Backend отдает только смысл (status, tagKind), а внешний вид решает фронт – здесь.
import type { TStageView, TTaskView } from '../schemas/path';

export const COLORS = {
  ink: '#141414',
  muted: '#5C5C5C',
  line: '#D6D6D6',
  bg: '#F4F4F4',
  white: '#FFFFFF',
  green: '#04E061',
  blue: '#00AAFF',
  mint: '#C9FFBF',
  sky: '#BADAFB',
  lilac: '#DFCDF3',
  pink: '#F7C1C5',
  ringEmpty: '#EDEDED',
} as const;

type Badge = { label: string; bg: string; border: string; color: string };

/** Бейдж справа от задачи. */
export function taskBadge(t: Pick<TTaskView, 'status' | 'isNext'>): Badge {
  if (t.isNext) return { label: 'следующий шаг', bg: COLORS.green, border: '0', color: COLORS.ink };
  switch (t.status) {
    case 'done':
      return { label: 'готово', bg: COLORS.mint, border: '0', color: COLORS.ink };
    case 'in_progress':
      return { label: 'в работе', bg: COLORS.sky, border: '0', color: COLORS.ink };
    case 'waiting':
      return { label: 'на согласовании', bg: COLORS.sky, border: '0', color: COLORS.ink };
    default:
      return { label: 'впереди', bg: 'transparent', border: `1px solid ${COLORS.line}`, color: COLORS.muted };
  }
}

/** Кружок слева от задачи. */
export function taskDot(status: TTaskView['status']) {
  if (status === 'done') return { bg: COLORS.green, border: '0' };
  if (status === 'todo') return { bg: COLORS.white, border: `1px solid ${COLORS.line}` };
  return { bg: COLORS.white, border: `2px solid ${COLORS.blue}` };
}

/** Цвет деления кольца прогресса и полоски под карточкой этапа. */
export function tickColor(status: TTaskView['status']) {
  if (status === 'done') return COLORS.green;
  if (status === 'todo') return COLORS.ringEmpty;
  return COLORS.blue;
}

/** Карточка этапа на обзоре. */
export function stageCard(s: Pick<TStageView, 'status' | 'color'>) {
  return {
    bg: s.status === 'todo' ? COLORS.white : s.color,
    outline: s.status === 'current' ? `2px solid ${COLORS.ink}` : '0',
    tag: s.status === 'done' ? 'готово' : s.status === 'current' ? 'сейчас' : 'впереди',
    tagBg: s.status === 'done' ? COLORS.green : s.status === 'current' ? COLORS.white : COLORS.bg,
    numColor: s.status === 'todo' ? COLORS.muted : COLORS.ink,
  };
}

/** Тег карточки в боковой панели этапа. */
export function resourceTag(kind: 'ok' | 'wait' | 'none' | 'mentor') {
  switch (kind) {
    case 'ok':
      return { bg: COLORS.mint, border: '0', color: COLORS.ink };
    case 'wait':
      return { bg: COLORS.sky, border: '0', color: COLORS.ink };
    case 'mentor':
      return { bg: COLORS.lilac, border: '0', color: COLORS.ink };
    default:
      return { bg: 'transparent', border: `1px solid ${COLORS.line}`, color: COLORS.muted };
  }
}

/** «срок сегодня», «срок завтра», «через 3 дня», «просрочено на 2 дня». */
export function dueText(days: number | null | undefined): string {
  if (days === null || days === undefined) return '';
  if (days === 0) return 'срок сегодня';
  if (days === 1) return 'срок завтра';
  if (days > 1) return `срок через ${days} ${plural(days, 'день', 'дня', 'дней')}`;
  const n = -days;
  return `просрочено на ${n} ${plural(n, 'день', 'дня', 'дней')}`;
}

/** Встреча: «чт», «12», «15:00», «через 2 дня». */
export function meetingParts(iso: string, now = new Date()) {
  const d = new Date(iso);
  const tz = { timeZone: 'Europe/Moscow' } as const;
  const day = Math.round(
    (Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()) -
      Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())) /
      86_400_000,
  );
  return {
    weekday: d.toLocaleDateString('ru-RU', { ...tz, weekday: 'short' }),
    date: d.toLocaleDateString('ru-RU', { ...tz, day: 'numeric' }),
    time: d.toLocaleTimeString('ru-RU', { ...tz, hour: '2-digit', minute: '2-digit' }),
    inText: day <= 0 ? 'сегодня' : day === 1 ? 'завтра' : `через ${day} ${plural(day, 'день', 'дня', 'дней')}`,
  };
}

/** Сегменты кольца прогресса: по одному на задачу, в порядке этапов. */
export function ringSegments(stages: TStageView[], radius = 42) {
  const all = stages.flatMap((s) => s.tasks.map((t) => t.status));
  const n = Math.max(all.length, 1);
  const slot = 360 / n;
  const c = 2 * Math.PI * radius;
  const arc = (c * (slot - 3)) / 360;
  return all.map((status, i) => ({
    color: tickColor(status),
    dash: `${arc.toFixed(2)} ${c.toFixed(2)}`,
    rotate: -90 + i * slot + 1.5,
  }));
}

function plural(n: number, one: string, few: string, many: string) {
  const m10 = n % 10;
  const m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few;
  return many;
}
