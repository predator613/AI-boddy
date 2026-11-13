// API ленты изменений наставника и согласований руководителя (docs/api-lead-feed.md).
import { z } from 'zod'
import { apiFetch, apiRequest } from './client'
import {
  Approval,
  Decision,
  Feed,
  MentorChecklist,
  MentorMetrics,
  type TDecision,
} from '../schemas/staff'

export const getFeed = () => apiRequest('/api/mentor/feed', Feed)

export const getMentorMetrics = () => apiRequest('/api/mentor/metrics', MentorMetrics)

export const markFeedRead = () => apiFetch('/api/mentor/feed/read', { method: 'POST' })

export const getApprovals = () => apiRequest('/api/approvals', z.array(Approval))

export const decideApproval = (id: string, input: TDecision) =>
  apiRequest(`/api/approvals/${encodeURIComponent(id)}`, Approval, {
    method: 'POST',
    body: Decision.parse(input),
  })

export const getChecklist = (internId: string) =>
  apiRequest(`/api/mentor/interns/${encodeURIComponent(internId)}/checklist`, MentorChecklist)

export const setChecklistItem = (internId: string, itemId: string, done: boolean) =>
  apiRequest(
    `/api/mentor/interns/${encodeURIComponent(internId)}/checklist/${encodeURIComponent(itemId)}`,
    MentorChecklist,
    { method: 'POST', body: { done } },
  )
