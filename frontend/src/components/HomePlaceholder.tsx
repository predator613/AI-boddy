import { Container, Title, Text } from '@mantine/core'

export function HomePlaceholder() {
  return (
    <Container size="sm" py="xl">
      <Title order={1}>AI-buddy</Title>
      <Text c="dimmed">Каркас фронтенда работает</Text>
    </Container>
  )
}
