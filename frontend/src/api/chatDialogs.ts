// История диалогов с AI-buddy (docs/api-chat-dialogs.md).
import { ChatReply } from '../schemas/assistant'
import { ChatDialogs } from '../schemas/chatDialogs'
import { apiRequest } from './client'

export const getDialogs = () => apiRequest('/api/chat/dialogs', ChatDialogs)

/** Сообщения прошлого диалога – только для чтения. */
export const getDialogMessages = (id: string) =>
  apiRequest(`/api/chat/dialogs/${encodeURIComponent(id)}/messages`, ChatReply)
