import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/calendar'
import type { TAnswerInput, TEntryInput } from '../schemas/calendar'

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
