import { createTheme, type MantineColorsTuple } from '@mantine/core'

// Палитра из PDF кейса (справочник «AI-buddy – палитра и шрифты кейса»).
// Основной оттенок каждого цвета стоит на индексе 6 – его берут кнопки и бейджи.
const green: MantineColorsTuple = [
  '#EBFFE8',
  '#D9FFD2',
  '#C9FFBF',
  '#9DFFB9',
  '#5CF08E',
  '#2BE673',
  '#04E061',
  '#02C754',
  '#02B84F',
  '#018F3D',
]
const blue: MantineColorsTuple = [
  '#E8F4FE',
  '#D4EAFC',
  '#BADAFB',
  '#8FCBFF',
  '#5CBBFF',
  '#2AB0FF',
  '#00AAFF',
  '#0095E0',
  '#0088CC',
  '#006BA3',
]
const purple: MantineColorsTuple = [
  '#F4EEFC',
  '#EADFF8',
  '#DFCDF3',
  '#CDB3F5',
  '#B892F0',
  '#A577ED',
  '#965EEB',
  '#8448DE',
  '#7442C8',
  '#5B329E',
]
const red: MantineColorsTuple = [
  '#FFF0F1',
  '#FCDCDF',
  '#F7C1C5',
  '#FF9BA5',
  '#FF7A87',
  '#FF5C6C',
  '#FF4053',
  '#E83346',
  '#D92A3C',
  '#B01F2F',
]
const ink: MantineColorsTuple = [
  '#F4F4F4',
  '#EDEDED',
  '#D6D6D6',
  '#B8B8B8',
  '#8A8A8A',
  '#5C5C5C',
  '#141414',
  '#101010',
  '#0B0B0B',
  '#000000',
]

export const C = {
  ink: '#141414',
  muted: '#5C5C5C',
  line: '#D6D6D6',
  bg: '#F4F4F4',
  white: '#FFFFFF',
  green: '#04E061',
  blue: '#00AAFF',
  purple: '#965EEB',
  red: '#FF4053',
  mint: '#C9FFBF',
  sky: '#BADAFB',
  lilac: '#DFCDF3',
  pink: '#F7C1C5',
} as const

export const theme = createTheme({
  primaryColor: 'green',
  primaryShade: 6,
  autoContrast: true,
  luminanceThreshold: 0.45,
  colors: { green, blue, purple, red, ink },
  black: C.ink,
  defaultRadius: 'md',
  fontFamily: 'Onest, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif',
  headings: {
    fontFamily: 'Onest, system-ui, sans-serif',
    fontWeight: '500',
    sizes: {
      h1: { fontSize: '40px', lineHeight: '1.05' },
      h2: { fontSize: '20px', lineHeight: '1.2' },
      h3: { fontSize: '16px', lineHeight: '1.25' },
    },
  },
  components: {
    Button: { defaultProps: { fw: 500 } },
    Title: { styles: { root: { letterSpacing: '-0.02em' } } },
  },
})
