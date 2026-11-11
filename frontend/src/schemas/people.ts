import { z } from 'zod'

// Контакты для «Поддержки»: имена из связей пользователя в backend (docs/api-chat-requests.md).
const PersonRef = z.object({ id: z.string(), name: z.string() })

export const Department = z.object({
  code: z.string(),
  title: z.string(),
  slaHours: z.number(),
  hours: z.string(),
  about: z.string(),
  items: z.string(),
})

export const Contacts = z.object({
  team: z.string(),
  mentor: PersonRef.nullable(),
  lead: PersonRef.nullable(),
  departments: z.array(Department),
})

export type TContacts = z.infer<typeof Contacts>
export type TDepartment = z.infer<typeof Department>
