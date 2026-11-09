import { z } from 'zod'

export const RoleSchema = z.enum(['intern', 'mentor', 'lead'])

export const UserSchema = z.object({
  id: z.string(),
  login: z.string(),
  name: z.string(),
  role: RoleSchema,
  teamId: z.string(),
  teamName: z.string(),
  mentorId: z.string().nullable(),
})

export const LoginRequestSchema = z.object({
  login: z.string().trim().min(1, 'Введите логин'),
  password: z.string().min(1, 'Введите пароль'),
})

export const LoginResponseSchema = z.object({
  token: z.string().min(1),
  user: UserSchema,
})

export const TestUserSchema = z.object({
  login: z.string(),
  name: z.string(),
  role: RoleSchema,
  teamName: z.string(),
})

export const TestUsersSchema = z.array(TestUserSchema)

export const ApiErrorSchema = z.object({
  error: z.object({
    code: z.string(),
    message: z.string(),
  }),
})

export type Role = z.infer<typeof RoleSchema>
export type User = z.infer<typeof UserSchema>
export type LoginRequest = z.infer<typeof LoginRequestSchema>
export type LoginResponse = z.infer<typeof LoginResponseSchema>
export type TestUser = z.infer<typeof TestUserSchema>

export const roleLabels: Record<Role, string> = {
  intern: 'Стажер',
  mentor: 'Наставник',
  lead: 'Руководитель',
}
