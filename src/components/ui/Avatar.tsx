import { cn } from '@/lib/cn'
import { initials } from '@/lib/format'

interface AvatarProps {
  name: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizes = {
  sm: 'h-8 w-8 text-xs',
  md: 'h-10 w-10 text-sm',
  lg: 'h-12 w-12 text-base',
}

// Deterministic tone per name so avatars feel distinct but stable.
const palettes = [
  'bg-brand-50 text-brand-600',
  'bg-blue-50 text-blue-600',
  'bg-emerald-50 text-emerald-600',
  'bg-amber-50 text-amber-600',
  'bg-violet-50 text-violet-600',
  'bg-rose-50 text-rose-600',
  'bg-cyan-50 text-cyan-600',
]

function paletteFor(name: string): string {
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) >>> 0
  return palettes[hash % palettes.length]
}

export function Avatar({ name, size = 'md', className }: AvatarProps) {
  return (
    <span
      className={cn(
        'inline-flex flex-shrink-0 items-center justify-center rounded-full font-semibold',
        sizes[size],
        paletteFor(name),
        className,
      )}
    >
      {initials(name)}
    </span>
  )
}
