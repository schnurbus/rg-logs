import { asPlayerClass } from '../lib/classes'
import {
  asPlayerSpec,
  specIconName,
  wowheadIconURL,
} from '../lib/specs'

type SpecIconProps = {
  spec?: string | null
  playerClass?: string | null
  name?: string
  size?: number
  className?: string
}

export function SpecIcon({
  spec,
  playerClass,
  name,
  size = 18,
  className = '',
}: SpecIconProps) {
  const resolvedSpec = asPlayerSpec(spec)
  const resolvedClass = asPlayerClass(playerClass)
  const icon = specIconName(resolvedSpec, resolvedClass)
  const alt = name ? `${name} Spec` : 'Spec'

  return (
    <img
      src={wowheadIconURL(icon)}
      alt={alt}
      width={size}
      height={size}
      className={[
        'inline-block shrink-0 rounded-sm border border-border-subtle bg-surface-raised object-cover',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
      loading="lazy"
      decoding="async"
    />
  )
}
