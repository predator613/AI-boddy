import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/calendar'
import type { TAnswerInput, TEntryInput, TMeetingNote } from '../schemas/calendar'

export const calendarKeys = {
  all: ['me', 'calendar'] as const,
  week: (from: string) => ['me', 'calendar', from] as const,
}

export function useCalendar(from: string, to: string) {
  return useQuery({ queryKey: calendarKeys.week(from), queryFn: () => api.getCalendar(from, to) })
}

function useCalendarMutation<A, R>(fn: (a: A) => Promise<R>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: calendarKeys.all })
      // ответ на встречу и перенос видны наставнику в переписке
      qc.invalidateQueries({ queryKey: ['me', 'mentor-thread'] })
    },
  })
}

export const useAddEntry = () => useCalendarMutation((input: TEntryInput) => api.addEntry(input))
export const useDeleteEntry = () => useCalendarMutation((id: string) => api.deleteEntry(id))
export const useAnswerEvent = () =>
  useCalendarMutation(({ id, ...input }: TAnswerInput & { id: string }) =>
    api.answerEvent(id, input),
  )

export const notesKey = ['calendar', 'notes'] as const
export const useNotes = () => useQuery({ queryKey: notesKey, queryFn: api.getNotes })

export function useSaveNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ eventId, text }: { eventId: string; text: string }) =>
      api.saveNote(eventId, text),
    onSuccess: (n) =>
      qc.setQueryData<TMeetingNote[]>(notesKey, (list = []) => [
        ...list.filter((x) => x.eventId !== n.eventId),
        n,
      ]),
  })
}
