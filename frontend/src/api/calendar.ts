// API календаря стажера (docs/api-intern-calendar.md).
import { apiFetch, apiRequest } from './client'
import {
  CalendarEvent,
  CalendarView,
  type TAnswerInput,
  type TEntryInput,
} from '../schemas/calendar'

const enc = encodeURIComponent

export const getCalendar = (from: string, to: string) =>
  apiRequest(`/api/me/calendar?from=${enc(from)}&to=${enc(to)}`, CalendarView)

export const addEntry = (input: TEntryInput) =>
  apiRequest('/api/me/calendar/entries', CalendarEvent, { method: 'POST', body: input })

export const deleteEntry = (id: string) =>
  apiFetch(`/api/me/calendar/entries/${enc(id)}`, { method: 'DELETE' })

export const answerEvent = (id: string, input: TAnswerInput) =>
  apiRequest(`/api/me/calendar/events/${enc(id)}/answer`, CalendarEvent, {
    method: 'POST',
    body: input,
  })
