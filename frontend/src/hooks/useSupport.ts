import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/support'
import type { TSupportTicket } from '../schemas/support'

const key = ['support', 'tickets'] as const

// Пока есть незакрытые обращения, статусы обновляются сами раз в 15 секунд.
export const useTickets = () =>
  useQuery({
    queryKey: key,
    queryFn: api.getTickets,
    refetchInterval: (q) => (q.state.data?.some((t) => t.status !== 'done') ? 15_000 : false),
  })

export function useCreateTicket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createTicket,
    onSuccess: (t) => qc.setQueryData<TSupportTicket[]>(key, (old) => [t, ...(old ?? [])]),
  })
}
