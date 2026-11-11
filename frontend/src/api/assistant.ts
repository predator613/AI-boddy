import { z } from 'zod'
import { apiRequest } from './client'
import {
  ChatReply,
  KbArticleDetail,
  KbSearch,
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

/** GET /api/chat/messages – переписка текущего диалога (восстановить чат после перезагрузки). */
export function getChatHistory() {
  return apiRequest('/api/chat/messages', ChatReply)
}

/** POST /api/chat/reset – «Новый диалог»: старая переписка остается в базе, но не показывается. */
export function resetChat() {
  return apiRequest('/api/chat/reset', ChatReply, { method: 'POST' })
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

/** GET /api/kb/articles/{id} – статья с текстом. Чужой команды – 404. */
export function getKbArticle(articleId: string, signal?: AbortSignal) {
  return apiRequest(`/api/kb/articles/${encodeURIComponent(articleId)}`, KbArticleDetail, {
    signal,
  })
}

/** GET /api/kb/search?q= – поиск по справочнику (до 20 результатов). */
export function searchKb(q: string, signal?: AbortSignal) {
  return apiRequest(`/api/kb/search?q=${encodeURIComponent(q)}`, KbSearch, { signal })
}
