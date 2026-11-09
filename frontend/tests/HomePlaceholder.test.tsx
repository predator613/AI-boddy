import { screen } from '@testing-library/react'
import { HomePlaceholder } from '../src/components/HomePlaceholder'
import { renderWithMantine } from './test-utils'

describe('HomePlaceholder', () => {
  it('показывает заголовок', () => {
    renderWithMantine(<HomePlaceholder />)
    expect(screen.getByText('AI-buddy')).toBeInTheDocument()
  })
})
