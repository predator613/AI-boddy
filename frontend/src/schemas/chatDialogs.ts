// Прошлые диалоги с AI-buddy для колонки истории в чате. Контракт – docs/api-chat-dialogs.md.
import { z } from 'zod'

export const ChatDialog = z.object({
  id: z.string(),
  title: z.string(), // первая реплика стажера, обрезанная до 60 символов
  preview: z.string(), // итог: «Заявка №1042 создана», «Передано наставнику», «Ответ из справочника»
  kind: z.enum(['request', 'handoff', 'answer']),
  updatedAt: z.string(),
  current: z.boolean(), // текущий открытый диалог
})

export const ChatDialogs = z.array(ChatDialog)
export type TChatDialog = z.infer<typeof ChatDialog>
