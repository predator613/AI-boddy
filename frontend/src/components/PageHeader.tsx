import { Group, Stack, Text, Title } from '@mantine/core'
import type { ReactNode } from 'react'

interface PageHeaderProps {
  overline: string
  title: ReactNode
  right?: ReactNode
}

export function PageHeader({ overline, title, right }: PageHeaderProps) {
  return (
    <Group justify="space-between" align="flex-end" gap="md" wrap="wrap">
      <Stack gap={8}>
        <Text size="11px" tt="uppercase" c="ink.5" style={{ letterSpacing: '0.06em' }}>
          {overline}
        </Text>
        <Title order={1}>{title}</Title>
      </Stack>
      {right}
    </Group>
  )
}

/** Выделение маркером, как ключевые фразы в кейсе. */
export function Marker({ color = '#04E061', children }: { color?: string; children: ReactNode }) {
  return (
    <span className="ab-marker" style={{ background: color }}>
      {children}
    </span>
  )
}
