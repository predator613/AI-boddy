import { useId } from 'react'

export type Mood = 'calm' | 'happy' | 'think' | 'mentor' | 'sad'

const PALETTE: Record<Mood, [string, string, string, string]> = {
  calm: ['#9DFFB9', '#04E061', '#02B84F', 'спокоен'],
  happy: ['#9DFFB9', '#04E061', '#02B84F', 'рад'],
  think: ['#9EDCFF', '#00AAFF', '#0088CC', 'думает'],
  mentor: ['#CDB3F5', '#965EEB', '#7442C8', 'зовет наставника'],
  sad: ['#FF9BA5', '#FF4053', '#D92A3C', 'грустит'],
}

interface MascotProps {
  mood?: Mood
  size?: number
  glow?: boolean
  float?: boolean
  /** Подпись для скринридера. Без нее маскот считается украшением. */
  label?: string
}

// Маскот AI-buddy: цвет тела показывает состояние помощника.
export function Mascot({
  mood = 'calm',
  size = 120,
  glow = false,
  float = false,
  label,
}: MascotProps) {
  const gid = useId().replace(/:/g, '')
  const [light, base, dark, moodText] = PALETTE[mood]
  const think = mood === 'think'
  const eyeY = think ? 47 : mood === 'sad' ? 56 : 52
  const eyeRy = mood === 'sad' ? 8 : 10.5
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      className={float ? 'ab-float' : undefined}
      role={label ? 'img' : undefined}
      aria-label={label ? `${label} (${moodText})` : undefined}
      aria-hidden={label ? undefined : true}
      data-mood={mood}
      style={{
        flex: 'none',
        filter: glow ? `drop-shadow(0 0 ${Math.round(size / 8)}px ${base}73)` : undefined,
      }}
    >
      <defs>
        <radialGradient id={gid} cx="40%" cy="35%" r="70%">
          <stop offset="0" stopColor={light} />
          <stop offset="0.55" stopColor={base} />
          <stop offset="1" stopColor={dark} />
        </radialGradient>
      </defs>
      <circle cx="16" cy="22" r="8" fill={base} />
      <circle cx="86" cy="86" r="6" fill={base} />
      <circle cx="50" cy="54" r="32" fill={`url(#${gid})`} />
      {mood === 'happy' ? (
        <>
          <path
            d="M33 55 Q40 44 47 55"
            fill="none"
            stroke="#FFFFFF"
            strokeWidth="5"
            strokeLinecap="round"
          />
          <path
            d="M53 55 Q60 44 67 55"
            fill="none"
            stroke="#FFFFFF"
            strokeWidth="5"
            strokeLinecap="round"
          />
        </>
      ) : (
        <>
          <ellipse cx={think ? 43 : 40} cy={eyeY} rx="6.5" ry={eyeRy} fill="#FFFFFF" />
          <ellipse cx={think ? 63 : 60} cy={eyeY} rx="6.5" ry={eyeRy} fill="#FFFFFF" />
        </>
      )}
      {mood === 'sad' && (
        <>
          <path d="M32 38 L46 43" stroke="#FFFFFF" strokeWidth="3.5" strokeLinecap="round" />
          <path d="M68 38 L54 43" stroke="#FFFFFF" strokeWidth="3.5" strokeLinecap="round" />
        </>
      )}
      {think && (
        <>
          <circle cx="78" cy="22" r="3" fill={base} />
          <circle cx="88" cy="14" r="4" fill={base} />
        </>
      )}
    </svg>
  )
}
