// Хуки ленты изменений наставника и согласований руководителя.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/staff'
import type { TApproval, TDecision, TFeed } from '../schemas/staff'

export const staffKeys = {
  feed: ['mentor', 'feed'] as const,
  approvals: ['approvals'] as const,
}

export const useFeed = () =>
  useQuery({ queryKey: staffKeys.feed, queryFn: api.getFeed, refetchInterval: 30_000 })

export const useMentorMetrics = () =>
  useQuery({ queryKey: ['mentor', 'metrics'], queryFn: api.getMentorMetrics })

export function useMarkFeedRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.markFeedRead,
    onSuccess: () =>
      qc.setQueryData<TFeed>(
        staffKeys.feed,
        (f) => f && { unread: 0, items: f.items.map((i) => ({ ...i, read: true })) },
      ),
  })
}

export const useApprovals = () =>
  useQuery({ queryKey: staffKeys.approvals, queryFn: api.getApprovals })

export function useDecide() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: TDecision & { id: string }) => api.decideApproval(id, input),
    onSuccess: (a) => {
      qc.setQueryData<TApproval[]>(staffKeys.approvals, (list) =>
        list?.map((x) => (x.id === a.id ? a : x)),
      )
      qc.invalidateQueries({ queryKey: ['path'] })
    },
  })
}

export const checklistKey = (internId: string) => ['mentor', 'checklist', internId] as const

export const useChecklist = (internId: string) =>
  useQuery({ queryKey: checklistKey(internId), queryFn: () => api.getChecklist(internId) })

export function useToggleChecklist(internId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ itemId, done }: { itemId: string; done: boolean }) =>
      api.setChecklistItem(internId, itemId, done),
    onSuccess: (c) => qc.setQueryData(checklistKey(internId), c),
  })
}
