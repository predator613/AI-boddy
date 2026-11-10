import { z } from 'zod'
import { apiRequest } from './client'
import {
  ChatReply,
  KnowledgeBase,
  RequestItem,
  SendMessageInput,
  type TSendMessageInput,
} from '../schemas/assistant'

/** POST /api/chat/messages – отправить сообщение помощнику (можно с действием из задачи). */
export function sendMessage(input: TSendMessageInput) {
  return apiRequest('/api/chat/messages', ChatReply, {
    method: 'POST',
    body: SendMessageInput.parse(input),
  })
}

/** POST /api/chat/actions/{id}/confirm – подтвердить действие (например, создать заявку). */
export function confirmAction(actionId: string) {
  return apiRequest(`/api/chat/actions/${encodeURIComponent(actionId)}/confirm`, ChatReply, {
    method: 'POST',
  })
}

/** POST /api/chat/actions/{id}/cancel – отказаться от действия. */
export function cancelAction(actionId: string) {
  return apiRequest(`/api/chat/actions/${encodeURIComponent(actionId)}/cancel`, ChatReply, {
    method: 'POST',
  })
}

/** GET /api/requests – мои заявки, новые сверху. */
export function getRequests() {
  return apiRequest('/api/requests', z.array(RequestItem))
}

/** GET /api/kb – разделы, статьи и словарик справочника. */
export function getKnowledgeBase() {
  return apiRequest('/api/kb', KnowledgeBase)
}
