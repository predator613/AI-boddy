// Хуки TanStack Query для кабинета наставника и чата стажера с наставником.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/mentor'
import type { TMaterialInput, TMeetingCreateInput, TReplyInput } from '../schemas/mentor'

export const mentorKeys = {
  stats: ['mentor', 'stats'] as const,
  threads: ['mentor', 'threads'] as const,
  thread: (id: string) => ['mentor', 'thread', id] as const,
  materials: ['materials'] as const,
  meetings: (from: string) => ['mentor', 'meetings', from] as const,
  automations: ['mentor', 'automations'] as const,
  myThread: ['me', 'mentor-thread'] as const,
}

export const useMentorStats = () =>
  useQuery({ queryKey: mentorKeys.stats, queryFn: api.getMentorStats })

// Переписку опрашиваем раз в 15 секунд, пока нет push-уведомлений.
const POLL = 15_000

export const useThreads = () =>
  useQuery({ queryKey: mentorKeys.threads, queryFn: api.getThreads, refetchInterval: POLL })

export function useThread(internId: string | null) {
  return useQuery({
    queryKey: mentorKeys.thread(internId ?? ''),
    queryFn: () => api.getThread(internId!),
    enabled: Boolean(internId),
    refetchInterval: POLL,
  })
}

export function useMarkThreadRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.markThreadRead,
    onSuccess: () => qc.invalidateQueries({ queryKey: mentorKeys.threads }),
  })
}

export function useReplyToIntern(internId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: TReplyInput) => api.replyToIntern(internId, input),
    onSuccess: (thread, input) => {
      qc.setQueryData(mentorKeys.thread(internId), thread)
      qc.invalidateQueries({ queryKey: mentorKeys.threads })
      if (input.addToKb) {
        qc.invalidateQueries({ queryKey: mentorKeys.materials })
        qc.invalidateQueries({ queryKey: ['kb'] })
      }
    },
  })
}

export const useMaterials = () =>
  useQuery({ queryKey: mentorKeys.materials, queryFn: api.getMaterials })

export function useUploadMaterial() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: TMaterialInput) => api.uploadMaterial(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: mentorKeys.materials })
      qc.invalidateQueries({ queryKey: ['kb'] })
    },
  })
}

export function useDeleteMaterial() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteMaterial,
    onSuccess: () => qc.invalidateQueries({ queryKey: mentorKeys.materials }),
  })
}

export function useMeetings(from: string, to: string) {
  return useQuery({ queryKey: mentorKeys.meetings(from), queryFn: () => api.getMeetings(from, to) })
}

function useMeetingMutation<A, R>(fn: (args: A) => Promise<R>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mentor', 'meetings'] })
      qc.invalidateQueries({ queryKey: ['path'] })
    },
  })
}

export const useCreateMentorMeeting = () =>
  useMeetingMutation((input: TMeetingCreateInput) => api.createMentorMeeting(input))
export const useMoveMeeting = () =>
  useMeetingMutation(({ id, startsAt }: { id: string; startsAt: string }) =>
    api.moveMeeting(id, startsAt),
  )
export const useCancelMeeting = () => useMeetingMutation((id: string) => api.cancelMeeting(id))

export const useAutomations = () =>
  useQuery({ queryKey: mentorKeys.automations, queryFn: api.getAutomations })

export function useSetAutomation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api.setAutomation(id, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: mentorKeys.automations }),
  })
}

// ---------- стажер ----------

export const useMyMentorThread = () =>
  useQuery({ queryKey: mentorKeys.myThread, queryFn: api.getMyMentorThread, refetchInterval: POLL })

export function useSendToMentor() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (text: string) => api.sendToMentor(text),
    onSuccess: (thread) => qc.setQueryData(mentorKeys.myThread, thread),
  })
}

export function useAckUpdate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (messageId: string) => api.ackUpdate(messageId),
    onSuccess: (thread) => qc.setQueryData(mentorKeys.myThread, thread),
  })
}
