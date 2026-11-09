import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import { MantineProvider } from '@mantine/core'
import { theme } from '../src/theme'

export function renderWithMantine(ui: ReactElement) {
  return render(<MantineProvider theme={theme}>{ui}</MantineProvider>)
}
