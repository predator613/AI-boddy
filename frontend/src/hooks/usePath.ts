// Хуки TanStack Query для «Моего пути» и консоли наставника.
// Мутации сразу кладут свежий путь в кеш (ответ backend – уже обновленный путь),
// поэтому экран обновляется без повторного запроса.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/path'
import type {
  TMeetingInput,
  TResourceInput,
  TSetStatusInput,
  TStageInput,
  TTaskInput,
} from '../schemas/path'

export const pathKeys = {
  my: ['path', 'me'] as const,
  intern: (id: string) => ['path', 'intern', id] as const,
  interns: ['interns'] as const,
  templates: ['console', 'templates'] as const,
  actionKinds: ['console', 'action-kinds'] as const,
}

// ---------- стажер ----------

export function useMyPath({ enabled = true }: { enabled?: boolean } = {}) {
  return useQuery({ queryKey: pathKeys.my, queryFn: api.getMyPath, enabled })
}

export function useCompleteTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (taskId: string) => api.completeTask(taskId),
    onSuccess: (path) => qc.setQueryData(pathKeys.my, path),
  })
}

// ---------- наставник и руководитель ----------

export function useInterns({ enabled = true }: { enabled?: boolean } = {}) {
  return useQuery({ queryKey: pathKeys.interns, queryFn: api.getInterns, enabled })
}

export function useInternPath(internId: string) {
  return useQuery({
    queryKey: pathKeys.intern(internId),
    queryFn: () => api.getInternPath(internId),
    enabled: Boolean(internId),
  })
}

export function useSetInternTaskStatus(internId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ taskId, ...input }: TSetStatusInput & { taskId: string }) =>
      api.setInternTaskStatus(internId, taskId, input),
    onSuccess: (path) => {
      qc.setQueryData(pathKeys.intern(internId), path)
      qc.invalidateQueries({ queryKey: pathKeys.interns })
    },
  })
}

export function useCreateMeeting(internId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: TMeetingInput) => api.createMeeting(internId, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: pathKeys.intern(internId) }),
  })
}

// ---------- консоль наставника ----------

export function useTemplates() {
  return useQuery({ queryKey: pathKeys.templates, queryFn: api.getTemplates })
}

export function useActionKinds() {
  return useQuery({
    queryKey: pathKeys.actionKinds,
    queryFn: api.getActionKinds,
    staleTime: Infinity,
  })
}

/** После любого изменения шаблона обновляем шаблоны и пути стажеров. */
function useTemplateMutation<TArgs, TResult>(fn: (args: TArgs) => Promise<TResult>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pathKeys.templates })
      qc.invalidateQueries({ queryKey: ['path'] })
      qc.invalidateQueries({ queryKey: pathKeys.interns })
    },
  })
}

export const useCreateStage = () =>
  useTemplateMutation((input: TStageInput) => api.createStage(input))
export const useUpdateStage = () =>
  useTemplateMutation(({ id, ...input }: TStageInput & { id: string; position: number }) =>
    api.updateStage(id, input),
  )
export const useDeleteStage = () => useTemplateMutation((id: string) => api.deleteStage(id))

export const useCreateTask = () =>
  useTemplateMutation(({ stageId, ...input }: TTaskInput & { stageId: string }) =>
    api.createTask(stageId, input),
  )
export const useUpdateTask = () =>
  useTemplateMutation(({ id, ...input }: TTaskInput & { id: string; position: number }) =>
    api.updateTask(id, input),
  )
export const useDeleteTask = () => useTemplateMutation((id: string) => api.deleteTask(id))

export const useCreateResource = () =>
  useTemplateMutation(({ stageId, ...input }: TResourceInput & { stageId: string }) =>
    api.createResource(stageId, input),
  )
export const useDeleteResource = () => useTemplateMutation((id: string) => api.deleteResource(id))
