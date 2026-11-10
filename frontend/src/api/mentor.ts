// API кабинета наставника и переписки стажера с наставником (docs/api-mentor-console.md).
import { z } from 'zod'
import { ApiError, apiFetch, apiRequest } from './client'
import {
  Automation,
  Material,
  MentorMeeting,
  MentorStats,
  Thread,
  ThreadSummary,
  type TMaterialInput,
  type TMeetingCreateInput,
  type TReplyInput,
} from '../schemas/mentor'

const enc = encodeURIComponent

// ---------- наставник ----------

export const getMentorStats = () => apiRequest('/api/mentor/stats', MentorStats)

export const getThreads = () => apiRequest('/api/mentor/threads', z.array(ThreadSummary))

export const getThread = (internId: string) =>
  apiRequest(`/api/mentor/threads/${enc(internId)}`, Thread)

export const replyToIntern = (internId: string, input: TReplyInput) =>
  apiRequest(`/api/mentor/threads/${enc(internId)}/messages`, Thread, {
    method: 'POST',
    body: input,
  })

export const markThreadRead = (internId: string) =>
  apiFetch(`/api/mentor/threads/${enc(internId)}/read`, { method: 'POST' })

export const getMaterials = () => apiRequest('/api/materials', z.array(Material))

/** POST /api/materials – multipart: файл или ссылка. */
export async function uploadMaterial(input: TMaterialInput) {
  const form = new FormData()
  form.set('title', input.title.trim())
  form.set('stageCode', input.stageCode ?? '')
  form.set('useInAi', String(input.useInAi ?? true))
  form.set('notify', String(input.notify ?? true))
  if (input.file) form.set('file', input.file)
  else form.set('url', (input.url ?? '').trim())
  const parsed = Material.safeParse(
    await apiFetch('/api/materials', { method: 'POST', body: form }),
  )
  if (!parsed.success)
    throw new ApiError(200, 'invalid_response', 'Сервер вернул данные в неожиданном формате')
  return parsed.data
}

export const deleteMaterial = (id: string) =>
  apiFetch(`/api/materials/${enc(id)}`, { method: 'DELETE' })

export const getMeetings = (from: string, to: string) =>
  apiRequest(`/api/mentor/meetings?from=${enc(from)}&to=${enc(to)}`, z.array(MentorMeeting))

export const createMentorMeeting = (input: TMeetingCreateInput) =>
  apiRequest('/api/mentor/meetings', MentorMeeting, { method: 'POST', body: input })

export const moveMeeting = (id: string, startsAt: string) =>
  apiRequest(`/api/mentor/meetings/${enc(id)}`, MentorMeeting, {
    method: 'PATCH',
    body: { startsAt },
  })

export const cancelMeeting = (id: string) =>
  apiFetch(`/api/mentor/meetings/${enc(id)}`, { method: 'DELETE' })

export const getAutomations = () => apiRequest('/api/mentor/automations', z.array(Automation))

export const setAutomation = (id: string, enabled: boolean) =>
  apiRequest(`/api/mentor/automations/${enc(id)}`, Automation, {
    method: 'PATCH',
    body: { enabled },
  })

// ---------- стажер: переписка с наставником ----------

export const getMyMentorThread = () => apiRequest('/api/me/mentor-thread', Thread)

export const sendToMentor = (text: string) =>
  apiRequest('/api/me/mentor-thread/messages', Thread, { method: 'POST', body: { text } })

export const ackUpdate = (messageId: string) =>
  apiRequest(`/api/me/mentor-thread/updates/${enc(messageId)}/ack`, Thread, { method: 'POST' })
