import type { ReactNode } from 'react'
import classes from './Page.module.css'

// Отступы страницы и раскладка «основная колонка + боковая панель».
export function Page({ children, maxWidth }: { children: ReactNode; maxWidth?: number }) {
  return (
    <div className={classes.page} style={maxWidth ? { maxWidth } : undefined}>
      {children}
    </div>
  )
}
