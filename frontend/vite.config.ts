/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { tanstackRouter, tanstackRouterGenerator } from '@tanstack/router-plugin/vite'

const isTest = Boolean(process.env.VITEST)

export default defineConfig({
  plugins: [
    // Плагин роутера собирает src/routeTree.gen.ts из файлов в src/routes и должен стоять ПЕРЕД react().
    // В тестах только генерация дерева, без служебного кода горячей перезагрузки, чтобы покрытие считалось честно.
    isTest
      ? tanstackRouterGenerator({ target: 'react' })
      : tanstackRouter({ target: 'react', autoCodeSplitting: true }),
    react(),
  ],
  server: { port: 5173 },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./tests/setup.ts'],
    // на медленных компьютерах длинные сценарии не укладываются в 5 секунд по умолчанию
    testTimeout: 30000,
    css: false,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/routeTree.gen.ts', 'src/main.tsx', 'src/mocks/**', 'src/**/*.d.ts'],
      thresholds: { lines: 80 },
    },
  },
})
