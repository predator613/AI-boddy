import { Alert, Button, PasswordInput, Stack, TextInput } from '@mantine/core'
import { useForm } from '@mantine/form'
import { zod4Resolver } from 'mantine-form-zod-resolver'
import { LoginRequestSchema, type LoginRequest } from '../schemas/auth'
import { loginErrorText, useLogin } from '../hooks/useLogin'

interface LoginFormProps {
  onSuccess?: () => void
}

export function LoginForm({ onSuccess }: LoginFormProps) {
  const mutation = useLogin(onSuccess)
  const form = useForm<LoginRequest>({
    mode: 'uncontrolled',
    initialValues: { login: '', password: '' },
    validate: zod4Resolver(LoginRequestSchema),
  })

  return (
    <form
      noValidate
      onSubmit={form.onSubmit((values) => mutation.mutate(LoginRequestSchema.parse(values)))}
    >
      <Stack gap="sm">
        <TextInput
          label="Логин"
          placeholder="например, intern_anna"
          autoComplete="username"
          key={form.key('login')}
          {...form.getInputProps('login')}
        />
        <PasswordInput
          label="Пароль"
          autoComplete="current-password"
          key={form.key('password')}
          {...form.getInputProps('password')}
        />
        {mutation.isError && (
          <Alert color="red" variant="light" role="alert">
            {loginErrorText(mutation.error)}
          </Alert>
        )}
        <Button type="submit" loading={mutation.isPending} fullWidth mt="xs">
          Войти
        </Button>
      </Stack>
    </form>
  )
}
