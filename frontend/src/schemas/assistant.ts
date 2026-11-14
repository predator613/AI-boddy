// Схемы чата, заявок и справочника. Контракты: docs/api-chat-requests.md, docs/api-goals-kb.md.
import { z } from 'zod'
import { Action } from './path'

export const Field = z.object({ label: z.string(), value: z.string() })

// Заявка во внешнем сервисе заявок (Jira Service Desk): ключ и ссылка на карточку.
// null – заявки там нет (отгул, демо-данные); поля нет – старый backend, ключ собирается по номеру.
export const ExternalRef = z
  .object({ system: z.string(), key: z.string(), url: z.string().nullable() })
  .nullish()

// cancelled – стажер отозвал заявку, пока она была на согласовании.
export const RequestStatus = z.enum(['waiting', 'approved', 'done', 'rejected', 'cancelled'])

export const RequestItem = z.object({
  id: z.string(),
  number: z.string(),
  kind: z.enum(['access', 'leave', 'other']),
  title: z.string(),
  status: RequestStatus,
  createdAt: z.string(),
  via: z.enum(['buddy', 'auto', 'manual']),
  fields: z.array(Field),
  history: z.array(
    z.object({ title: z.string(), at: z.string(), state: z.enum(['done', 'now', 'rejected']) }),
  ),
  rejectReason: z.string().nullable(),
  external: ExternalRef,
})

export const ChatMessage = z.object({
  id: z.string(),
  role: z.enum(['user', 'assistant']),
  kind: z.enum(['text', 'confirm', 'result', 'error']),
  text: z.string(),
  source: z.object({ title: z.string(), articleId: z.string().nullable() }).nullable(),
  confirm: z
    .object({
      actionId: z.string(),
      title: z.string(),
      fields: z.array(Field),
      state: z.enum(['pending', 'confirmed', 'cancelled']),
    })
    .nullable(),
  result: z
    .object({
      requestId: z.string(),
      number: z.string(),
      title: z.string(),
      status: RequestStatus,
      external: ExternalRef,
    })
    .nullable(),
  mood: z.enum(['calm', 'happy', 'think', 'mentor', 'sad']),
})

export const ChatReply = z.object({ messages: z.array(ChatMessage) })

export const SendMessageInput = z.object({
  text: z.string().trim().min(1, 'Напиши сообщение').max(2000),
  action: Action.nullable().default(null),
  // Задача «Моего пути», из которой пришел вопрос: backend закроет ее, когда заявка выполнится.
  taskId: z.string().nullable().optional(),
  // «Как отвечать» из профиля: помощник подстраивает длину ответа.
  style: z.enum(['short', 'normal', 'detailed']).optional(),
})

export const KbSection = z.object({
  id: z.string(),
  title: z.string(),
  hand: z.string(),
  color: z.string(),
  count: z.number(),
})
export const KbArticle = z.object({
  id: z.string(),
  sectionId: z.string(),
  title: z.string(),
  summary: z.string(),
  minutes: z.number(),
  stageCode: z.string().nullable(),
})
export const KbTerm = z.object({ term: z.string(), def: z.string() })
// GET /api/kb/articles/{id} – статья целиком. Абзацы в body разделены пустой строкой.
export const KbArticleDetail = KbArticle.extend({
  body: z.string(),
  source: z.enum(['seed', 'upload', 'link', 'mentor_answer']),
  url: z.string().nullable(),
  authorName: z.string().nullable(),
  useInAi: z.boolean(),
  updatedAt: z.string(),
})
// GET /api/kb/search?q= – поиск с учетом русской морфологии, лучший фрагмент каждой статьи.
export const KbSearchMatch = z.object({
  articleId: z.string(),
  sectionId: z.string(),
  title: z.string(),
  summary: z.string(),
  snippet: z.string(),
  stageCode: z.string().nullable(),
  score: z.number(),
})
export const KbSearch = z.object({ query: z.string(), results: z.array(KbSearchMatch) })
export const KnowledgeBase = z.object({
  sections: z.array(KbSection),
  articles: z.array(KbArticle),
  terms: z.array(KbTerm),
})

export type TRequestItem = z.infer<typeof RequestItem>
export type TRequestStatus = z.infer<typeof RequestStatus>
export type TChatMessage = z.infer<typeof ChatMessage>
export type TChatReply = z.infer<typeof ChatReply>
export type TSendMessageInput = z.input<typeof SendMessageInput>
export type TKnowledgeBase = z.infer<typeof KnowledgeBase>
export type TKbArticle = z.infer<typeof KbArticle>
export type TKbArticleDetail = z.infer<typeof KbArticleDetail>
export type TKbSearch = z.infer<typeof KbSearch>
